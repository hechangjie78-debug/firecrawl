package services

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/firecrawl/go-api/models"
	"github.com/rs/zerolog/log"
)

// CrawlService 支持高并发分布式 Redis + RabbitMQ 的异步全站爬虫服务管理器
type CrawlService struct {
	scrapeService   *ScrapeService
	redisService    *RedisService
	rabbitmqService *RabbitMQService

	// 备用单机内存兜底（当 Redis/RabbitMQ 不可用时无缝降级使用）
	jobsLock sync.RWMutex
	jobs     map[string]*models.CrawlJob
}

// NewCrawlService 初始化创建 CrawlService 实例并挂载 Redis 与 RabbitMQ 扩展组件
func NewCrawlService(scrapeService *ScrapeService, redisService *RedisService, rabbitmqService *RabbitMQService) *CrawlService {
	s := &CrawlService{
		scrapeService:   scrapeService,
		redisService:    redisService,
		rabbitmqService: rabbitmqService,
		jobs:            make(map[string]*models.CrawlJob),
	}

	// 如果 RabbitMQ 可用，启动后台 Consumer 消费管道
	if rabbitmqService != nil {
		err := rabbitmqService.StartConsumer(func(jobID string) {
			s.executeCrawlFromQueue(jobID)
		})
		if err != nil {
			log.Warn().Err(err).Msg("启动 RabbitMQ 异步爬虫消费者失败，将在单机模式下运行")
		}
	}

	return s
}

// CreateCrawlJob 初始化并启动一个后台异步全站爬取任务
func (s *CrawlService) CreateCrawlJob(ctx context.Context, req *models.CrawlRequest, teamID string, crawlID string) *models.CrawlJob {
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	maxDepth := req.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 2
	}

	cctx, cancel := context.WithCancel(context.Background())

	job := &models.CrawlJob{
		ID:                 crawlID,
		TeamID:             teamID,
		URL:                req.URL,
		Status:             models.StatusScraping,
		Limit:              limit,
		MaxDepth:           maxDepth,
		AllowBackwardLinks: req.AllowBackwardLinks,
		AllowExternalLinks: req.AllowExternalLinks,
		ScrapeOptions:      req.ScrapeOptions,
		Documents:          make([]*models.Document, 0),
		Errors:             make([]*models.CrawlErrorItem, 0),
		RobotsBlocked:      make([]string, 0),
		VisitedURLs:        make(map[string]bool),
		CreatedAt:          time.Now(),
		ExpiresAt:          time.Now().Add(24 * time.Hour),
		CancelFunc:         cancel,
	}

	// 1. 优先使用 Redis 进行分布式状态持久化
	if s.redisService != nil {
		if err := s.redisService.SaveCrawlJob(ctx, job); err != nil {
			log.Error().Err(err).Str("crawlID", crawlID).Msg("写入 Redis 失败，退回内存保存")
		}
	}

	// 单机内存兜底
	s.jobsLock.Lock()
	s.jobs[crawlID] = job
	s.jobsLock.Unlock()

	// 2. 优先使用 RabbitMQ 消息队列进行削峰推送
	if s.rabbitmqService != nil {
		if err := s.rabbitmqService.PublishCrawlJob(ctx, crawlID); err == nil {
			return job
		}
		log.Warn().Str("crawlID", crawlID).Msg("推送到 RabbitMQ 失败，降级使用 Go 内置 Goroutine 执行抓取")
	}

	// 3. 降级方案：启动后台 Goroutine 执行调度
	go s.runCrawlWorker(cctx, job, req)

	return job
}

// GetCrawlJob 从 Redis 或内存读取任务状态与结果
func (s *CrawlService) GetCrawlJob(crawlID string) (*models.CrawlJob, bool) {
	// 优先查询 Redis
	if s.redisService != nil {
		job, err := s.redisService.GetCrawlJob(context.Background(), crawlID)
		if err == nil && job != nil {
			return job, true
		}
	}

	// 内存兜底
	s.jobsLock.RLock()
	defer s.jobsLock.RUnlock()
	job, exists := s.jobs[crawlID]
	return job, exists
}

