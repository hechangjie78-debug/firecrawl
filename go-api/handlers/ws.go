package handlers

import (
	"net/http"
	"time"

	"github.com/firecrawl/go-api/models"
	"github.com/firecrawl/go-api/services"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源跨域连接
	},
}

// WSHandler 实时 WebSocket 流式推送控制器
type WSHandler struct {
	crawlService *services.CrawlService
}

// NewWSHandler 初始化 WSHandler
func NewWSHandler(crawlService *services.CrawlService) *WSHandler {
	return &WSHandler{
		crawlService: crawlService,
	}
}

// HandleCrawlStatusWS 处理 /v1/crawl/:id WebSocket 长连接
func (h *WSHandler) HandleCrawlStatusWS(c *gin.Context) {
	jobID := c.Param("id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "jobId required"})
		return
	}

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error().Err(err).Str("jobID", jobID).Msg("WebSocket 升级失败")
		return
	}
	defer ws.Close()

	log.Info().Str("jobID", jobID).Msg("客户端成功建立 WebSocket 实时监听流")

	sentDocsCount := 0
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		job, exists := h.crawlService.GetCrawlJob(jobID)
		if !exists || job == nil {
			_ = ws.WriteJSON(gin.H{
				"type":  "error",
				"error": "Job not found",
			})
			return
		}

		// 推送新爬取完成的增量文档
		for sentDocsCount < len(job.Documents) {
			doc := job.Documents[sentDocsCount]
			err := ws.WriteJSON(gin.H{
				"type": "document",
				"data": doc,
			})
			if err != nil {
				log.Warn().Err(err).Msg("向 WebSocket 客户端推送 Document 失败，连接已断开")
				return
			}
			sentDocsCount++
		}

		// 检查任务是否已完成或取消
		if job.Status == models.StatusCompleted || job.Status == models.StatusCancelled || job.Status == models.StatusFailed {
			_ = ws.WriteJSON(gin.H{
				"type":   "done",
				"status": job.Status,
				"total":  len(job.Documents),
			})
			return
		}
	}
}
