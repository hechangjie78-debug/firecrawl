# Go HTML-to-MD 转换微服务部署与连接文档

## 📌 服务简介
`go-html-to-md-service` 是基于 Go 语言开发的高性能无状态 HTML 转 Markdown 微服务。它利用流式解析与极低 CPU/内存内存占用，将复杂 HTML 字符串高速转换为干净的 Markdown 格式。

## 🔗 依赖与网络连接关系
* 无状态独立微服务，对外暴露 `8080` 端口。

## 🚀 独立部署指令

```bash
# 1. 构建镜像
docker build -t firecrawl-go-html-to-md .

# 2. 运行容器
docker run -d \
  --name firecrawl-go-html-to-md \
  -p 8080:8080 \
  firecrawl-go-html-to-md
```

## 🔍 健康检查与转换测试
```bash
curl -X POST http://209.33.176.13:8080/convert \
  -H "Content-Type: application/json" \
  -d '{"html": "<h1>Hello Firecrawl</h1>"}'
```
