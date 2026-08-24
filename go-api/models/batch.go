package models

import (
	"time"
)

// BatchScrapeRequest 提交批量抓取的请求参数定义
type BatchScrapeRequest struct {
	URLs    []string      `json:"urls" binding:"required"`
	Options ScrapeRequest `json:"options,omitempty"`
}

// BatchScrapeJob 代表一个批量的抓取任务实例
type BatchScrapeJob struct {
	ID        string      `json:"id"`
	TeamID    string      `json:"teamId"`
	Status    CrawlJobStatus `json:"status"` // scraping, completed, failed, cancelled
	Completed int         `json:"completed"`
	Total     int         `json:"total"`
	Documents     []*Document       `json:"documents"`
	Errors        []*CrawlErrorItem `json:"errors,omitempty"`
	RobotsBlocked []string          `json:"robotsBlocked,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
	ExpiresAt     time.Time         `json:"expiresAt"`
}

// MapRequest POST /v1/map 站点地图结构扫描请求
type MapRequest struct {
	URL           string `json:"url" binding:"required"`
	Search        string `json:"search,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	IgnoreSitemap bool   `json:"ignoreSitemap,omitempty"`
	IncludeSubdomains bool `json:"includeSubdomains,omitempty"`
}

// MapResponse POST /v1/map 的输出响应结构
type MapResponse struct {
	Success bool     `json:"success"`
	Links   []string `json:"links"`
	Error   string   `json:"error,omitempty"`
}
