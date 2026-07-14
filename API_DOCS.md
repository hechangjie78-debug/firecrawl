# Firecrawl API 接口文档

> **Base URL**: `http://209.33.176.13:32088`
> **鉴权**: 当前为无鉴权模式（`USE_DB_AUTHENTICATION=false`），`Authorization` 头可省略
> **数据格式**: 全部 JSON

---

## 一、V2 核心接口（推荐使用）

### 1.1 单 URL 抓取

```
POST /v2/scrape
```

```json
{
  "url": "https://example.com",
  "formats": ["markdown"],
  "onlyMainContent": true,
  "timeout": 30000,
  "waitFor": 0,
  "mobile": false,
  "includeTags": [],
  "excludeTags": [],
  "headers": {},
  "proxy": "auto",
  "blockAds": true,
  "removeBase64Images": true,
  "fastMode": false,
  "storeInCache": true,
  "location": { "country": "us-generic" },
  "actions": [
    { "type": "wait", "milliseconds": 1000 },
    { "type": "click", "selector": "#button" },
    { "type": "scroll", "direction": "down" }
  ]
}
```

| 字段 | 类型 | 默认 | 必填 | 说明 |
|------|------|------|------|------|
| `url` | `string` | - | ✅ | 目标 URL |
| `formats` | `string[]` | `["markdown"]` | - | `markdown` `html` `rawHtml` `links` `screenshot` `json` `summary` `extract` |
| `onlyMainContent` | `bool` | `true` | - | 只保留正文 |
| `timeout` | `int` | `30000` | - | 超时(ms) |
| `waitFor` | `int` | `0` | - | 页面加载后等待(ms) |
| `mobile` | `bool` | `false` | - | 模拟移动端 |
| `includeTags` | `string[]` | `[]` | - | 只抓取这些标签 |
| `excludeTags` | `string[]` | `[]` | - | 排除这些标签 |
| `headers` | `object` | `{}` | - | 自定义请求头 |
| `proxy` | `string` | `"auto"` | - | `basic` `stealth` `enhanced` `auto` |
| `blockAds` | `bool` | `true` | - | 拦截广告 |
| `removeBase64Images` | `bool` | `true` | - | 移除 base64 图片 |
| `fastMode` | `bool` | `false` | - | 不渲染JS |
| `storeInCache` | `bool` | `true` | - | 缓存结果 |
| `location` | `object` | - | - | IP 地理位置 |
| `actions` | `array` | - | - | 页面交互动作 |

**actions 支持的类型**：

```json
{ "type": "click", "selector": "#button" }
{ "type": "scroll", "direction": "down" }
{ "type": "wait", "milliseconds": 2000 }
{ "type": "screenshot", "fullPage": true }
{ "type": "write", "text": "hello", "selector": "#input" }
{ "type": "press", "key": "Enter" }
{ "type": "select", "value": "opt", "selector": "#sel" }
```

**响应**：

```json
{
  "success": true,
  "data": {
    "markdown": "# Page Title\n\nContent...",
    "html": "<html>...</html>",
    "rawHtml": "<html>...</html>",
    "links": ["https://..."],
    "screenshot": "base64...",
    "metadata": {
      "title": "Example Domain",
      "sourceURL": "https://example.com",
      "statusCode": 200,
      "scrapeId": "uuid"
    }
  }
}
```

```bash
# 基础
curl -X POST http://209.33.176.13:32088/v2/scrape \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://example.com"}'

# 截图
curl -X POST http://209.33.176.13:32088/v2/scrape \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://example.com", "formats": ["screenshot"]}'

# 带交互
curl -X POST http://209.33.176.13:32088/v2/scrape \
  -H 'Content-Type: application/json' \
  -d '{
    "url": "https://example.com",
    "actions": [
      {"type": "wait", "milliseconds": 2000},
      {"type": "screenshot", "fullPage": true}
    ]
  }'
```

---

### 1.2 异步抓取

```
POST /v2/scrape                  -> 创建（async: true）
GET  /v2/scrape/:jobId           -> 查询结果
POST /v2/scrape/:jobId/interact  -> 页面交互
DEL  /v2/scrape/:jobId/interact  -> 停止交互
```

```bash
# 创建异步任务
curl -X POST http://209.33.176.13:32088/v2/scrape \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://example.com", "async": true}'

# 轮询结果
curl http://209.33.176.13:32088/v2/scrape/your-job-id

# 页面交互
curl -X POST http://209.33.176.13:32088/v2/scrape/job-id/interact \
  -H 'Content-Type: application/json' \
  -d '{"action": "click", "selector": "#btn"}'
```

