package routes

import (
	"net/http"
	"time"

	"github.com/firecrawl/go-api/handlers"
	"github.com/firecrawl/go-api/middleware"
	"github.com/gin-gonic/gin"
)

// SetupRouter 统一进行中间件配置与路由初始化绑定
func SetupRouter(
	scrapeHandler *handlers.ScrapeHandler,
	crawlHandler *handlers.CrawlHandler,
	batchHandler *handlers.BatchHandler,
	extractHandler *handlers.ExtractHandler,
	researchHandler *handlers.DeepResearchHandler,
	llmsTxtHandler *handlers.LLMsTextHandler,
	systemHandler *handlers.SystemHandler,
	wsHandler *handlers.WSHandler,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	// 挂载全局 CORS 跨域中间件
	r.Use(middleware.CORSMiddleware())

	// 健康检查探针路由
	healthCheck := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "go-api",
			"time":    time.Now().Format(time.RFC3339),
		})
	}
	r.GET("/health", healthCheck)
	r.GET("/health/liveness", healthCheck)
	r.GET("/health/readiness", healthCheck)
	r.GET("/v0/health/liveness", healthCheck)
	r.GET("/v0/health/readiness", healthCheck)

	// WebSocket 实时流 (支持 /v1/crawl/:id 与 /v2/crawl/:id)
	r.GET("/v1/crawl/:id/ws", wsHandler.HandleCrawlStatusWS)
	r.GET("/v2/crawl/:id/ws", wsHandler.HandleCrawlStatusWS)

	// 注册 v1 版本的受保护路由组
	v1Group := r.Group("/v1")
	v1Group.Use(middleware.AuthMiddleware())
	{
		// 1. 核心抓取与递归爬虫
		v1Group.POST("/scrape", scrapeHandler.HandleScrape)
		v1Group.GET("/scrape/:id", scrapeHandler.HandleGetScrapeStatus)
		v1Group.POST("/crawl", crawlHandler.HandleCreateCrawl)
		v1Group.GET("/crawl/ongoing", crawlHandler.HandleGetOngoingCrawls)
		v1Group.GET("/crawl/active", crawlHandler.HandleGetOngoingCrawls)
		v1Group.GET("/crawl/:id", func(c *gin.Context) {
			if c.GetHeader("Upgrade") == "websocket" {
				wsHandler.HandleCrawlStatusWS(c)
				return
			}
			crawlHandler.HandleGetCrawlStatus(c)
		})
		v1Group.GET("/crawl/:id/errors", crawlHandler.HandleGetCrawlErrors)
		v1Group.DELETE("/crawl/:id", crawlHandler.HandleCancelCrawl)
		v1Group.DELETE("/crawl/cancel/:id", crawlHandler.HandleCancelCrawl)

		// 2. 批量并发与 Sitemap 拓扑
		v1Group.POST("/batch/scrape", batchHandler.HandleCreateBatchScrape)
		v1Group.GET("/batch/scrape/:id", batchHandler.HandleGetBatchStatus)
		v1Group.GET("/batch/scrape/:id/errors", batchHandler.HandleGetBatchErrors)
		v1Group.DELETE("/batch/scrape/:id", batchHandler.HandleCancelBatch)
		v1Group.DELETE("/batch/scrape/cancel/:id", batchHandler.HandleCancelBatch)
		v1Group.POST("/map", batchHandler.HandleMap)

		// 3. 智能搜索与 JSON 结构化提取
		v1Group.POST("/search", extractHandler.HandleSearch)
		v1Group.POST("/extract", extractHandler.HandleCreateExtract)
		v1Group.GET("/extract/:id", extractHandler.HandleGetExtractStatus)

		// 4. 深度研究与 LLMs.txt 生成
		v1Group.POST("/deep-research", researchHandler.HandleCreateDeepResearch)
		v1Group.GET("/deep-research/:id", researchHandler.HandleGetDeepResearchStatus)
		v1Group.POST("/llmstxt", llmsTxtHandler.HandleCreateLLMsText)
		v1Group.GET("/llmstxt/:id", llmsTxtHandler.HandleGetLLMsTextStatus)

		// 5. 团队配额、并发与队列状态
		v1Group.GET("/concurrency-check", systemHandler.HandleConcurrencyCheck)
		v1Group.GET("/team/credit-usage", systemHandler.HandleCreditUsage)
		v1Group.GET("/team/credit-usage/historical", systemHandler.HandleCreditUsageHistorical)
		v1Group.GET("/team/token-usage", systemHandler.HandleTokenUsage)
		v1Group.GET("/team/token-usage/historical", systemHandler.HandleTokenUsageHistorical)
		v1Group.GET("/team/queue-status", systemHandler.HandleQueueStatus)
		v1Group.POST("/fireclaw", systemHandler.HandleFireclaw)
	}

	// 注册 v0 历史版本兼容路由组
	v0Group := r.Group("/v0")
	v0Group.Use(middleware.AuthMiddleware())
	{
		v0Group.POST("/scrape", scrapeHandler.HandleScrape)
		v0Group.POST("/crawl", crawlHandler.HandleCreateCrawl)
		v0Group.GET("/crawl/status/:id", crawlHandler.HandleGetCrawlStatus)
		v0Group.DELETE("/crawl/cancel/:id", crawlHandler.HandleCancelCrawl)
		v0Group.POST("/search", extractHandler.HandleSearch)
		v0Group.GET("/keyAuth", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"success": true})
		})
	}

	return r
}
