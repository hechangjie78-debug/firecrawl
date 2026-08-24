# Index Worker 微服务部署与连接文档

## 📌 服务简介
`index-worker` 负责搜索引擎及知识库索引的更新任务，处理抓取后文本的向量化及检索引擎同步。

## 🔗 依赖与网络连接关系

| 依赖服务 | 作用 | 目标地址配置 (环境变量) |
| :--- | :--- | :--- |
| **PostgreSQL 数据库** | 读取抓取文档及存储索引元数据 | `POSTGRES_HOST=209.33.176.8`, `POSTGRES_PORT=5432`, `POSTGRES_USER=root`, `POSTGRES_PASSWORD=he123` |
| **Redis 缓存** | 索引任务状态缓存 | `REDIS_URL=redis://:he123@209.33.176.8:6379` |
| **RabbitMQ 消息队列** | 监听 Indexing 索引更新任务队列 | `NUQ_RABBITMQ_URL=amqp://rabbitmq:5672` |

## 🚀 独立部署指令

```bash
docker run -d \
  --name firecrawl-index-worker \
  -e POSTGRES_HOST="209.33.176.8" \
  -e POSTGRES_PORT=5432 \
  -e POSTGRES_USER="root" \
  -e POSTGRES_PASSWORD="he123" \
  -e REDIS_URL="redis://:he123@209.33.176.8:6379" \
  -e NUQ_RABBITMQ_URL="amqp://209.33.176.13:5672" \
  firecrawl-index-worker
```
