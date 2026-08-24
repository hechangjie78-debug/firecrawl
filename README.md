# Firecrawl 微服务架构与全 Go 统一网关项目

本目录包含将 Firecrawl 彻底重构拆分为高性能独立微服务的完整项目结构。API 网关已全面迁移为 Go 原生实现（`go-api`），提供毫秒级并发响应能力与超低内存开销。

## 📁 目录结构与微服务清单

```
/home/hcj/桌面/project/firecrawl/
├── go-api/                 # 1. 统一 API 网关微服务 (Go Gin 原生高性能网关，接管端口 3002)
│   ├── Dockerfile
│   └── routes/
├── queue-worker/           # 2. 抓取队列 Worker 微服务 (处理 Scrape & Crawl 任务)
│   ├── Dockerfile
│   └── .github/workflows/deploy.yml
├── extract-worker/         # 3. 数据提取 Worker 微服务 (处理 LLM 结构化提取)
│   ├── Dockerfile
│   └── .github/workflows/deploy.yml
├── index-worker/           # 4. 索引 Worker 微服务 (处理搜索索引更新)
│   ├── Dockerfile
│   └── .github/workflows/deploy.yml
├── playwright-service/     # 5. Playwright 浏览器渲染微服务 (动态 JS 渲染与模拟操作)
│   ├── Dockerfile
│   └── .github/workflows/deploy.yml
├── go-html-to-md-service/  # 6. Go HTML-to-MD 转换微服务 (流式高性能 Markdown 转换)
│   ├── Dockerfile
│   └── .github/workflows/deploy.yml
├── ui/                     # 7. Web 前端 UI 微服务 (网页控制台应用)
│   ├── Dockerfile
│   └── .github/workflows/deploy.yml
├── dashboard/              # 8. 轻量级管理面板
└── docker-compose.yml      # 统领所有微服务的编排文件
```

## 🚀 本地一键启动所有微服务

在当前根目录下执行以下命令：

```bash
docker compose up -d
```

## 🔄 API 网关核心特性 (Go Gin)
- **高性能 & 低资源占用**：原生 Go 协程并发，极速响应。
- **全接口覆盖**：支持 `/v1/scrape`、`/v1/crawl`、`/v1/batch/scrape`、`/v1/map`、`/v1/search`、`/v1/extract`、`/v1/deep-research`、`/v1/llmstxt`、WebSocket 增量流及 `/v0` 历史兼容路由。
- **高可用消息机制**：结合 Redis 分布式持久化与 RabbitMQ 任务队列削峰。
