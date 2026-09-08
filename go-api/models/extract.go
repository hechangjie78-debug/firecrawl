package models

import (
	"time"
)

// SearchRequest POST /v1/search 搜索请求定义
type SearchRequest struct {
	Query         string        `json:"query" binding:"required"`
	Limit         int           `json:"limit,omitempty"`
	Lang          string        `json:"lang,omitempty"`
	Country       string        `json:"country,omitempty"`
	Location      string        `json:"location,omitempty"`
	ScrapeOptions ScrapeRequest `json:"scrapeOptions,omitempty"`
}

// SearchResultItem 搜索抓取单项结果
type SearchResultItem struct {
	URL         string            `json:"url"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Markdown    string            `json:"markdown,omitempty"`
	HTML        string            `json:"html,omitempty"`
	RawHTML     string            `json:"rawHtml,omitempty"`
	Metadata    *DocumentMetadata `json:"metadata,omitempty"`
}

// SearchResponse POST /v1/search 的输出响应结构
type SearchResponse struct {
	Success bool                `json:"success"`
	Data    []*SearchResultItem `json:"data"`
	Warning string              `json:"warning,omitempty"`
	Error   string              `json:"error,omitempty"`
}

// ExtractRequest POST /v1/extract 结构化提取请求定义
type ExtractRequest struct {
	URLs          []string               `json:"urls,omitempty"`
	Prompt        string                 `json:"prompt,omitempty"`
	Schema        map[string]interface{} `json:"schema,omitempty"`
	ScrapeOptions ScrapeRequest          `json:"scrapeOptions,omitempty"`
}

// ExtractJob 代表一个结构化提取任务实例
type ExtractJob struct {
	ID          string                 `json:"id"`
	TeamID      string                 `json:"teamId"`
	Status      CrawlJobStatus         `json:"status"` // scraping, completed, failed, cancelled
	ExtractData map[string]interface{} `json:"extractData"`
	CreatedAt   time.Time              `json:"createdAt"`
	ExpiresAt   time.Time              `json:"expiresAt"`
}
