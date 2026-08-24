package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/firecrawl/go-api/handlers"
	"github.com/firecrawl/go-api/routes"
	"github.com/firecrawl/go-api/services"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// 1. 初始化 Zerolog 日志框架
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	// 2. 加载环境变量配置 (默认 3002 接管原网关端口)
	port := os.Getenv("PORT")
	if port == "" {
		port = "3002"
	}
	env := os.Getenv("GO_ENV")
	log.Info().Str("env", env).Str("port", port).Msg("正在启动 Firecrawl Go API 统一网关引擎...")

	// 3. 依赖注入初始化 Service
	workerClient := services.NewScrapeService()

	redisService, err := services.NewRedisService()
	if err != nil {
		log.Warn().Err(err).Msg("Redis 客户端初始化未就绪，将在单机内存模式下运行")
	}

	rabbitmqService, err := services.NewRabbitMQService()
	if err != nil {
		log.Warn().Err(err).Msg("RabbitMQ 客户端初始化未就绪，将在单机并发模式下运行")
	}

	crawlService := services.NewCrawlService(workerClient, redisService, rabbitmqService)
	batchService := services.NewBatchService(workerClient, redisService, rabbitmqService)
	extractService := services.NewExtractService(workerClient, redisService)
	researchService := services.NewDeepResearchService(workerClient, extractService, redisService)
	llmsTxtService := services.NewLLMsTextService(workerClient, redisService)
	systemService := services.NewSystemService(redisService)

	// 4. 初始化 Handlers 控制器
	scrapeHandler := handlers.NewScrapeHandler(workerClient)
	crawlHandler := handlers.NewCrawlHandler(crawlService)
	batchHandler := handlers.NewBatchHandler(batchService)
	extractHandler := handlers.NewExtractHandler(extractService)
	researchHandler := handlers.NewDeepResearchHandler(researchService)
	llmsTxtHandler := handlers.NewLLMsTextHandler(llmsTxtService)
	systemHandler := handlers.NewSystemHandler(systemService, workerClient)
	wsHandler := handlers.NewWSHandler(crawlService)

	// 5. 独立模块挂载：使用 routes 子包加载所有中间件与路由定义
	r := routes.SetupRouter(
		scrapeHandler,
		crawlHandler,
		batchHandler,
		extractHandler,
		researchHandler,
		llmsTxtHandler,
		systemHandler,
		wsHandler,
	)

	// 6. 创建 HTTP Server 并安全启动
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP 服务启动失败")
		}
	}()

	log.Info().Msgf("Firecrawl Go API 网关已就绪，监听地址: http://0.0.0.0:%s", port)

	// 7. 平滑安全关闭 (Graceful Shutdown)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("收到系统关闭信号，正在安全关闭 Go API 服务...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("服务强制关闭异常")
	}
	log.Info().Msg("Go API 服务已安全退出")
}
