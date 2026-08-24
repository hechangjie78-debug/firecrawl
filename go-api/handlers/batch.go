package handlers

import (
	"net/http"

	"github.com/firecrawl/go-api/middleware"
	"github.com/firecrawl/go-api/models"
	"github.com/firecrawl/go-api/services"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// BatchHandler 负责处理批量抓取与站点地图 HTTP 控制器
type BatchHandler struct {
	batchService *services.BatchService
}

// NewBatchHandler 初始化 BatchHandler 实例
func NewBatchHandler(batchService *services.BatchService) *BatchHandler {
	return &BatchHandler{batchService: batchService}
}

// HandleCreateBatchScrape POST /v1/batch/scrape 提交批量并发抓取任务
func (h *BatchHandler) HandleCreateBatchScrape(c *gin.Context) {
	teamAuth, ok := middleware.GetTeamAuth(c)
	if !ok || teamAuth == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized team token"})
		return
	}

	var req models.BatchScrapeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON request payload: " + err.Error()})
		return
	}

	if len(req.URLs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "urls array cannot be empty"})
		return
	}

	job, err := h.batchService.CreateBatchScrape(c.Request.Context(), &req, teamAuth.TeamID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create batch job: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"id":      job.ID,
		"url":     c.Request.Host + "/v1/batch/scrape/" + job.ID,
	})
}

// HandleGetBatchStatus GET /v1/batch/scrape/:id 查询批量任务状态与结果
func (h *BatchHandler) HandleGetBatchStatus(c *gin.Context) {
	batchID := c.Param("id")
	if batchID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "batch job ID is required"})
		return
	}

	job, exists := h.batchService.GetBatchJob(batchID)
	if !exists || job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Batch job not found or expired"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"status":    job.Status,
		"completed": job.Completed,
		"total":     job.Total,
		"data":      job.Documents,
		"expiresAt": job.ExpiresAt,
	})
}

// HandleCancelBatch DELETE /v1/batch/scrape/:id 取消正在进行的批量抓取任务
func (h *BatchHandler) HandleCancelBatch(c *gin.Context) {
	batchID := c.Param("id")
	if batchID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "batch job ID is required"})
		return
	}

	success := h.batchService.CancelBatchJob(batchID)
	if !success {
		c.JSON(http.StatusNotFound, gin.H{"error": "Batch job not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"status":  "cancelled",
	})
}

// HandleGetBatchErrors GET /v1/batch/scrape/:id/errors 获取批量抓取错误明细
func (h *BatchHandler) HandleGetBatchErrors(c *gin.Context) {
	batchID := c.Param("id")
	if batchID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "batch job ID is required"})
		return
	}

	job, exists := h.batchService.GetBatchJob(batchID)
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

// HandleMap POST /v1/map 极速扫描并生成站点 Sitemap 结构
func (h *BatchHandler) HandleMap(c *gin.Context) {
	var req models.MapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if req.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url is required"})
		return
	}

	links, err := h.batchService.ExecuteMap(c.Request.Context(), &req)
	if err != nil {
		log.Error().Err(err).Str("url", req.URL).Msg("执行 /v1/map 导出失败")
		c.JSON(http.StatusInternalServerError, models.MapResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.MapResponse{
		Success: true,
		Links:   links,
	})
}
