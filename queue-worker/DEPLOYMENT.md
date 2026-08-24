# Queue Worker 微服务部署与连接文档

## 📌 服务简介
`queue-worker` 是负责执行网页抓取 (Scrape) 与站点递归爬取 (Crawl) 核心任务的后台 Worker 节点。它从 RabbitMQ 消息队列获取任务调度指令，调用 Playwright 或 Go 服务抓取页面并写入结果。

## 🔗 依赖与网络连接关系

| 依赖服务 | 作用 | 目标地址配置 (环境变量) |
| :--- | :--- | :--- |
| **PostgreSQL 数据库** | 读取与更新 Crawl/Job 状态数据 | `POSTGRES_HOST=209.33.176.8`, `POSTGRES_PORT=5432`, `POSTGRES_USER=root`, `POSTGRES_PASSWORD=he123` |
| **Redis 缓存与队列** | Worker 限流与状态锁 | `REDIS_URL=redis://:he123@209.33.176.8:6379` |
| **RabbitMQ 消息队列** | 消费抓取任务队列 | `NUQ_RABBITMQ_URL=amqp://rabbitmq:5672` (或 `209.33.176.13:5672`) |
| **Playwright 渲染服务** | 页面渲染与 Action 动作 | `PLAYWRIGHT_MICROSERVICE_URL=http://playwright-service:3000/scrape` |

## 🚀 独立部署指令

```bash
# 1. 构建镜像
docker build -t firecrawl-queue-worker .

# 2. 运行容器
docker run -d \
  --name firecrawl-queue-worker \
  -e POSTGRES_HOST="209.33.176.8" \
  -e POSTGRES_PORT=5432 \
  -e POSTGRES_USER="root" \
  -e POSTGRES_PASSWORD="he123" \
  -e POSTGRES_DB="postgres" \
  -e REDIS_URL="redis://:he123@209.33.176.8:6379" \
  -e REDIS_RATE_LIMIT_URL="redis://:he123@209.33.176.8:6379" \
  -e NUQ_RABBITMQ_URL="amqp://209.33.176.13:5672" \
  -e PLAYWRIGHT_MICROSERVICE_URL="http://209.33.176.13:3000/scrape" \
  firecrawl-queue-worker
```