// GetOngoingCrawls 获取正在进行的爬取任务列表
func (s *CrawlService) GetOngoingCrawls(teamID string) []*models.OngoingCrawlItem {
	s.jobsLock.RLock()
	defer s.jobsLock.RUnlock()

	crawls := make([]*models.OngoingCrawlItem, 0)
	for _, job := range s.jobs {
		if (teamID == "" || job.TeamID == teamID) && job.Status == models.StatusScraping {
			crawls = append(crawls, &models.OngoingCrawlItem{
				ID:        job.ID,
				TeamID:    job.TeamID,
				URL:       job.URL,
				CreatedAt: job.CreatedAt.Format(time.RFC3339),
				Options: &models.CrawlRequest{
					URL:                job.URL,
					Limit:              job.Limit,
					MaxDepth:           job.MaxDepth,
					AllowBackwardLinks: job.AllowBackwardLinks,
					AllowExternalLinks: job.AllowExternalLinks,
					ScrapeOptions:      job.ScrapeOptions,
				},
			})
		}
	}
	return crawls
}

// CancelCrawlJob 取消正在运行的异步爬虫任务
func (s *CrawlService) CancelCrawlJob(crawlID string) bool {
	// 更新 Redis
	if s.redisService != nil {
		_ = s.redisService.UpdateJobStatus(context.Background(), crawlID, models.StatusCancelled)
	}

	// 更新内存
	s.jobsLock.Lock()
	defer s.jobsLock.Unlock()
	job, exists := s.jobs[crawlID]
	if exists {
		job.Status = models.StatusCancelled
		if job.CancelFunc != nil {
			job.CancelFunc()
		}
	}

	log.Info().Str("crawlID", crawlID).Msg("用户已主动取消异步爬虫任务")
	return true
}

