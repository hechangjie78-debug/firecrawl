# 项目经历

## Firecrawl — 网页抓取与结构化数据 API 平台（Go）
基于 Go 构建的高性能网页抓取平台，将网页内容转换为干净 Markdown 与结构化 JSON，服务 AI Agent 与企业数据管道，支撑百万级页面的并发抓取。

- 使用 Go 从零搭建核心 API 服务（gin/echo），提供 /v0、/v1、/v2 抓取、爬取与提取接口，P95 延迟控制在秒级。
- 设计并实现多进程任务编排：API、worker、extract、index 等后台服务基于 Go goroutine + 消息队列解耦，水平扩展。
- 独立实现高性能 HTML→Markdown 转换引擎（go-html-to-md-service），以流式解析降低内存占用与 CPU 开销。
- 负责基础设施与自托管部署：基于 Docker Compose 与 Kubernetes（Helm）编排 PostgreSQL、Redis、队列等组件，定义资源限制与副本调度。
- 技术栈：Go、PostgreSQL、Redis、ClickHouse、Playwright、Docker、Kubernetes、gRPC。
