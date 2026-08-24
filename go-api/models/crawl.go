package models

import (
	"context"
	"time"
)

// CrawlRequest 全站/多页面爬取请求体结构定义 (带详尽 JSON 标签与中文注释)
type CrawlRequest struct {
	// 必填：起始爬取的目标 URL 网页地址
	URL string `json:"url" binding:"required"`

	// 可选：最多爬取的网页总数量限制 (默认 10 页)
	Limit int `json:"limit,omitempty"`

	// 可选：从起始 URL 开始的最大爬取深度限制 (默认 2)
	MaxDepth int `json:"maxDepth,omitempty"`

	// 可选：指定必须包含的子路径白名单正则列表
	IncludePaths []string `json:"includePaths,omitempty"`

	// 可选：指定排除忽略的子路径黑名单正则列表
	ExcludePaths []string `json:"excludePaths,omitempty"`

	// 可选：是否允许向父级或同级路径反向爬取 (默认为 false，只向下爬取子路径)
	AllowBackwardLinks bool `json:"allowBackwardLinks,omitempty"`

	// 可选：是否允许爬取跨域的外部链接 (默认为 false，只爬取本站同源域名)
	AllowExternalLinks bool `json:"allowExternalLinks,omitempty"`

	// 可选：是否忽略站点 Sitemap XML 种子文件
	IgnoreSitemap bool `json:"ignoreSitemap,omitempty"`

	// 可选：针对每个爬取的子页面使用的提取参数配置
	ScrapeOptions ScrapeRequest `json:"scrapeOptions,omitempty"`

	// 可选：并发数控制 (同时爬取的最大协程数)
	ConcurrencyLimit int `json:"concurrencyLimit,omitempty"`
}

// CrawlJobStatus 爬虫异步任务的当前生命周期状态枚举
type CrawlJobStatus string

const (
	StatusScraping  CrawlJobStatus = "scraping"  // 正在并发抓取中
	StatusCompleted CrawlJobStatus = "completed" // 全部网页爬取完毕
	StatusFailed    CrawlJobStatus = "failed"    // 爬取异常中断失败
	StatusCancelled CrawlJobStatus = "cancelled" // 被用户主动取消
)

// CrawlStatusResponse 轮询爬取任务状态时的 HTTP 响应结构
type CrawlStatusResponse struct {
	// 任务是否处理成功
	Success bool `json:"success"`

	// 任务当前的阶段状态 (scraping, completed, failed, cancelled)
	Status CrawlJobStatus `json:"status"`

	// 已经成功爬取并转换完成的网页总数量
	Completed int `json:"completed"`

	// 计划爬取的页面总数
	Total int `json:"total"`

	// 爬取的文档结果数组 (包含各个页面的 Markdown/HTML 及 Metadata)
	Data []*Document `json:"data"`

	// 任务过期失效的时间戳
	ExpiresAt time.Time `json:"expiresAt"`

	// 错误提示信息 (如果失败)
	Error string `json:"error,omitempty"`
}

// CrawlJob 内存中维护的并发安全爬虫任务实体结构
type CrawlJob struct {
	ID                 string
	TeamID             string
	URL                string
	Status             CrawlJobStatus
	Limit              int
	MaxDepth           int
	AllowBackwardLinks bool
	AllowExternalLinks bool
	ScrapeOptions      ScrapeRequest
	Documents          []*Document
	Errors             []*CrawlErrorItem
	RobotsBlocked      []string
	VisitedURLs        map[string]bool
	CreatedAt          time.Time
	ExpiresAt          time.Time
	CancelFunc         context.CancelFunc `json:"-"`
}