---

### 1.3 全站爬取

```
POST /v2/crawl                -> 创建爬取任务（异步）
POST /v2/crawl/params-preview -> 预览参数
GET  /v2/crawl/ongoing        -> 进行中的任务
GET  /v2/crawl/active         -> 同 /ongoing
GET  /v2/crawl/:jobId         -> 查询结果
DEL  /v2/crawl/:jobId         -> 取消
GET  /v2/crawl/:jobId/errors  -> 错误列表
WS   /v2/crawl/:jobId         -> WebSocket 实时推送
```

```json
{
  "url": "https://docs.firecrawl.dev",
  "limit": 100,
  "maxDiscoveryDepth": 2,
  "includePaths": [],
  "excludePaths": [],
  "ignoreRobotsTxt": false,
  "allowSubdomains": false,
  "allowExternalLinks": false,
  "deduplicateSimilarURLs": true,
  "sitemap": "include",
  "scrapeOptions": {
    "formats": ["markdown"],
    "onlyMainContent": true
  },
  "webhook": {
    "url": "https://your-server.com/callback",
    "metadata": {}
  }
}
```

| 字段 | 类型 | 默认 | 说明 |
|------|------|------|------|
| `url` | `string` | **必填** | 起始 URL |
| `limit` | `int` | `10000` | 最大抓取页数 |
| `maxDiscoveryDepth` | `int` | - | 最大发现深度 |
| `includePaths` | `string[]` | `[]` | 只爬这些路径(正则) |
| `excludePaths` | `string[]` | `[]` | 排除路径 |
| `allowSubdomains` | `bool` | `false` | 允许子域名 |
| `allowExternalLinks` | `bool` | `false` | 允许外链 |
| `sitemap` | `string` | `"include"` | `skip` / `include` / `only` |
| `ignoreRobotsTxt` | `bool` | `false` | 忽略 robots.txt |
| `deduplicateSimilarURLs` | `bool` | `true` | 去重相似 URL |
| `webhook` | `object` | - | 回调通知 |

**响应（创建）**：

```json
{ "success": true, "id": "job-uuid", "url": "http://.../v2/crawl/job-uuid" }
```

**响应（查询结果）**：

```json
{
  "status": "completed",
  "total": 50,
  "completed": 50,
  "creditsUsed": 50,
  "data": [
    {
      "markdown": "# Content...",
      "metadata": { "title": "Page Title", "sourceURL": "https://...", "statusCode": 200 }
    }
  ]
}
```

```bash
# 创建
curl -X POST http://209.33.176.13:32088/v2/crawl \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://docs.firecrawl.dev", "limit": 10}'

# 查询
curl http://209.33.176.13:32088/v2/crawl/job-uuid

# 取消
curl -X DELETE http://209.33.176.13:32088/v2/crawl/job-uuid

# 错误列表
curl http://209.33.176.13:32088/v2/crawl/job-uuid/errors

# WebSocket
wscat -c ws://209.33.176.13:32088/v2/crawl/job-uuid
```

**状态说明**：`scraping` 爬取中 / `completed` 已完成 / `failed` 失败 / `cancelled` 已取消

---

### 1.4 批量抓取

```
POST /v2/batch/scrape          -> 创建
GET  /v2/batch/scrape/:jobId   -> 查询
DEL  /v2/batch/scrape/:jobId   -> 取消
GET  /v2/batch/scrape/:jobId/errors -> 错误
```

```json
{
  "urls": ["https://example.com", "https://example.org"],
  "formats": ["markdown"],
  "onlyMainContent": true
}
```

```bash
curl -X POST http://209.33.176.13:32088/v2/batch/scrape \
  -H 'Content-Type: application/json' \
  -d '{"urls": ["https://example.com", "https://httpbin.org/html"]}'
```

---

### 1.5 搜索

```
POST /v2/search                    -> 搜索
POST /v2/search/:jobId/feedback    -> 提交反馈
```

```json
{
  "query": "firecrawl web scraper",
  "limit": 10,
  "sources": ["web"],
  "lang": "en",
  "country": "us",
  "scrapeOptions": { "formats": ["markdown"] },
  "includeDomains": ["github.com"],
  "excludeDomains": [],
  "tbs": "",
  "asyncScraping": false
}
```

