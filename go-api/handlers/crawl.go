package handlers

import (
	"net/http"

	"github.com/firecrawl/go-api/middleware"
	"github.com/firecrawl/go-api/models"
	"github.com/firecrawl/go-api/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// CrawlHandler 处理全站/多页面异步爬虫相关 HTTP 接口控制器
type CrawlHandler struct {
	crawlService *services.CrawlService
}

// NewCrawlHandler 初始化创建 CrawlHandler 实体
func NewCrawlHandler(crawlService *services.CrawlService) *CrawlHandler {
	return &CrawlHandler{
		crawlService: crawlService,
	}
}

// HandleCreateCrawl POST /v1/crawl - 提交异步全站/多页面爬取任务
func (h *CrawlHandler) HandleCreateCrawl(c *gin.Context) {
	var req models.CrawlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn().Err(err).Msg("提交爬取任务请求参数校验失败")
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "无效的请求 JSON 参数: " + err.Error(),
		})
		return
	}

	// 从中间件中获取已认证团队身份上下文
	teamAuth, _ := middleware.GetTeamAuth(c)

	// 生成标准 UUIDv7 任务标识
	crawlID, err := uuid.NewV7()
	if err != nil {
		crawlID = uuid.New()
	}
	crawlIDStr := crawlID.String()

	// 提交启动后台异步爬虫协程
	job := h.crawlService.CreateCrawlJob(c.Request.Context(), &req, teamAuth.TeamID, crawlIDStr)

	log.Info().Str("crawlID", crawlIDStr).Str("url", req.URL).Msg("异步爬取任务成功创建")

	// 遵循 Node.js API 响应契约，立即返回 200/202 成功与 url / id
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"id":      job.ID,
		"url":     c.Request.Host + "/v1/crawl/" + job.ID,
	})
}

// HandleGetCrawlStatus GET /v1/crawl/:id - 轮询获取异步爬取任务进度与结果文档
func (h *CrawlHandler) HandleGetCrawlStatus(c *gin.Context) {
	crawlID := c.Param("id")
	if crawlID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "必须指定 crawlID",
		})
		return
	}

	job, exists := h.crawlService.GetCrawlJob(crawlID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "未找到指定的爬取任务 ID: " + crawlID,
		})
		return
	}

	// 返回兼容 Express 的完整轮询结构
	resp := models.CrawlStatusResponse{
		Success:   true,
		Status:    job.Status,
		Completed: len(job.Documents),
		Total:     job.Limit,
		Data:      job.Documents,
		ExpiresAt: job.ExpiresAt,
	}

	c.JSON(http.StatusOK, resp)
}

// HandleCancelCrawl DELETE /v1/crawl/:id - 主动中断取消正在运行的爬虫任务
func (h *CrawlHandler) HandleCancelCrawl(c *gin.Context) {
	crawlID := c.Param("id")
	if crawlID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "必须指定 crawlID",
		})
		return
	}

	success := h.crawlService.CancelCrawlJob(crawlID)
	if !success {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "未找到可取消的爬虫任务 ID: " + crawlID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"status":  "cancelled",
		"message": "爬虫任务已成功取消",
	})
}

// HandleGetOngoingCrawls GET /v1/crawl/ongoing 或 /v1/crawl/active 获取活动爬虫
func (h *CrawlHandler) HandleGetOngoingCrawls(c *gin.Context) {
	teamAuth, _ := middleware.GetTeamAuth(c)
	crawls := h.crawlService.GetOngoingCrawls(teamAuth.TeamID)

	c.JSON(http.StatusOK, models.OngoingCrawlsResponse{
		Success: true,
		Crawls:  crawls,
	})
}

// HandleGetCrawlErrors GET /v1/crawl/:id/errors 获取爬取失败的错误列表与 Robots 拦截
func (h *CrawlHandler) HandleGetCrawlErrors(c *gin.Context) {
	crawlID := c.Param("id")
	if crawlID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "crawlID is required"})
		return
	}

	job, exists := h.crawlService.GetCrawlJob(crawlID)
	if !exists || job == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Job not found"})
		return
	}

	errors := job.Errors
	if errors == nil {
		errors = make([]*models.CrawlErrorItem, 0)
	}
	robotsBlocked := job.RobotsBlocked
	if robotsBlocked == nil {
		robotsBlocked = make([]string, 0)
	}

	c.JSON(http.StatusOK, models.CrawlErrorsResponse{
		Success:       true,
		Errors:        errors,
		RobotsBlocked: robotsBlocked,
	})
}
