package handlers

import (
	"net/http"

	"github.com/firecrawl/go-api/middleware"
	"github.com/firecrawl/go-api/models"
	"github.com/firecrawl/go-api/services"
	"github.com/gin-gonic/gin"
)

// ExtractHandler 负责处理智能搜索与 JSON 结构化提取 HTTP 控制器
type ExtractHandler struct {
	extractService *services.ExtractService
}

// NewExtractHandler 初始化 ExtractHandler 实例
func NewExtractHandler(extractService *services.ExtractService) *ExtractHandler {
	return &ExtractHandler{extractService: extractService}
}

// HandleSearch POST /v1/search 智能网页搜索 + 自动内容并发抓取聚合
func (h *ExtractHandler) HandleSearch(c *gin.Context) {
	var req models.SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.SearchResponse{
			Success: false,
			Error:   "Invalid request payload: " + err.Error(),
		})
		return
	}

	if req.Query == "" {
		c.JSON(http.StatusBadRequest, models.SearchResponse{
			Success: false,
			Error:   "query is required",
		})
		return
	}

	results, err := h.extractService.ExecuteSearch(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.SearchResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.SearchResponse{
		Success: true,
		Data:    results,
	})
}

// HandleCreateExtract POST /v1/extract 提交基于 Prompt/Schema 的结构化提取任务
func (h *ExtractHandler) HandleCreateExtract(c *gin.Context) {
	teamAuth, ok := middleware.GetTeamAuth(c)
	if !ok || teamAuth == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized team token"})
		return
	}

	var req models.ExtractRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if len(req.URLs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "urls array cannot be empty"})
		return
	}

	job, err := h.extractService.CreateExtractJob(c.Request.Context(), &req, teamAuth.TeamID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create extract job: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"id":      job.ID,
		"url":     c.Request.Host + "/v1/extract/" + job.ID,
	})
}

// HandleGetExtractStatus GET /v1/extract/:id 查询结构化提取任务状态与 JSON 结果
func (h *ExtractHandler) HandleGetExtractStatus(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "extract job ID is required"})
		return
	}

	job, exists := h.extractService.GetExtractJob(jobID)
	if !exists || job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Extract job not found or expired"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"status":    job.Status,
		"data":      job.ExtractData,
		"createdAt": job.CreatedAt,
		"expiresAt": job.ExpiresAt,
	})
}