| 字段 | 类型 | 默认 | 说明 |
|------|------|------|------|
| `query` | `string` | **必填** | 搜索词 |
| `limit` | `int` | `10` | 结果数(最大100) |
| `sources` | `string[]` | `["web"]` | `web` `news` `images` |
| `lang` | `string` | `"en"` | 语言 |
| `country` | `string` | `"us"` | 国家 |
| `includeDomains` | `string[]` | - | 限定域名 |
| `excludeDomains` | `string[]` | - | 排除域名 |
| `tbs` | `string` | - | 时间范围 |
| `asyncScraping` | `bool` | `false` | 异步抓取结果页 |

**响应**：

```json
{
  "success": true,
  "data": {
    "web": [
      { "url": "https://firecrawl.dev", "title": "Firecrawl", "description": "API", "markdown": "#..." }
    ]
  }
}
```

```bash
curl -X POST http://209.33.176.13:32088/v2/search \
  -H 'Content-Type: application/json' \
  -d '{"query": "web scraping", "limit": 5}'
```

---

### 1.6 URL 发现（站点地图）

```
POST /v2/map
```

```json
{
  "url": "https://firecrawl.dev",
  "search": "pricing",
  "limit": 5000,
  "includeSubdomains": true,
  "sitemap": "include",
  "ignoreRobotsTxt": false
}
```

| 字段 | 类型 | 默认 | 说明 |
|------|------|------|------|
| `url` | `string` | **必填** | 站点 URL |
| `search` | `string` | - | 关键词筛选 |
| `limit` | `int` | `5000` | 最大返回数 |
| `includeSubdomains` | `bool` | `true` | 包含子域名 |
| `sitemap` | `string` | `"include"` | `skip` / `include` / `only` |
| `ignoreRobotsTxt` | `bool` | `false` | 忽略 robots.txt |

```bash
curl -X POST http://209.33.176.13:32088/v2/map \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://firecrawl.dev"}'
```

---

## 二、V2 AI & 交互

### 2.1 AI Agent 自动采集

```
POST /v2/agent       -> 创建
GET  /v2/agent/:id   -> 查询
DEL  /v2/agent/:id   -> 取消
WS   /agent-livecast -> 实时推送
```

```json
{
  "prompt": "Find the pricing plans for Notion",
  "urls": ["https://notion.so/pricing"],
  "schema": {
    "type": "object",
    "properties": {
      "plans": {
        "type": "array",
        "items": {
          "type": "object",
          "properties": {
            "name": { "type": "string" },
            "price": { "type": "string" }
          }
        }
      }
    }
  },
  "model": "spark-1-pro",
  "maxCredits": 100,
  "webhook": { "url": "https://...", "metadata": {} }
}
```

| 字段 | 类型 | 默认 | 说明 |
|------|------|------|------|
| `prompt` | `string` | **必填** | 描述你需要什么数据 |
| `urls` | `string[]` | - | 限定搜索范围 |
| `schema` | `object` | - | JSON Schema 结构化输出 |
| `model` | `string` | `"spark-1-pro"` | `spark-1-pro` / `spark-1-mini` |
| `maxCredits` | `int` | - | 最大额度 |
| `webhook` | `object` | - | 回调 |

> 注意: `/v2/extract` 已废弃，统一用 `/v2/agent`

```bash
# 简单
curl -X POST http://209.33.176.13:32088/v2/agent \
  -H 'Content-Type: application/json' \
  -d '{"prompt": "Find founders of Firecrawl"}'

# 结构化
curl -X POST http://209.33.176.13:32088/v2/agent \
  -H 'Content-Type: application/json' \
  -d '{
    "prompt": "Find pricing for Notion",
    "schema": {
      "type": "object",
      "properties": {
        "plans": {
          "type": "array",
          "items": {
            "type": "object",
            "properties": {
              "name": {"type": "string"},
              "price": {"type": "string"}
            }
          }
        }
      }
    }
  }'
```

---

### 2.2 浏览器会话

```
POST   /v2/browser                  -> 创建会话
GET    /v2/browser                  -> 会话列表
POST   /v2/browser/:id/execute      -> 执行命令
DELETE /v2/browser/:id              -> 关闭
POST   /v2/browser/webhook/destroyed -> 销毁回调
```

```bash
# 创建
curl -X POST http://209.33.176.13:32088/v2/browser

# 执行命令
curl -X POST http://209.33.176.13:32088/v2/browser/session-id/execute \
  -H 'Content-Type: application/json' \
  -d '{"action": "click", "selector": "#submit-btn"}'

# 关闭
curl -X DELETE http://209.33.176.13:32088/v2/browser/session-id
```

**execute actions**:

