package services

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/firecrawl/go-api/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// BatchService 专门负责批量抓取与 Sitemap 扫描业务的服务层
type BatchService struct {
	scrapeService   *ScrapeService
	redisService    *RedisService
	rabbitmqService *RabbitMQService
	httpClient      *http.Client

	// 内存双保险
	jobsLock sync.RWMutex
	jobs     map[string]*models.BatchScrapeJob
}

// NewBatchService 初始化创建 BatchService 实例
func NewBatchService(scrapeService *ScrapeService, redisService *RedisService, rabbitmqService *RabbitMQService) *BatchService {
	return &BatchService{
		scrapeService:   scrapeService,
		redisService:    redisService,
		rabbitmqService: rabbitmqService,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		jobs: make(map[string]*models.BatchScrapeJob),
	}
}

// CreateBatchScrape 提交批量抓取任务并分配唯一 Batch Job ID (1ms 极速入队)
func (s *BatchService) CreateBatchScrape(ctx context.Context, req *models.BatchScrapeRequest, teamID string) (*models.BatchScrapeJob, error) {
	batchID := uuid.New().String()
	total := len(req.URLs)
	if total == 0 {
		return nil, fmt.Errorf("urls list cannot be empty")
	}

	job := &models.BatchScrapeJob{
		ID:        batchID,
		TeamID:    teamID,
		Status:    models.StatusScraping,
		Completed: 0,
		Total:     total,
		Documents: make([]*models.Document, 0),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	// 1. 存入 Redis 持久化
	if s.redisService != nil {
		_ = s.redisService.SaveCrawlJob(ctx, &models.CrawlJob{
			ID:        batchID,
			TeamID:    teamID,
			Status:    models.StatusScraping,
			Limit:     total,
			Documents: make([]*models.Document, 0),
			CreatedAt: job.CreatedAt,
			ExpiresAt: job.ExpiresAt,
		})
	}

	// 2. 存入内存结构
	s.jobsLock.Lock()
	s.jobs[batchID] = job
	s.jobsLock.Unlock()

	// 3. 启动后台协程池进行批量并发抓取
	go s.processBatchWorker(job, req)

	return job, nil
}

// processBatchWorker 批量后台并发调度提取 Worker
func (s *BatchService) processBatchWorker(job *models.BatchScrapeJob, req *models.BatchScrapeRequest) {
	log.Info().Str("batchID", job.ID).Int("total", job.Total).Msg("后台批量并发抓取任务池启动")

	semaphore := make(chan struct{}, 10)
	var wg sync.WaitGroup

	for _, targetURL := range req.URLs {
		semaphore <- struct{}{}
		wg.Add(1)

		go func(u string) {
			defer func() {
				<-semaphore
				wg.Done()
			}()

			scrapeOpt := req.Options
			scrapeOpt.URL = u

			doc, err := s.scrapeService.ExecuteScrape(context.Background(), &scrapeOpt, job.ID+"-"+u)
			if err != nil {
				log.Warn().Err(err).Str("url", u).Msg("批量抓取单个页面失败")
				s.jobsLock.Lock()
				job.Errors = append(job.Errors, &models.CrawlErrorItem{
					ID:        job.ID + "-" + u,
					URL:       u,
					Error:     err.Error(),
					Timestamp: time.Now().Format(time.RFC3339),
				})
				s.jobsLock.Unlock()
				return
			}

			s.jobsLock.Lock()
			job.Documents = append(job.Documents, doc)
			job.Completed = len(job.Documents)
			s.jobsLock.Unlock()

			if s.redisService != nil {
				_ = s.redisService.AppendDocument(context.Background(), job.ID, doc)
			}
		}(targetURL)
	}

	wg.Wait()
	s.jobsLock.Lock()
	job.Status = models.StatusCompleted
	s.jobsLock.Unlock()

	if s.redisService != nil {
		_ = s.redisService.UpdateJobStatus(context.Background(), job.ID, models.StatusCompleted)
	}
	log.Info().Str("batchID", job.ID).Int("completed", job.Completed).Msg("批量抓取任务顺利完成！")
}

// GetBatchJob 获取批量任务进度与提取文档
func (s *BatchService) GetBatchJob(batchID string) (*models.BatchScrapeJob, bool) {
	if s.redisService != nil {
		crawlJob, err := s.redisService.GetCrawlJob(context.Background(), batchID)
		if err == nil && crawlJob != nil {
			return &models.BatchScrapeJob{
				ID:        crawlJob.ID,
				TeamID:    crawlJob.TeamID,
				Status:    crawlJob.Status,
				Completed: len(crawlJob.Documents),
				Total:     crawlJob.Limit,
				Documents: crawlJob.Documents,
				CreatedAt: crawlJob.CreatedAt,
				ExpiresAt: crawlJob.ExpiresAt,
			}, true
		}
	}

	s.jobsLock.RLock()
	defer s.jobsLock.RUnlock()
	job, exists := s.jobs[batchID]
	return job, exists
}

// CancelBatchJob 取消正在执行的批量任务
func (s *BatchService) CancelBatchJob(batchID string) bool {
	if s.redisService != nil {
		_ = s.redisService.UpdateJobStatus(context.Background(), batchID, models.StatusCancelled)
	}

	s.jobsLock.Lock()
	defer s.jobsLock.Unlock()
	job, exists := s.jobs[batchID]
	if exists {
		job.Status = models.StatusCancelled
		return true
	}
	return true
}

// Sitemap 对应 xml 结构的节点
type urlSetXML struct {
	URLs []struct {
		Loc string `xml:"loc"`
	} `xml:"url"`
}

// ExecuteMap 极速扫描并生成目标站点的 Sitemap 链接列表 (POST /v1/map)
func (s *BatchService) ExecuteMap(ctx context.Context, req *models.MapRequest) ([]string, error) {
	parsedURL, err := url.Parse(req.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL: %w", err)
	}

	limit := req.Limit
	// 如果用户未传 limit 参数 (即 limit <= 0)，设为 100000 提取扫描到的全部链接
	maxCount := 100000
	if limit > 0 {
		maxCount = limit
	}

	linkMap := make(map[string]bool)

	// 1. 尝试快速从 sitemap.xml 或 sitemap_index.xml 提取
	if !req.IgnoreSitemap {
		sitemapCandidates := []string{
			fmt.Sprintf("%s://%s/sitemap.xml", parsedURL.Scheme, parsedURL.Host),
			fmt.Sprintf("%s://%s/sitemap_index.xml", parsedURL.Scheme, parsedURL.Host),
		}

		for _, smURL := range sitemapCandidates {
			linksFromSitemap := s.fetchLinksFromSitemap(ctx, smURL)
			for _, l := range linksFromSitemap {
				linkMap[l] = true
			}
			if limit > 0 && len(linkMap) >= limit {
				break
			}
		}
	}

	// 2. 如果 sitemap 提取不够，尝试从首页发起网页抓取分析 DOM 超链接
	if limit <= 0 || len(linkMap) < limit {
		homeDoc, _, err := s.scrapeService.fastFetchHTML(ctx, req.URL)
		if err == nil && len(homeDoc) > 100 {
			extracted := extractLinks(homeDoc, homeDoc, parsedURL, req.IncludeSubdomains)
			for _, l := range extracted {
				linkMap[l] = true
			}
		}

		// 3. 如果 Fast-Fetch 依然没拿够链接，自动降级调用 ExecuteScrape 拿到完整 DOM HTML
		if limit <= 0 || len(linkMap) < limit {
			scrapeDoc, err := s.scrapeService.ExecuteScrape(ctx, &models.ScrapeRequest{
				URL: req.URL,
			}, "map-fetch-"+req.URL)
			if err == nil && (scrapeDoc.RawHTML != "" || scrapeDoc.Markdown != "") {
				extracted := extractLinks(scrapeDoc.RawHTML, scrapeDoc.Markdown, parsedURL, req.IncludeSubdomains)
				for _, l := range extracted {
					linkMap[l] = true
				}
			}
		}
	}

	result := make([]string, 0, len(linkMap))
	for l := range linkMap {
		if req.Search != "" && !strings.Contains(strings.ToLower(l), strings.ToLower(req.Search)) {
			continue
		}
		result = append(result, l)
		if len(result) >= maxCount {
			break
		}
	}

	log.Info().Str("targetURL", req.URL).Int("totalLinks", len(result)).Msg("POST /v1/map 站点地图结构扫描顺利完成")
	return result, nil
}

// fetchLinksFromSitemap 极速拉取并 XML 解包 sitemap.xml
func (s *BatchService) fetchLinksFromSitemap(ctx context.Context, sitemapURL string) []string {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", sitemapURL, nil)
	if err != nil {
		return nil
	}
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil || resp.StatusCode != 200 {
		return nil
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var xmlData urlSetXML
	if err := xml.Unmarshal(bodyBytes, &xmlData); err == nil && len(xmlData.URLs) > 0 {
		links := make([]string, 0, len(xmlData.URLs))
		for _, u := range xmlData.URLs {
			loc := strings.TrimSpace(u.Loc)
			if loc != "" {
				links = append(links, loc)
			}
		}
		return links
	}

	locRegex := regexp.MustCompile(`<loc>(.*?)</loc>`)
	matches := locRegex.FindAllStringSubmatch(string(bodyBytes), -1)
	links := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) >= 2 {
			links = append(links, strings.TrimSpace(m[1]))
		}
	}
	return links
}
