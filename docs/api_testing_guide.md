# 🚀 Firecrawl Go API 接口测试与使用说明文档

本文档详细介绍了已经使用 **Go (Gin)** 框架重构完成的核心 API 接口，包括参数规格说明、鉴权方式、完整的 `curl` 测试命令以及响应数据格式示例。

---

## 🛠️ 1. 环境准备与服务架构

在测试接口之前，请确保底层依赖微服务及 Go API 主引擎已成功启动：

| 服务名称 | 内部/外部端口 | 功能描述 |
| :--- | :--- | :--- |
| **Go API 引擎 (go-api)** | `3005` | 核心路由分发、异步 Task 调度与状态维护 |
| **Playwright 微服务** | `3000` | 动态 JavaScript 页面渲染与 DOM 提取 |
| **HTML-to-Markdown 服务** | `8085` / `8080` | 高性能 HTML DOM 转纯 Markdown 文档 |

### 🔑 统一鉴权方式 (HTTP Header)
所有需要鉴权的 API 接口均须在请求头中携带 Bearer 令牌：
```http
Authorization: Bearer test_api_key_123456
```

---

## 📌 2. 接口测试说明

### 2.1 单页抓取与 Markdown 转换 (`POST /v1/scrape`)

#### 🔹 接口描述
对给定的单个网页进行加载、渲染、抽取，并实时转换为 Markdown 格式文档。支持直接抓取基于 React/Vue/Nuxt 等 SSR/SPA 渲染的动态页面。

#### 🔹 请求 URL
`POST http://127.0.0.1:3005/v1/scrape`

#### 🔹 请求 Body 参数 (JSON)
| 字段名 | 类型 | 必填 | 默认值 | 描述 |
| :--- | :--- | :--- | :--- | :--- |
| `url` | string | **是** | - | 目标网页的完整 URL |
| `formats` | array | 否 | `["markdown"]` | 输出格式，如 `["markdown", "html"]` |
| `onlyMainContent` | bool | 否 | `true` | 是否仅提取网页正文 (去除导航栏/页脚/广告) |
| `waitFor` | int | 否 | `0` | 页面加载完成后额外等待的毫秒数 (用于 AJAX 数据渲染) |

#### 🔹 `curl` 测试命令

##### 基础测试：
```bash
curl -s -X POST http://127.0.0.1:3005/v1/scrape \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test_api_key_123456" \
  -d '{
    "url": "https://httpbin.org/html"
  }'
```

##### 复杂/SSR 动态渲染站点抓取（示例：911proxy）：
```bash
curl -s -X POST http://127.0.0.1:3005/v1/scrape \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test_api_key_123456" \
  -d '{
    "url": "https://www.911proxy.com/",
    "waitFor": 2000,
    "onlyMainContent": true
  }'
```

#### 🔹 成功响应示例
```json
{
  "success": true,
  "data": {
    "markdown": "# Herman Melville - Moby-Dick\n\nAvailing himself of the mild...",
    "metadata": {
      "title": "Scraped Page Result",
      "sourceURL": "https://httpbin.org/html",
      "statusCode": 200
    }
  },
  "scrape_id": "019fe9b8-28a5-78e5-8f89-eb9837576607"
}
```

---

### 2.2 开启整站 / 多页异步爬取 (`POST /v1/crawl`)

#### 🔹 接口描述
提交一个全站/多页面异步递归爬取任务。主引擎将在后台启动轻量 Goroutine 队列，按照深度优先/广度优先策略并发抓取关联页面。

#### 🔹 请求 URL
`POST http://127.0.0.1:3005/v1/crawl`

#### 🔹 请求 Body 参数 (JSON)
| 字段名 | 类型 | 必填 | 默认值 | 描述 |
| :--- | :--- | :--- | :--- | :--- |
| `url` | string | **是** | - | 起始爬取的目标 URL 地址 |
| `limit` | int | 否 | `10` | 本次任务抓取的最大网页数量限制 |
| `maxDepth` | int | depth | `2` | 最大爬取层级深度 |
| `allowBackwardLinks` | bool | 否 | `false` | 是否允许向同级/父级目录回退爬取 |
| `allowExternalLinks` | bool | 否 | `false` | 是否允许爬取跨域外部链接 |

#### 🔹 `curl` 测试命令
```bash
curl -s -X POST http://127.0.0.1:3005/v1/crawl \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test_api_key_123456" \
  -d '{
    "url": "https://httpbin.org/html",
    "limit": 3,
    "maxDepth": 2
  }'
```