| action | 说明 |
|--------|------|
| `click` | 点击 `{ "selector": "#id" }` |
| `write` | 输入 `{ "text": "hello", "selector": "#input" }` |
| `press` | 按键 `{ "key": "Enter" }` |
| `scroll` | 滚动 `{ "direction": "down" }` |
| `wait` | 等待 `{ "milliseconds": 1000 }` |
| `screenshot` | 截图 `{ "fullPage": true }` |
| `select` | 下拉框 `{ "value": "opt", "selector": "#sel" }` |
| `getHtml` | 获取 HTML |
| `getContent` | 获取 Markdown |
| `getCookies` | 获取 Cookie |

---

### 2.3 文件解析

```
POST /v2/parse
```

multipart/form-data 上传文件，支持 PDF / DOCX / 图片（OCR）。

```bash
curl -X POST http://209.33.176.13:32088/v2/parse \
  -F "file=@document.pdf" \
  -F "formats=markdown"
```

---

## 三、V2 监控 & 团队

### 3.1 URL 变更监控

```
POST   /v2/monitor                     -> 创建
GET    /v2/monitor                     -> 列表
GET    /v2/monitor/:id                 -> 详情
PATCH  /v2/monitor/:id                 -> 更新
DELETE /v2/monitor/:id                 -> 删除
POST   /v2/monitor/:id/run             -> 立即检查
GET    /v2/monitor/:id/checks          -> 检查历史
GET    /v2/monitor/:id/checks/:checkId -> 详情
POST   /v2/monitor/email/confirm       -> 确认邮件
POST   /v2/monitor/email/unsubscribe   -> 退订
```

```json
{
  "url": "https://example.com",
  "interval": "daily",
  "prompt": "Check if pricing changed",
  "webhook": { "url": "https://..." }
}
```

---

### 3.2 团队用量

```
GET /v2/team/credit-usage             -> 额度
GET /v2/team/credit-usage/historical  -> 历史额度
GET /v2/team/token-usage              -> Token
GET /v2/team/token-usage/historical   -> 历史 Token
GET /v2/team/queue-status             -> 队列状态
GET /v2/team/activity                 -> 最近活动
GET /v2/concurrency-check             -> 并发状态
```

---

### 3.3 其他

```
POST /v2/support/ask                  -> 客服
POST /v2/support/docs-search          -> 文档搜索
ALL  /v2/research/*                   -> 研究服务(需配置)
POST /v2/x402/search                  -> 微支付(需配置)
```

---

## 四、V1 接口（向后兼容）

所有 V2 接口都有 V1 版本，路径换 `/v1/`：

```
POST /v1/scrape
POST /v1/crawl
POST /v1/batch/scrape
POST /v1/search
POST /v1/map
GET  /v1/crawl/ongoing
GET  /v1/crawl/active
GET  /v1/crawl/:jobId
DEL  /v1/crawl/:jobId
GET  /v1/crawl/:jobId/errors
GET  /v1/batch/scrape/:jobId
DEL  /v1/batch/scrape/:jobId
GET  /v1/batch/scrape/:jobId/errors
GET  /v1/scrape/:jobId
GET  /v1/concurrency-check
WS   /v1/crawl/:jobId
GET  /v1/team/credit-usage
GET  /v1/team/queue-status
```

**V1 专用**：

```
POST /v1/fireclaw   -> 深度抓取(100 credits)
```

```bash
curl -X POST http://209.33.176.13:32088/v1/fireclaw \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://example.com"}'
```

---

## 五、V0 接口（已废弃，仅健康检查保留）

```
GET /v0/health/liveness   -> 存活检测
GET /v0/health/readiness  -> 就绪检测
```

```bash
curl http://209.33.176.13:32088/v0/health/liveness
# {"status":"ok"}
```

其余 V0 接口（`/v0/scrape`、`/v0/crawl`、`/v0/search` 等）已废弃，前端不要用。

---

## 六、系统接口

```
GET  /             -> API 根信息
GET  /e2e-test     -> E2E 健康检测
GET  /is-production -> 是否生产环境
```

```bash
curl http://209.33.176.13:32088/
curl http://209.33.176.13:32088/e2e-test
```

---

## 七、管理员接口

路径格式：`/admin/{BULL_AUTH_KEY}/...`，当前 `BULL_AUTH_KEY = firecrawl-admin`

