# Extract Worker 微服务部署与连接文档

## 📌 服务简介
`extract-worker` 是专门负责大语言模型 (LLM) 结构化数据提取的微服务，它监听 Extract 队列任务，对接 OpenAI / Ollama 等 LLM 模型接口从网页文本中提取 Schema 结构。

## 🔗 依赖与网络连接关系

| 依赖服务 | 作用 | 目标地址配置 (环境变量) |
| :--- | :--- | :--- |
| **PostgreSQL 数据库** | 存储抽取 Schema 与结果数据 | `POSTGRES_HOST=209.33.176.8`, `POSTGRES_PORT=5432`, `POSTGRES_USER=root`, `POSTGRES_PASSWORD=he123` |
| **Redis 缓存** | 任务锁与速率控制 | `REDIS_URL=redis://:he123@209.33.176.8:6379` |
| **RabbitMQ 消息队列** | 消费 Extract 提取任务队列 | `NUQ_RABBITMQ_URL=amqp://rabbitmq:5672` (或 `209.33.176.13:5672`) |

## 🚀 独立部署指令

```bash
# 1. 构建镜像
docker build -t firecrawl-extract-worker .

# 2. 运行容器
docker run -d \
  --name firecrawl-extract-worker \
  -e POSTGRES_HOST="209.33.176.8" \
  -e POSTGRES_PORT=5432 \
  -e POSTGRES_USER="root" \
  -e POSTGRES_PASSWORD="he123" \
  -e POSTGRES_DB="postgres" \
  -e REDIS_URL="redis://:he123@209.33.176.8:6379" \
  -e REDIS_RATE_LIMIT_URL="redis://:he123@209.33.176.8:6379" \
  -e NUQ_RABBITMQ_URL="amqp://209.33.176.13:5672" \
  firecrawl-extract-worker
```
