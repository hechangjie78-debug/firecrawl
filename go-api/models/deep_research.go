package models

import "time"

// DeepResearchRequest 深度研究分析请求体
type DeepResearchRequest struct {
	Query          string                 `json:"query"`
	Topic          string                 `json:"topic,omitempty"` // 兼容老字段
	MaxDepth       int                    `json:"maxDepth,omitempty"`
	MaxURLs        int                    `json:"maxUrls,omitempty"`
	TimeLimit      int                    `json:"timeLimit,omitempty"`
	AnalysisPrompt string                 `json:"analysisPrompt,omitempty"`
	SystemPrompt   string                 `json:"systemPrompt,omitempty"`
	Formats        []string               `json:"formats,omitempty"`
	JSONOptions    map[string]interface{} `json:"jsonOptions,omitempty"`
}

// DeepResearchResponse 提交任务返回
type DeepResearchResponse struct {
	Success bool   `json:"success"`
	ID      string `json:"id,omitempty"`
	URL     string `json:"url,omitempty"`
	Error   string `json:"error,omitempty"`
}

// DeepResearchData 结果数据结构
type DeepResearchData struct {
	FinalAnalysis string                 `json:"finalAnalysis,omitempty"`
	Sources       []string               `json:"sources,omitempty"`
	Activities    []string               `json:"activities,omitempty"`
	JSON          map[string]interface{} `json:"json,omitempty"`
}

// DeepResearchStatusResponse 状态查询返回
type DeepResearchStatusResponse struct {
	Success      bool              `json:"success"`
	Status       string            `json:"status"`
	Data         *DeepResearchData `json:"data,omitempty"`
	Error        string            `json:"error,omitempty"`
	ExpiresAt    time.Time         `json:"expiresAt"`
	CurrentDepth int               `json:"currentDepth"`
	MaxDepth     int               `json:"maxDepth"`
	TotalURLs    int               `json:"totalUrls"`
	Activities   []string          `json:"activities,omitempty"`
	Sources      []string          `json:"sources,omitempty"`
}

// DeepResearchJob 内部维护的深度研究实体
type DeepResearchJob struct {
	ID             string                 `json:"id"`
	TeamID         string                 `json:"teamId"`
	Query          string                 `json:"query"`
	Status         string                 `json:"status"`
	CurrentDepth   int                    `json:"currentDepth"`
	MaxDepth       int                    `json:"maxDepth"`
	MaxURLs        int                    `json:"maxUrls"`
	TimeLimit      int                    `json:"timeLimit"`
	AnalysisPrompt string                 `json:"analysisPrompt"`
	FinalAnalysis  string                 `json:"finalAnalysis"`
	Sources        []string               `json:"sources"`
	Activities     []string               `json:"activities"`
	JSON           map[string]interface{} `json:"json"`
	Error          string                 `json:"error,omitempty"`
	CreatedAt      time.Time              `json:"createdAt"`
	ExpiresAt      time.Time              `json:"expiresAt"`
}