```
GET    /admin/firecrawl-admin/redis-health           -> Redis 健康
GET    /admin/firecrawl-admin/autumn-health          -> 计费健康
POST   /admin/firecrawl-admin/acuc-cache-clear       -> 清理缓存
GET    /admin/firecrawl-admin/feng-check             -> 引擎连通性
GET    /admin/firecrawl-admin/cclog                  -> 并发日志
GET    /admin/firecrawl-admin/precrawl               -> 预抓取
GET    /admin/firecrawl-admin/metrics                -> 应用指标
GET    /admin/firecrawl-admin/nuq-metrics            -> NUQ 指标
POST   /admin/firecrawl-admin/fsearch               -> 实时搜索
POST   /admin/firecrawl-admin/concurrency-queue-backfill -> 回填
POST   /admin/firecrawl-admin/crawl-monitor          -> 监控爬取
GET    /admin/firecrawl-admin/queues/*               -> Bull 队列管理
```

```bash
curl http://209.33.176.13:32088/admin/firecrawl-admin/redis-health
curl http://209.33.176.13:32088/admin/firecrawl-admin/metrics
```

---

## 八、前端对接建议

### 8.1 异步任务轮询

```javascript
async function pollCrawl(jobId) {
  while (true) {
    const res = await fetch(`http://209.33.176.13:32088/v2/crawl/${jobId}`);
    const data = await res.json();
    if (data.status === 'completed' || data.status === 'failed') {
      return data;
    }
    await new Promise(r => setTimeout(r, 2000));
  }
}
```

### 8.2 WebSocket 实时

```javascript
const ws = new WebSocket('ws://209.33.176.13:32088/v2/crawl/job-uuid');
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(data.status, data.completed, data.total);
};
```

### 8.3 错误处理

```json
// 成功
{ "success": true, "data": { ... } }
// 失败
{ "success": false, "error": "错误信息" }
```

| HTTP 状态码 | 含义 |
|-------------|------|
| 200 | 成功 |
| 400 | 参数错误 |
| 404 | 资源不存在 |
| 429 | 速率限制 |
| 500 | 服务端错误 |

---

## 附录：端点速查表

```
========== V2 核心 ==========
POST   /v2/scrape                   单 URL 抓取
GET    /v2/scrape/:jobId            异步结果
POST   /v2/scrape/:jobId/interact   页面交互
DEL    /v2/scrape/:jobId/interact   停止交互
POST   /v2/crawl                    全站爬取
POST   /v2/crawl/params-preview     预览参数
GET    /v2/crawl/ongoing            进行中
GET    /v2/crawl/:jobId             查询爬取
DEL    /v2/crawl/:jobId             取消
GET    /v2/crawl/:jobId/errors      错误
WS     /v2/crawl/:jobId             实时推送
POST   /v2/batch/scrape             批量
GET    /v2/batch/scrape/:jobId      查询批量
DEL    /v2/batch/scrape/:jobId      取消批量
GET    /v2/batch/scrape/:jobId/errors 批量错误
POST   /v2/search                   搜索
POST   /v2/search/:jobId/feedback   搜索反馈
POST   /v2/map                      URL 发现

========== V2 AI & 交互 ==========
POST   /v2/agent                    AI Agent
GET    /v2/agent/:jobId             查询
DEL    /v2/agent/:jobId             取消
POST   /v2/browser                  创建浏览器
GET    /v2/browser                  浏览器列表
POST   /v2/browser/:id/execute      执行
DEL    /v2/browser/:id              关闭
POST   /v2/parse                    文件解析
WS     /agent-livecast              Agent 推送

========== V2 监控 & 团队 ==========
POST   /v2/monitor                  创建监控
GET    /v2/monitor                  监控列表
GET    /v2/monitor/:id              详情
PATCH  /v2/monitor/:id              更新
DEL    /v2/monitor/:id              删除
POST   /v2/monitor/:id/run          立即执行
GET    /v2/monitor/:id/checks       检查历史
GET    /v2/monitor/:id/checks/:checkId 详情
GET    /v2/team/credit-usage        额度
GET    /v2/team/queue-status        队列
GET    /v2/concurrency-check        并发
POST   /v2/support/ask              客服

========== V1 / V0 / 系统 ==========
POST   /v1/scrape                   (兼容)
POST   /v1/crawl                    (兼容)
POST   /v1/fireclaw                 深度抓取
GET    /v0/health/liveness          存活
GET    /v0/health/readiness         就绪
GET    /                            根信息
GET    /e2e-test                    E2E 检测

========== Admin ==========
GET    /admin/:key/redis-health
GET    /admin/:key/autumn-health
POST   /admin/:key/acuc-cache-clear
GET    /admin/:key/feng-check
GET    /admin/:key/cclog
GET    /admin/:key/metrics
GET    /admin/:key/nuq-metrics
GET    /admin/:key/queues/*
