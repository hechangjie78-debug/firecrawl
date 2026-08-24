package handlers

import (
	"net/http"

	"github.com/firecrawl/go-api/middleware"
	"github.com/firecrawl/go-api/models"
	"github.com/firecrawl/go-api/services"
	"github.com/gin-gonic/gin"
)

// SystemHandler 系统资源、团队配额与并发控制器
type SystemHandler struct {
	systemService *services.SystemService
	scrapeService *services.ScrapeService
}

// NewSystemHandler 初始化 SystemHandler
func NewSystemHandler(systemService *services.SystemService, scrapeService *services.ScrapeService) *SystemHandler {
	return &SystemHandler{
		systemService: systemService,
		scrapeService: scrapeService,
	}
}

// HandleConcurrencyCheck GET /v1/concurrency-check 查询当前并发占用
func (h *SystemHandler) HandleConcurrencyCheck(c *gin.Context) {
	teamAuth, _ := middleware.GetTeamAuth(c)
	resp := h.systemService.GetConcurrencyCheck(c.Request.Context(), teamAuth.TeamID)
	c.JSON(http.StatusOK, resp)
}

// HandleCreditUsage GET /v1/team/credit-usage 查询积分额度
func (h *SystemHandler) HandleCreditUsage(c *gin.Context) {
	teamAuth, _ := middleware.GetTeamAuth(c)
	resp := h.systemService.GetCreditUsage(c.Request.Context(), teamAuth.TeamID)
	c.JSON(http.StatusOK, resp)
}

// HandleCreditUsageHistorical GET /v1/team/credit-usage/historical 查询历史积分用量
func (h *SystemHandler) HandleCreditUsageHistorical(c *gin.Context) {
	teamAuth, _ := middleware.GetTeamAuth(c)
	resp := h.systemService.GetCreditUsage(c.Request.Context(), teamAuth.TeamID)
	c.JSON(http.StatusOK, resp)
}

// HandleTokenUsage GET /v1/team/token-usage 查询 Token 用量
func (h *SystemHandler) HandleTokenUsage(c *gin.Context) {
	teamAuth, _ := middleware.GetTeamAuth(c)
	resp := h.systemService.GetTokenUsage(c.Request.Context(), teamAuth.TeamID)
	c.JSON(http.StatusOK, resp)
}

// HandleTokenUsageHistorical GET /v1/team/token-usage/historical 查询历史 Token 用量
func (h *SystemHandler) HandleTokenUsageHistorical(c *gin.Context) {
	teamAuth, _ := middleware.GetTeamAuth(c)
	resp := h.systemService.GetTokenUsage(c.Request.Context(), teamAuth.TeamID)
	c.JSON(http.StatusOK, resp)
}

// HandleQueueStatus GET /v1/team/queue-status 查询队列积压与并发状态
func (h *SystemHandler) HandleQueueStatus(c *gin.Context) {
	teamAuth, _ := middleware.GetTeamAuth(c)
	resp := h.systemService.GetQueueStatus(c.Request.Context(), teamAuth.TeamID)
	c.JSON(http.StatusOK, resp)
}

// HandleFireclaw POST /v1/fireclaw 专属高级提取接口
func (h *SystemHandler) HandleFireclaw(c *gin.Context) {
	var req models.FireclawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.FireclawResponse{
			Success: false,
			Error:   "Invalid request payload: " + err.Error(),
		})
		return
	}

	doc, err := h.scrapeService.ExecuteScrape(c.Request.Context(), &models.ScrapeRequest{
		URL:     req.URL,
		Formats: []string{"markdown", "html"},
	}, "fireclaw-"+req.URL)

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.FireclawResponse{
			Success: false,
			Error:   "Fireclaw failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.FireclawResponse{
		Success: true,
		Data:    doc,
	})
}
