# Firecrawl 本地后端 API 接口文档 (v1)

> **服务基础地址 (Base URL)**: `http://localhost:3002`  
> **协议**: HTTP / WebSocket  
> **数据交换格式**: JSON (UTF-8)

---

## 🔐 身份验证 (Authentication)

本地开发环境中，默认配置为 `USE_DB_AUTHENTICATION=false`：
- **可免认证调用**：直接发起 HTTP 请求即可，系统默认分配开发租户身份；
- **携带令牌（推荐）**：请求头中加入 `Authorization: Bearer <任意测试令牌>`，例如：
  ```http
  Authorization: Bearer firecrawl_dev_key
  ```

---

## 📋 接口分类目录

1. [服务健康与系统状态](#1-服务健康与系统状态)
   - `GET /health` - 服务健康检查
   - `GET /v1/team/queue-status` - 任务队列积压与 Worker 状态
   - `GET /v1/concurrency-check` - 团队并发占用检查
2. [单页网页抓取 (Scraping)](#2-单页网页抓取-scraping)
   - `POST /v1/scrape` - 单页抓取与 Markdown 转换
   - `GET /v1/scrape/:id` - 异步单页抓取结果查询
3. [整站与多页深度爬虫 (Crawling)](#3-整站与多页深度爬虫-crawling)
   - `POST /v1/crawl` - 启动全站/多页递归爬虫
   - `GET /v1/crawl/:id` - 查询爬虫任务进度与文档列表
   - `DELETE /v1/crawl/:id` - 取消正在运行的爬取任务
   - `GET /v1/crawl/:id/errors` - 获取爬取错误与拦截列表
   - `GET /v1/crawl/ongoing` - 查看当前活动的爬取任务
   - `WS /v1/crawl/:id/ws` - WebSocket 实时流式推送增量数据
4. [批量并发与站点地图 (Batch & Mapping)](#4-批量并发与站点地图-batch--mapping)
   - `POST /v1/batch/scrape` - 提交批量网页并发抓取
   - `GET /v1/batch/scrape/:id` - 查询批量抓取任务结果
   - `DELETE /v1/batch/scrape/:id` - 取消批量抓取任务
   - `POST /v1/map` - 快速扫描站点 URL 拓扑树
5. [AI 提取、搜索与深度研究 (AI & Search)](#5-ai-提取搜索与深度研究-ai--search)
   - `POST /v1/extract` - 基于大模型的结构化 JSON 提取
   - `GET /v1/extract/:id` - 查询结构化提取任务
   - `POST /v1/search` - 网页搜索与内容自动抓取
   - `POST /v1/deep-research` - 深度多源网页探索研究
   - `GET /v1/deep-research/:id` - 查询深度研究报告
   - `POST /v1/llmstxt` - 为网站生成标准的 llms.txt
   - `GET /v1/llmstxt/:id` - 查询 llms.txt 生成结果

---

## 1. 服务健康与系统状态

### 1.1 服务健康检查
- **接口地址**: `GET /health` (同时兼容 `/v0/health/liveness`)
- **说明**: 探针健康检测，用于确认 API 网关运行正常。
- **请求示例**:
  ```bash
  curl http://localhost:3002/health
  ```
- **响应示例**:
  ```json
  {
    "status": "ok",
    "service": "go-api",
    "time": "2026-09-08T08:32:07Z"
  }
  ```

---

### 1.2 查询任务队列状态
- **接口地址**: `GET /v1/team/queue-status`
- **说明**: 查看 RabbitMQ 与 Redis 队列中的积压任务数、运行中任务数与最大并发。
- **请求示例**:
  ```bash
  curl http://localhost:3002/v1/team/queue-status
  ```
- **响应示例**:
  ```json
  {
    "success": true,
    "jobsInQueue": 0,
    "activeJobsInQueue": 0,
    "waitingJobsInQueue": 0,
    "maxConcurrency": 10,
    "mostRecentSuccess": "2026-09-08T08:32:12Z"
  }
  ```

---

### 1.3 并发额度检查
- **接口地址**: `GET /v1/concurrency-check`
- **说明**: 查看当前并发数限制及当前正在占用的并发。
- **请求示例**:
  ```bash
  curl http://localhost:3002/v1/concurrency-check
  ```
- **响应示例**:
  ```json
  {
    "success": true,
    "concurrencyLimit": 10,
    "currentConcurrency": 0
  }
  ```

---

## 2. 单页网页抓取 (Scraping)

### 2.1 单页抓取与内容转换
- **接口地址**: `POST /v1/scrape`
- **说明**: 
  - 支持静态页毫秒级快速抓取（零 Headless 浏览器开销）；
  - 动态/复杂网页自动调用底层 Playwright 模拟浏览器渲染；
  - 自动清洗页面噪点（导航、页脚、广告等），输出整洁的 Markdown 与元数据。
- **请求体 (JSON)**:
  | 参数名 | 类型 | 必填 | 说明 |
  | :--- | :--- | :--- | :--- |
  | `url` | string | 是 | 目标网页完整的 URL 地址 |
  | `formats` | array | 否 | 期望的输出格式，可选 `["markdown", "html", "rawHtml", "links"]`，默认 `["markdown"]` |
  | `onlyMainContent` | boolean | 否 | 是否仅提取正文主内容（过滤页眉/页脚/侧边栏），默认 `true` |
  | `waitFor` | int | 否 | 页面加载后额外等待毫秒数（用于等待动态内容渲染），默认 `0` |
  | `timeout` | int | 否 | 网页抓取超时时间 (毫秒)，默认 `30000` |
  | `includeTags` | array | 否 | 只保留的 HTML 标签列表，例如 `["article", "main"]` |
  | `excludeTags` | array | 否 | 剔除的 HTML 标签列表，例如 `["nav", "footer"]` |
  | `headers` | object | 否 | 自定义 HTTP 请求头 (键值对) |
  | `mobile` | boolean | 否 | 是否启用移动端 User-Agent 与视口模拟 |
- **请求示例**:
  ```bash
  curl -X POST http://localhost:3002/v1/scrape \
    -H "Content-Type: application/json" \
    -d '{
      "url": "https://example.com",
      "formats": ["markdown", "html"],
      "onlyMainContent": true
    }'
  ```
- **响应示例**:
  ```json
  {
    "success": true,
    "data": {
      "markdown": "Example Domain\n\n# Example Domain\n\nThis domain is for use in documentation examples...",
      "html": "<p>Example Domain...</p>",
      "metadata": {
        "title": "Example Domain",
        "description": "",
        "sourceURL": "https://example.com",
        "statusCode": 200
      }
    },
    "scrape_id": "01a08025-7d62-7c82-a511-c7cfdb19ad69"
  }
  ```

---

## 3. 整站与多页深度爬虫 (Crawling)

### 3.1 提交递归爬虫任务
- **接口地址**: `POST /v1/crawl` (异步任务)
- **说明**: 从起始 URL 递归抓取全站或关联页面，任务将放入 RabbitMQ 异步消费，状态存入 Redis。
- **请求体 (JSON)**:
  | 参数名 | 类型 | 必填 | 默认值 | 说明 |
  | :--- | :--- | :--- | :--- | :--- |
  | `url` | string | 是 | - | 起始爬取入口 URL |
  | `limit` | int | 否 | `10` | 最大爬取的网页页面总数 |
  | `maxDepth` | int | 否 | `2` | 递归最大爬取深度 |
  | `allowBackwardLinks` | boolean | 否 | `false` | 是否允许向上/同级反向爬取 |
  | `allowExternalLinks` | boolean | 否 | `false` | 是否允许爬取跨域的外部第三方链接 |
  | `includePaths` | array | 否 | `[]` | 仅爬取的路径正则/前缀列表，例如 `["/docs/.*"]` |
  | `excludePaths` | array | 否 | `[]` | 忽略不爬取的路径白名单，例如 `["/admin", "/login"]` |
  | `scrapeOptions` | object | 否 | `{}` | 每个子页面抓取时应用的选项 (同 `ScrapeRequest`) |
- **请求示例**:
  ```bash
  curl -X POST http://localhost:3002/v1/crawl \
    -H "Content-Type: application/json" \
    -d '{
      "url": "https://example.com",
      "limit": 5,
      "maxDepth": 2
    }'
  ```
- **响应示例**:
  ```json
  {
    "success": true,
    "id": "01a08025-8c1a-7548-8d72-428dd3307270",
    "url": "127.0.0.1:3002/v1/crawl/01a08025-8c1a-7548-8d72-428dd3307270"
  }
  ```

---

### 3.2 查询爬取任务状态与结果
- **接口地址**: `GET /v1/crawl/:id`
- **说明**: 轮询任务完成状态，任务完成后返回所有抓取到的子页面文档集合。
- **响应状态值 (`status`)**:
  - `scraping`: 正在抓取中
  - `completed`: 爬取已完成
  - `cancelled`: 用户主动取消
  - `failed`: 爬取异常中断
- **请求示例**:
  ```bash
  curl http://localhost:3002/v1/crawl/01a08025-8c1a-7548-8d72-428dd3307270
  ```
- **响应示例**:
  ```json
  {
    "success": true,
    "status": "completed",
    "completed": 1,
    "total": 1,
    "data": [
      {
        "markdown": "# Example Domain\n\nThis domain is...",
        "metadata": {
          "title": "Example Domain",
          "sourceURL": "https://example.com",
          "statusCode": 200
        }
      }
    ],
    "expiresAt": "2026-09-09T08:32:19Z"
  }
  ```

---

### 3.3 取消正在运行的爬取任务
- **接口地址**: `DELETE /v1/crawl/:id` (同时兼容 `DELETE /v1/crawl/cancel/:id`)
- **说明**: 中止当前后台仍在进行的爬虫作业。
- **请求示例**:
  ```bash
  curl -X DELETE http://localhost:3002/v1/crawl/01a08025-8c1a-7548-8d72-428dd3307270
  ```
- **响应示例**:
  ```json
  {
    "success": true,
    "message": "Crawl job cancelled successfully"
  }
  ```

---

### 3.4 查看正在运行的活动爬虫
- **接口地址**: `GET /v1/crawl/ongoing` (或 `GET /v1/crawl/active`)
- **请求示例**:
  ```bash
  curl http://localhost:3002/v1/crawl/ongoing
  ```

---

### 3.5 WebSocket 实时流式推送
- **连接地址**: `ws://localhost:3002/v1/crawl/:id/ws`
- **说明**: 爬虫每抓取解析完一页，会立刻向 WebSocket 客户端推送增量文档数据，适用于前端实时渲染。

---

## 4. 批量并发与站点地图 (Batch & Mapping)

### 4.1 提交批量抓取任务
- **接口地址**: `POST /v1/batch/scrape`
- **请求体 (JSON)**:
  ```json
  {
    "urls": [
      "https://example.com/page1",
      "https://example.com/page2",
      "https://example.com/page3"
    ],
    "formats": ["markdown"],
    "onlyMainContent": true
  }
  ```
- **请求示例**:
  ```bash
  curl -X POST http://localhost:3002/v1/batch/scrape \
    -H "Content-Type: application/json" \
    -d '{
      "urls": ["https://example.com"],
      "formats": ["markdown"]
    }'
  ```
- **响应示例**:
  ```json
  {
    "success": true,
    "id": "batch_job_id_xyz",
    "url": "http://localhost:3002/v1/batch/scrape/batch_job_id_xyz"
  }
  ```

---

### 4.2 查询批量抓取结果
- **接口地址**: `GET /v1/batch/scrape/:id`
- **请求示例**:
  ```bash
  curl http://localhost:3002/v1/batch/scrape/batch_job_id_xyz
  ```

---

### 4.3 快速扫描生成站点地图 (Map)
- **接口地址**: `POST /v1/map`
- **说明**: 只扫描站点的 URL 拓扑结构与链接树，不抓取正文，速度极快。
- **请求体 (JSON)**:
  ```json
  {
    "url": "https://example.com",
    "search": "docs",
    "limit": 50
  }
  ```
- **请求示例**:
  ```bash
  curl -X POST http://localhost:3002/v1/map \
    -H "Content-Type: application/json" \
    -d '{
      "url": "https://example.com",
      "limit": 10
    }'
  ```
- **响应示例**:
  ```json
  {
    "success": true,
    "links": [
      "https://example.com",
      "https://example.com/about",
      "https://example.com/contact"
    ]
  }
  ```

---

## 5. AI 提取、搜索与深度研究 (AI & Search)

### 5.1 LLM 结构化数据提取
- **接口地址**: `POST /v1/extract`
- **说明**: 抓取目标网页后，使用大模型按给定 Schema 提取结构化数据。
- **请求体 (JSON)**:
  ```json
  {
    "urls": ["https://example.com/product/1"],
    "prompt": "提取商品名称、售价以及核心规格参数",
    "schema": {
      "type": "object",
      "properties": {
        "name": { "type": "string" },
        "price": { "type": "number" },
        "specs": { "type": "array", "items": { "type": "string" } }
      },
      "required": ["name", "price"]
    }
  }
  ```
- **响应示例**:
  ```json
  {
    "success": true,
    "id": "extract_job_123",
    "data": {
      "name": "示例商品",
      "price": 99.9,
      "specs": ["标准版", "黑色"]
    }
  }
  ```

---

### 5.2 网页智能搜索
- **接口地址**: `POST /v1/search`
- **说明**: 依赖 SearXNG 实例执行公网搜索，并自动并发抓取命中结果转换为 Markdown / HTML。
- **环境配置**:
  - `SEARXNG_ENDPOINT`: SearXNG 实例服务地址 (如 `http://searxng:8080` 或公网 SearXNG 地址)。
  - `SEARXNG_ENGINES`: (可选) 优先调用的引擎 (如 `google,bing,duckduckgo`)。
  - `SEARXNG_CATEGORIES`: (可选) 搜索分类 (默认通用搜索)。
- **请求体 (JSON)**:
  | 参数名 | 类型 | 必填 | 说明 |
  | :--- | :--- | :--- | :--- |
  | `query` | string | 是 | 搜索关键字或完整 URL |
  | `limit` | int | 否 | 抓取的最大搜索结果数，默认 `5` |
  | `lang` | string | 否 | 搜索语言代码 (例如 `zh`, `en`) |
  | `scrapeOptions` | object | 否 | 命中的结果页抓取选项 (同 `ScrapeRequest`) |
- **请求示例**:
  ```json
  {
    "query": "Firecrawl architecture",
    "limit": 5,
    "lang": "en",
    "scrapeOptions": {
      "formats": ["markdown"]
    }
  }
  ```

---

### 5.3 深度研究 (Deep Research)
- **接口地址**: `POST /v1/deep-research`
- **说明**: 针对复杂研究主题，自动多步扩展抓取并生成综合分析报告。
- **请求体 (JSON)**:
  ```json
  {
    "query": "2026年开源网络爬虫工具对比与性能评测",
    "maxDepth": 3
  }
  ```
- **查询结果**: `GET /v1/deep-research/:id`

---

### 5.4 生成网站 LLMs.txt
- **接口地址**: `POST /v1/llmstxt`
- **说明**: 自动分析网站页面并生成适合大语言模型读取的 `llms.txt` 与 `llms-full.txt` 规范文件。
- **请求体 (JSON)**:
  ```json
  {
    "url": "https://example.com",
    "maxUrls": 10
  }
  ```
- **查询结果**: `GET /v1/llmstxt/:id`

---

## 🛠️ 本地常用测试命令速查

```bash
# 1. 检查服务存活
curl http://localhost:3002/health

# 2. 快速抓取单页生成 Markdown
curl -X POST http://localhost:3002/v1/scrape \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com"}'

# 3. 启动全站爬虫任务
curl -X POST http://localhost:3002/v1/crawl \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com","limit":3}'

# 4. 查看后台并发与队列负载
curl http://localhost:3002/v1/team/queue-status
```
