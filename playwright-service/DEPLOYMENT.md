# Playwright 渲染微服务部署与连接文档

## 📌 服务简介
`playwright-service` 是基于 Node.js/Playwright 的无头浏览器微服务。它接收 HTTP 页面渲染请求，执行动态 JavaScript 渲染、截图以及 Actions (点击/滚动/输入等) 操作，向 Worker 提供渲染后的 DOM/HTML 数据。

## 🔗 依赖与网络连接关系
* 本服务为无状态（Stateless）微服务，对外暴露 `3000` 端口。
* 接收来自 `api-gateway` 和 `queue-worker` 的 HTTP 请求。

## 🚀 独立部署指令

```bash
# 1. 构建镜像
docker build -t firecrawl-playwright-service .

# 2. 运行容器
docker run -d \
  --name firecrawl-playwright \
  -p 3000:3000 \
  -e PORT=3000 \
  -e MAX_CONCURRENT_PAGES=10 \
  firecrawl-playwright-service
```

## 🔍 健康检查
```bash
curl http://209.33.176.13:3000/
```
