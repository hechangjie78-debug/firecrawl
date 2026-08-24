package handlers

import (
	"net/http"
	"net/url"
	"time"

	"github.com/firecrawl/go-api/middleware"
	"github.com/firecrawl/go-api/models"
	"github.com/firecrawl/go-api/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// ScrapeHandler 包含该 Controller 依赖的基础服务
type ScrapeHandler struct {
	scrapeService *services.ScrapeService
}

// NewScrapeHandler 初始化 ScrapeHandler
func NewScrapeHandler(scrapeService *services.ScrapeService) *ScrapeHandler {
	return &ScrapeHandler{
		scrapeService: scrapeService,
	}
}

// HandleScrape 处理 POST /v1/scrape 请求的核心控制器函数
func (h *ScrapeHandler) HandleScrape(c *gin.Context) {
	startTime := time.Now()

	// 1. 生成唯一请求任务 ID (采用 UUIDv7 保持时间有序)
	scrapeID, err := uuid.NewV7()
	scrapeIDStr := scrapeID.String()
	if err != nil {
		scrapeIDStr = uuid.New().String()
	}

	// 2. 从 Context 中读取解析好的团队鉴权信息
	authVal, exists := c.Get(middleware.AuthContextKey)
	if !exists {
		log.Error().Str("scrapeID", scrapeIDStr).Msg("无法从 Context 中读取团队鉴权信息")
		c.JSON(http.StatusUnauthorized, models.ScrapeResponse{
			Success: false,
			Error:   "Unauthorized: Authentication context missing",
			Code:    "UNAUTHORIZED",
		})
		return
	}
	teamAuth := authVal.(*models.TeamAuth)

	// 3. 绑定并校验前端传来的 JSON 请求参数
	var req models.ScrapeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn().Err(err).Str("scrapeID", scrapeIDStr).Msg("JSON 请求参数格式解析失败")
		c.JSON(http.StatusBadRequest, models.ScrapeResponse{
			Success: false,
			Error:   "Invalid JSON payload: " + err.Error(),
			Code:    "BAD_REQUEST",
		})
		return
	}

	if req.URL == "" {
		c.JSON(http.StatusBadRequest, models.ScrapeResponse{
			Success: false,
			Error:   "URL field is required",
			Code:    "BAD_REQUEST",
		})
		return
	}

	// 4. URL 基础格式合法性校验与补充
	parsedURL, err := url.Parse(req.URL)
	if err != nil || req.URL == "" {
		c.JSON(http.StatusBadRequest, models.ScrapeResponse{
			Success: false,
			Error:   "Invalid or missing target URL",
			Code:    "SCRAPE_INVALID_URL",
		})
		return
	}
	if parsedURL.Scheme == "" {
		req.URL = "http://" + req.URL
	}

	log.Info().
		Str("scrapeID", scrapeIDStr).
		Str("teamID", teamAuth.TeamID).
		Str("targetURL", req.URL).
		Interface("formats", req.Formats).
		Msg("收到新的 /v1/scrape 抓取任务")

	// 5. 调度下游服务
	doc, err := h.scrapeService.ExecuteScrape(c.Request.Context(), &req, scrapeIDStr)
	if err != nil {
		log.Error().Err(err).Str("scrapeID", scrapeIDStr).Msg("/v1/scrape 任务执行异常")
		c.JSON(http.StatusInternalServerError, models.ScrapeResponse{
			Success: false,
			Error:   "Scrape failed: " + err.Error(),
			Code:    "SCRAPE_EXECUTION_ERROR",
		})
		return
	}

	// 6. 计算处理总时长
	duration := time.Since(startTime)
	log.Info().
		Str("scrapeID", scrapeIDStr).
		Str("teamID", teamAuth.TeamID).
		Dur("durationMs", duration).
		Msg("/v1/scrape 抓取任务处理完毕并返回结果")

	// 7. 返回标准 JSON
	c.JSON(http.StatusOK, models.ScrapeResponse{
		Success:  true,
		Data:     doc,
		ScrapeID: scrapeIDStr,
	})
}

// HandleGetScrapeStatus GET /v1/scrape/:id 查询单页抓取任务状态
func (h *ScrapeHandler) HandleGetScrapeStatus(c *gin.Context) {
	scrapeID := c.Param("id")
	if scrapeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Job ID required"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"status":  "completed",
		"id":      scrapeID,
	})
}
