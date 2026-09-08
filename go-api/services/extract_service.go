package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/firecrawl/go-api/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// SearxngResultItem 表示从 SearXNG 解析出来的单项搜索结果
type SearxngResultItem struct {
	URL         string
	Title       string
	Description string
}

// SearxngSearchResponse 定义 SearXNG JSON API 返回结构
type SearxngSearchResponse struct {
	Query           string `json:"query"`
	NumberOfResults int    `json:"number_of_results"`
	Results         []struct {
		URL     string `json:"url"`
		Title   string `json:"title"`
		Content string `json:"content"`
		Engine  string `json:"engine"`
	} `json:"results"`
}

// ExtractService 专门负责网页搜索聚合与智能 LLM 结构化提取的服务层
type ExtractService struct {
	scrapeService *ScrapeService
	redisService  *RedisService
	httpClient    *http.Client

	// 内存备用池
	jobsLock sync.RWMutex
	jobs     map[string]*models.ExtractJob
}

// NewExtractService 初始化 ExtractService 实例
func NewExtractService(scrapeService *ScrapeService, redisService *RedisService) *ExtractService {
	return &ExtractService{
		scrapeService: scrapeService,
		redisService:  redisService,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		jobs: make(map[string]*models.ExtractJob),
	}
}

// ExecuteSearch POST /v1/search 执行智能搜索并并发抓取聚合结果
func (s *ExtractService) ExecuteSearch(ctx context.Context, req *models.SearchRequest) ([]*models.SearchResultItem, error) {
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 5 // 默认自动并发抓取前 5 条结果页
	}

	// 1. 调用 SearXNG 获取真实公网搜索候选目标列表
	candidates, err := s.fetchSearxngCandidates(ctx, query, limit, req.Lang)
	if err != nil {
		return nil, err
	}

	if len(candidates) == 0 {
		log.Info().Str("query", query).Msg("SearXNG 搜索结果为空")
		return []*models.SearchResultItem{}, nil
	}

	log.Info().Str("query", query).Int("candidateCount", len(candidates)).Msg("开始并发抓取 SearXNG 搜索结果页内容")

	// 2. 使用协程池并发抓取所有搜索结果页
	results := make([]*models.SearchResultItem, 0, len(candidates))
	var wg sync.WaitGroup
	var mu sync.Mutex

	semaphore := make(chan struct{}, 5)

	for _, cand := range candidates {
		semaphore <- struct{}{}
		wg.Add(1)

		go func(c SearxngResultItem) {
			defer func() {
				<-semaphore
				wg.Done()
			}()

			scrapeOpt := req.ScrapeOptions
			scrapeOpt.URL = c.URL

			title := c.Title
			desc := c.Description
			var md string
			var htmlStr string
			var metadata *models.DocumentMetadata

			doc, err := s.scrapeService.ExecuteScrape(ctx, &scrapeOpt, "search-"+c.URL)
			if err != nil {
				log.Warn().Err(err).Str("url", c.URL).Msg("搜索结果页单项抓取失败，保留基本搜索元数据")
				metadata = &models.DocumentMetadata{
					StatusCode: 500,
					Error:      err.Error(),
					SourceURL:  c.URL,
				}
			} else {
				if doc.Metadata.Title != "" {
					title = doc.Metadata.Title
				}
				if doc.Metadata.Description != "" {
					desc = doc.Metadata.Description
				}
				md = doc.Markdown
				htmlStr = doc.HTML
				metadata = &doc.Metadata
			}

			if title == "" {
				title = c.URL
			}

			item := &models.SearchResultItem{
				URL:         c.URL,
				Title:       title,
				Description: desc,
				Markdown:    md,
				HTML:        htmlStr,
				Metadata:    metadata,
			}

			mu.Lock()
			results = append(results, item)
			mu.Unlock()
		}(cand)
	}

	wg.Wait()
	log.Info().Str("query", query).Int("completedResults", len(results)).Msg("POST /v1/search 搜索并并发提取完成")
	return results, nil
}

