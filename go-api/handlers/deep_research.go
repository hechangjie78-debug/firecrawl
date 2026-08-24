package handlers

import (
	"net/http"

	"github.com/firecrawl/go-api/middleware"
	"github.com/firecrawl/go-api/models"
	"github.com/firecrawl/go-api/services"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// DeepResearchHandler 深度研究与智能研判控制器
type DeepResearchHandler struct {
	researchService *services.DeepResearchService
}

// NewDeepResearchHandler 初始化 DeepResearchHandler
func NewDeepResearchHandler(researchService *services.DeepResearchService) *DeepResearchHandler {
	return &DeepResearchHandler{
		researchService: researchService,
	}
}

// HandleCreateDeepResearch POST /v1/deep-research 启动深度研究任务
func (h *DeepResearchHandler) HandleCreateDeepResearch(c *gin.Context) {
	teamAuth, _ := middleware.GetTeamAuth(c)

	var req models.DeepResearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.DeepResearchResponse{
			Success: false,
			Error:   "Invalid JSON payload: " + err.Error(),
		})
		return
	}

	if req.Query == "" && req.Topic == "" {
		c.JSON(http.StatusBadRequest, models.DeepResearchResponse{
			Success: false,
			Error:   "Either 'query' or 'topic' parameter is required",
		})
		return
	}

	job, err := h.researchService.CreateDeepResearchJob(c.Request.Context(), &req, teamAuth.TeamID)
	if err != nil {
		log.Error().Err(err).Msg("创建 Deep Research 任务失败")
		c.JSON(http.StatusInternalServerError, models.DeepResearchResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.DeepResearchResponse{
		Success: true,
		ID:      job.ID,
		URL:     c.Request.Host + "/v1/deep-research/" + job.ID,
	})
}

// HandleGetDeepResearchStatus GET /v1/deep-research/:id 查询深度研究进度与结果报告
func (h *DeepResearchHandler) HandleGetDeepResearchStatus(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Job ID is required"})
		return
	}

	job, exists := h.researchService.GetDeepResearchJob(jobID)
	if !exists || job == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Deep research job not found"})
		return
	}

	resp := models.DeepResearchStatusResponse{
		Success:      job.Status != "failed",
		Status:       job.Status,
		CurrentDepth: job.CurrentDepth,
		MaxDepth:     job.MaxDepth,
		TotalURLs:    len(job.Sources),
		ExpiresAt:    job.ExpiresAt,
		Activities:   job.Activities,
		Sources:      job.Sources,
		Error:        job.Error,
	}

	if job.Status == "completed" {
		resp.Data = &models.DeepResearchData{
			FinalAnalysis: job.FinalAnalysis,
			Sources:       job.Sources,
			Activities:    job.Activities,
			JSON:          job.JSON,
		}
	}

	c.JSON(http.StatusOK, resp)
}
