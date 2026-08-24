# Firecrawl API 重构与迁移完成总览 (apiplan.md)

本文档记录了 Firecrawl 后端 API 从 Node.js (Express) 全面迁移至 Go (Gin) 框架的重构完成状态。

---

## 📊 重构进度总览

- **总接口数**: 22 个
- **已完成 Go 重构**: 22 个 (100% 达成全部迁移)
- **迁移成果**: 生产级 Go 网关全面接管原 Node.js API Gateway，并在单机和分布式场景下完成高并发与单元测试验证。
- **下线状态**: 原版 Node.js `api-gateway` 已安全下线并清理。

---

## 🚀 接口实现清单 (全部 100% Go 原生实现)

### 1. 核心抓取与爬取 (Scraping & Crawling) - 🟢 已完成
- `POST /v1/scrape` - 单页抓取与内容转换
- `GET /v1/scrape/:jobId` - 单页抓取状态查询
- `POST /v1/crawl` - 开启整站/多页递归爬取任务
- `GET /v1/crawl/:jobId` - 获取爬取任务状态与结果
- `DELETE /v1/crawl/:jobId` - 取消正在运行的爬取任务
- `GET /v1/crawl/ongoing` & `GET /v1/crawl/active` - 获取正在运行的活动爬取任务列表
- `GET /v1/crawl/:jobId/errors` - 爬取错误与拦截列表

### 2. 批量处理与站点地图 (Batch & Mapping) - 🟢 已完成
- `POST /v1/batch/scrape` - 提交批量 URLs 抓取任务
- `GET /v1/batch/scrape/:jobId` - 查询批量抓取任务结果
- `GET /v1/batch/scrape/:jobId/errors` - 批量抓取错误明细
- `DELETE /v1/batch/scrape/:jobId` - 取消批量抓取任务
- `POST /v1/map` - 快速扫描并生成网站 Sitemap 结构

### 3. AI 智能提取与搜索 (Extract & Search) - 🟢 已完成
- `POST /v1/search` - 网页搜索 + 内容自动抓取提取
- `POST /v1/extract` - 基于 LLM 结构化提取网页数据
- `GET /v1/extract/:jobId` - 查询结构化提取任务状态
- `POST /v1/deep-research` - 深度研究与智能摘要提取
- `GET /v1/deep-research/:jobId` - 获取深度研究任务结果
- `POST /v1/llmstxt` - 生成站点 llms.txt 描述说明
- `GET /v1/llmstxt/:jobId` - 获取 llms.txt 生成任务状态

### 4. 账号、配额与系统状态 (Account & System) - 🟢 已完成
- `GET /v1/concurrency-check` - 查询当前团队并发额度与占用
- `GET /v1/team/credit-usage` & `historical` - 查询积分消耗情况
- `GET /v1/team/token-usage` & `historical` - 查询 LLM Token 使用量
- `GET /v1/team/queue-status` - 查询后端任务队列状态
- `POST /v1/fireclaw` - Fireclaw 专属提取

### 5. WebSocket 实时推送与历史兼容 - 🟢 已完成
- `WS /v1/crawl/:jobId` & `WS /v2/crawl/:jobId` - WebSocket 实时推送爬取进度与增量数据
- `POST /v0/scrape`, `POST /v0/crawl`, `GET /v0/crawl/status/:id`, `DELETE /v0/crawl/cancel/:id`, `POST /v0/search`, `/v0/health/*`