// executeCrawlFromQueue 处理 RabbitMQ 弹出的异步爬虫 Task 消息
func (s *CrawlService) executeCrawlFromQueue(jobID string) {
	job, exists := s.GetCrawlJob(jobID)
	if !exists || job == nil {
		log.Warn().Str("jobID", jobID).Msg("收到 RabbitMQ 消息但未找到对应 Job，忽略")
		return
	}

	if job.Status == models.StatusCancelled {
		log.Info().Str("jobID", jobID).Msg("任务已被用户提前取消，忽略执行")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	job.CancelFunc = cancel

	s.runCrawlWorker(ctx, job, &models.CrawlRequest{
		URL:                job.URL,
		Limit:              job.Limit,
		MaxDepth:           job.MaxDepth,
		AllowBackwardLinks: job.AllowBackwardLinks,
		AllowExternalLinks: job.AllowExternalLinks,
		ScrapeOptions:      job.ScrapeOptions,
	})
}

// runCrawlWorker 核心并发爬取工作例程 (广度优先算法 BFS 提取子链接)
func (s *CrawlService) runCrawlWorker(ctx context.Context, job *models.CrawlJob, req *models.CrawlRequest) {
	defer func() {
		if job.CancelFunc != nil {
			job.CancelFunc()
		}
	}()

	log.Info().Str("crawlID", job.ID).Str("startURL", job.URL).Int("limit", job.Limit).Msg("后台并发爬虫协程池启动")

	type queueItem struct {
		url   string
		depth int
	}

	queue := []queueItem{{url: job.URL, depth: 1}}
	job.VisitedURLs[job.URL] = true

	baseURL, err := url.Parse(job.URL)
	if err != nil {
		log.Error().Err(err).Str("crawlID", job.ID).Msg("解析初始起始 URL 失败")
		job.Status = models.StatusFailed
		if s.redisService != nil {
			_ = s.redisService.SaveCrawlJob(ctx, job)
		}
		return
	}

	for len(queue) > 0 && len(job.Documents) < job.Limit {
		select {
		case <-ctx.Done():
			log.Info().Str("crawlID", job.ID).Msg("爬虫任务收到取消指令中断退出")
			return
		default:
		}

		// 检查 Redis 中最新状态，如已被修改为 cancelled 则及时退出
		if s.redisService != nil {
			latestJob, err := s.redisService.GetCrawlJob(ctx, job.ID)
			if err == nil && latestJob != nil && latestJob.Status == models.StatusCancelled {
				log.Info().Str("crawlID", job.ID).Msg("检测到任务被设置为 Cancelled，提前中断抓取")
				return
			}
		}

		curr := queue[0]
		queue = queue[1:]

		if curr.depth > job.MaxDepth {
			continue
		}

		scrapeReq := job.ScrapeOptions
		scrapeReq.URL = curr.url
		if len(scrapeReq.Formats) == 0 {
			scrapeReq.Formats = []string{"markdown"}
		}

		doc, err := s.scrapeService.ExecuteScrape(ctx, &scrapeReq, job.ID+"-"+curr.url)
		if err != nil {
			log.Warn().Err(err).Str("url", curr.url).Msg("单页面抓取失败，记录错误并跳过")
			job.Errors = append(job.Errors, &models.CrawlErrorItem{
				ID:        job.ID + "-" + curr.url,
				URL:       curr.url,
				Error:     err.Error(),
				Timestamp: time.Now().Format(time.RFC3339),
			})
			continue
		}

		// 实时持久化文档到 Redis
		job.Documents = append(job.Documents, doc)
		if s.redisService != nil {
			_ = s.redisService.AppendDocument(ctx, job.ID, doc)
		}

		// 提取超链接
		if curr.depth < job.MaxDepth && len(job.Documents) < job.Limit {
			foundLinks := extractLinks(doc.RawHTML, doc.Markdown, baseURL, job.AllowExternalLinks)
			for _, link := range foundLinks {
				if !job.VisitedURLs[link] {
					job.VisitedURLs[link] = true
					queue = append(queue, queueItem{url: link, depth: curr.depth + 1})
				}
			}
		}
	}

	// 最终更新状态
	if job.Status == models.StatusScraping {
		job.Status = models.StatusCompleted
	}

	if s.redisService != nil {
		_ = s.redisService.SaveCrawlJob(ctx, job)
	}

	log.Info().Str("crawlID", job.ID).Int("totalCompleted", len(job.Documents)).Msg("全站/多页面异步爬取任务顺利完成！")
}

// extractLinks 提取网页 DOM 或文本/Markdown 中的所有 Valid href 子链接
func extractLinks(rawHTML string, markdown string, baseURL *url.URL, allowExternal bool) []string {
	links := make(map[string]bool)

	// 1. 正则匹配 HTML 的 href="..."
	hrefRegex := regexp.MustCompile(`href=["']([^"']+)["']`)
	matches := hrefRegex.FindAllStringSubmatch(rawHTML, -1)
	for _, m := range matches {
		if len(m) >= 2 {
			addValidLink(m[1], baseURL, allowExternal, links)
		}
	}

	// 2. 正则匹配 Markdown 的 [title](https://...)
	mdLinkRegex := regexp.MustCompile(`\[.*?\]\((https?://[^\s\)]+)\)`)
	mdMatches := mdLinkRegex.FindAllStringSubmatch(rawHTML+" "+markdown, -1)
	for _, m := range mdMatches {
		if len(m) >= 2 {
			addValidLink(m[1], baseURL, allowExternal, links)
		}
	}

	result := make([]string, 0, len(links))
	for l := range links {
		result = append(result, l)
	}
	return result
}

func addValidLink(rawLink string, baseURL *url.URL, allowExternal bool, links map[string]bool) {
	rawLink = strings.TrimSpace(rawLink)
	if rawLink == "" || strings.HasPrefix(rawLink, "#") || strings.HasPrefix(rawLink, "javascript:") || strings.HasPrefix(rawLink, "mailto:") {
		return
	}

	parsed, err := url.Parse(rawLink)
	if err != nil {
		return
	}

	resolved := baseURL.ResolveReference(parsed)
	if !allowExternal && !strings.HasSuffix(resolved.Host, baseURL.Host) && resolved.Host != baseURL.Host {
		return
	}

	resolved.Fragment = ""
	links[resolved.String()] = true
}