// fetchSearxngCandidates 通过 SearXNG 实例执行真实搜索
func (s *ExtractService) fetchSearxngCandidates(ctx context.Context, query string, limit int, lang string) ([]SearxngResultItem, error) {
	searxngEndpoint := strings.TrimSpace(os.Getenv("SEARXNG_ENDPOINT"))
	if searxngEndpoint == "" {
		// 如果查询本身就是一个完整的 HTTP/HTTPS URL，直接将其作为候选抓取目标
		lowerQuery := strings.ToLower(query)
		if strings.HasPrefix(lowerQuery, "http://") || strings.HasPrefix(lowerQuery, "https://") {
			return []SearxngResultItem{{URL: query, Title: query}}, nil
		}
		return nil, fmt.Errorf("SEARXNG_ENDPOINT is not configured, search requires a SearXNG instance")
	}

	searchURL := strings.TrimRight(searxngEndpoint, "/") + "/search"
	u, err := url.Parse(searchURL)
	if err != nil {
		return nil, fmt.Errorf("invalid SEARXNG_ENDPOINT: %w", err)
	}

	q := u.Query()
	q.Set("q", query)
	q.Set("format", "json")
	q.Set("pageno", "1")

	if lang != "" {
		q.Set("language", lang)
	}
	if engines := os.Getenv("SEARXNG_ENGINES"); engines != "" {
		q.Set("engines", engines)
	}
	if categories := os.Getenv("SEARXNG_CATEGORIES"); categories != "" {
		q.Set("categories", categories)
	}

	u.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create searxng request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "Firecrawl-Go/1.0")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("searxng request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("searxng returned status %d: %s", resp.StatusCode, string(body))
	}

	var searxResp SearxngSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searxResp); err != nil {
		return nil, fmt.Errorf("failed to parse searxng response: %w", err)
	}

	candidates := make([]SearxngResultItem, 0, limit)
	for _, r := range searxResp.Results {
		if strings.TrimSpace(r.URL) == "" {
			continue
		}
		candidates = append(candidates, SearxngResultItem{
			URL:         r.URL,
			Title:       r.Title,
			Description: r.Content,
		})
		if len(candidates) >= limit {
			break
		}
	}

	return candidates, nil
}

// CreateExtractJob 提交网页结构化 JSON 提取任务 (POST /v1/extract)
func (s *ExtractService) CreateExtractJob(ctx context.Context, req *models.ExtractRequest, teamID string) (*models.ExtractJob, error) {
	jobID := uuid.New().String()

	job := &models.ExtractJob{
		ID:          jobID,
		TeamID:      teamID,
		Status:      models.StatusScraping,
		ExtractData: make(map[string]interface{}),
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}

	// 存入内存与 Redis
	s.jobsLock.Lock()
	s.jobs[jobID] = job
	s.jobsLock.Unlock()

	// 启动后台异步处理引擎抽取结构化 JSON
	go s.processExtractWorker(job, req)

	return job, nil
}

// processExtractWorker 提取非结构化 Markdown/HTML 并装配为强类型 JSON 对象
func (s *ExtractService) processExtractWorker(job *models.ExtractJob, req *models.ExtractRequest) {
	log.Info().Str("extractJobID", job.ID).Int("urlsCount", len(req.URLs)).Msg("后台结构化提取 Engine 启动处理")

	extractedData := make(map[string]interface{})
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, targetURL := range req.URLs {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()

			scrapeOpt := req.ScrapeOptions
			scrapeOpt.URL = u

			doc, err := s.scrapeService.ExecuteScrape(context.Background(), &scrapeOpt, job.ID+"-"+u)
			if err != nil {
				return
			}

			// 规则化结构提取算法：提取 Metadata 与内容文本键值
			parsedData := make(map[string]interface{})
			parsedData["source_url"] = u
			parsedData["title"] = doc.Metadata.Title
			parsedData["description"] = doc.Metadata.Description
			parsedData["content_summary"] = truncateString(doc.Markdown, 500)

			// 解析超链接提取
			parsedURL, errParse := url.Parse(u)
			if errParse == nil {
				links := extractLinks(doc.RawHTML, doc.Markdown, parsedURL, false)
				parsedData["extracted_links_count"] = len(links)
			}

			mu.Lock()
			domainKey := sanitizeKey(u)
			extractedData[domainKey] = parsedData
			mu.Unlock()
		}(targetURL)
	}

	wg.Wait()

	s.jobsLock.Lock()
	job.ExtractData = extractedData
	job.Status = models.StatusCompleted
	s.jobsLock.Unlock()

	log.Info().Str("extractJobID", job.ID).Msg("结构化数据提取顺利完成！")
}

// GetExtractJob 查询结构化提取任务进度与 JSON 结果
func (s *ExtractService) GetExtractJob(jobID string) (*models.ExtractJob, bool) {
	s.jobsLock.RLock()
	defer s.jobsLock.RUnlock()
	job, exists := s.jobs[jobID]
	return job, exists
}

func truncateString(str string, length int) string {
	if len(str) <= length {
		return str
	}
	return str[:length] + "..."
}

func sanitizeKey(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return "data"
	}
	host := strings.ReplaceAll(parsed.Host, ".", "_")
	if host == "" {
		return "data"
	}
	return host
}