#### 🔹 成功响应示例
```json
{
  "id": "019fe9d1-290f-7673-9382-73b4240fb752",
  "success": true,
  "url": "127.0.0.1:3005/v1/crawl/019fe9d1-290f-7673-9382-73b4240fb752"
}
```

---

### 2.3 查询爬取任务状态与结果 (`GET /v1/crawl/:jobId`)

#### 🔹 接口描述
通过 `POST /v1/crawl` 返回的 `id` 轮询查询爬取进度以及已转换完成的文档 Markdown 内容。

#### 🔹 请求 URL
`GET http://127.0.0.1:3005/v1/crawl/:jobId`

#### 🔹 `curl` 测试命令
```bash
# 替换为真实的 job ID
CRAWL_ID="019fe9d1-290f-7673-9382-73b4240fb752"

curl -s http://127.0.0.1:3005/v1/crawl/$CRAWL_ID \
  -H "Authorization: Bearer test_api_key_123456"
```

#### 🔹 响应状态说明
响应结果中的 `status` 字段代表当前任务所处的阶段：
- `scraping`: 后台 Goroutine 正处于并发抓取与转换阶段。
- `completed`: 所有目标网页已全部抓取并解析完毕。
- `failed`: 爬取任务遭遇不可恢复的异常。
- `cancelled`: 任务已被用户手动中断取消。

#### 🔹 抓取中 (`scraping`) 响应示例
```json
{
  "success": true,
  "status": "scraping",
  "completed": 0,
  "total": 3,
  "data": [],
  "expiresAt": "2026-08-11T03:57:06.703Z"
}
```

#### 🔹 完成 (`completed`) 响应示例
```json
{
  "success": true,
  "status": "completed",
  "completed": 1,
  "total": 3,
  "data": [
    {
      "markdown": "# Herman Melville - Moby-Dick\n\nAvailing himself...",
      "metadata": {
        "title": "Scraped Page Result",
        "sourceURL": "https://httpbin.org/html",
        "statusCode": 200
      }
    }
  ],
  "expiresAt": "2026-08-11T03:57:06.703Z"
}
```

---

### 2.4 取消爬取任务 (`DELETE /v1/crawl/:jobId`)

#### 🔹 接口描述
主动取消中断正在执行的异步爬取任务。主引擎会立即触发 Context 取消信号，停止后续 Goroutine 的抓取调度。

#### 🔹 请求 URL
`DELETE http://127.0.0.1:3005/v1/crawl/:jobId`

#### 🔹 `curl` 测试命令
```bash
CRAWL_ID="019fe9d1-290f-7673-9382-73b4240fb752"

curl -s -X DELETE http://127.0.0.1:3005/v1/crawl/$CRAWL_ID \
  -H "Authorization: Bearer test_api_key_123456"
```

#### 🔹 成功响应示例
```json
{
  "success": true,
  "status": "cancelled",
  "message": "爬虫任务已成功取消"
}
```

---

## 🐍 3. Python 自动化测试脚本示例

如果你使用 Python 脚本进行自动化测试，可直接使用以下完整代码：

```python
import requests
import time

BASE_URL = "http://127.0.0.1:3005"
API_KEY = "test_api_key_123456"
HEADERS = {
    "Authorization": f"Bearer {API_KEY}",
    "Content-Type": "application/json"
}

def test_single_scrape():
    print("=== 1. 测试单页抓取 POST /v1/scrape ===")
    payload = {
        "url": "https://httpbin.org/html",
        "waitFor": 1000
    }
    resp = requests.post(f"{BASE_URL}/v1/scrape", json=payload, headers=HEADERS)
    print("响应状态码:", resp.status_code)
    print("抓取结果 preview:", resp.json().get("data", {}).get("markdown", "")[:150])

def test_async_crawl():
    print("\n=== 2. 测试多页异步爬取 POST /v1/crawl ===")
    payload = {
        "url": "https://httpbin.org/html",
        "limit": 2,
        "maxDepth": 2
    }
    resp = requests.post(f"{BASE_URL}/v1/crawl", json=payload, headers=HEADERS)
    data = resp.json()
    job_id = data.get("id")
    print(f"任务创建成功! Job ID: {job_id}")

    # 轮询获取任务状态
    for i in range(5):
        time.sleep(2)
        status_resp = requests.get(f"{BASE_URL}/v1/crawl/{job_id}", headers=HEADERS).json()
        status = status_resp.get("status")
        completed = status_resp.get("completed")
        print(f"[{i+1}/5] 任务状态: {status}, 已完成页数: {completed}")
        if status == "completed":
            print("爬取完毕! 获取到的文档数:", len(status_resp.get("data", [])))
            break

if __name__ == "__main__":
    test_single_scrape()
    test_async_crawl()
```
