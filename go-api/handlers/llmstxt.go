package handlers

import (
	"net/http"

	"github.com/firecrawl/go-api/middleware"
	"github.com/firecrawl/go-api/models"
	"github.com/firecrawl/go-api/services"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// LLMsTextHandler 站点 llms.txt 自动化生成控制器
type LLMsTextHandler struct {
	llmsTxtService *services.LLMsTextService
}

// NewLLMsTextHandler 初始化 LLMsTextHandler
func NewLLMsTextHandler(llmsTxtService *services.LLMsTextService) *LLMsTextHandler {
	return &LLMsTextHandler{
		llmsTxtService: llmsTxtService,
	}
}

// HandleCreateLLMsText POST /v1/llmstxt 提交生成请求
func (h *LLMsTextHandler) HandleCreateLLMsText(c *gin.Context) {
	teamAuth, _ := middleware.GetTeamAuth(c)

	var req models.GenerateLLMsTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.GenerateLLMsTextResponse{
			Success: false,
			Error:   "Invalid JSON payload: " + err.Error(),
		})
		return
	}

	if req.URL == "" {
		c.JSON(http.StatusBadRequest, models.GenerateLLMsTextResponse{
			Success: false,
			Error:   "URL field is required",
		})
		return
	}

	job, err := h.llmsTxtService.CreateLLMsTextJob(c.Request.Context(), &req, teamAuth.TeamID)
	if err != nil {
		log.Error().Err(err).Msg("创建 llms.txt 任务失败")
		c.JSON(http.StatusInternalServerError, models.GenerateLLMsTextResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.GenerateLLMsTextResponse{
		Success: true,
		ID:      job.ID,
		URL:     c.Request.Host + "/v1/llmstxt/" + job.ID,
	})
}

// HandleGetLLMsTextStatus GET /v1/llmstxt/:id 查询生成状态与文本内容
func (h *LLMsTextHandler) HandleGetLLMsTextStatus(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Job ID is required"})
		return
	}

	job, exists := h.llmsTxtService.GetLLMsTextJob(jobID)
	if !exists || job == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "llms.txt generation job not found"})
		return
	}

	resp := models.GenerateLLMsTextStatusResponse{
		Success:   job.Status != "failed",
		Status:    job.Status,
		ExpiresAt: job.ExpiresAt,
		Error:     job.Error,
	}

	if job.Status == "completed" {
		data := &models.LLMsTextData{
			LLMsTxt: job.GeneratedTxt,
		}
		if job.ShowFullText {
			data.LLMsFullTxt = job.FullTxt
		}
		resp.Data = data
	}

	c.JSON(http.StatusOK, resp)
}
