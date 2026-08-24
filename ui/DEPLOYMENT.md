# UI Web 前端微服务部署与连接文档

## 📌 服务简介
`ui` 是基于 React / Vite 开发的前端 Web 控制台微服务，提供图形化控制界面与在线网页抓取/提取 Playground。

## 🔗 依赖与网络连接关系
* 前端网页被浏览器加载后，通过 HTTP REST API 调用后端 `api-gateway`。
* 环境变量 / 配置：指向 `http://209.33.176.13:3002`

## 🚀 独立部署指令

```bash
# 1. 构建 Nginx 静态服务镜像
docker build -t firecrawl-ui .

# 2. 运行容器 (暴露 32088 或 3000 端口)
docker run -d \
  --name firecrawl-ui \
  -p 32088:80 \
  firecrawl-ui
```

## 🔍 访问测试
在浏览器直接访问：`http://209.33.176.13:32088/`
