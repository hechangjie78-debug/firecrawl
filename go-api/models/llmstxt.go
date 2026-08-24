package models

import "time"

// GenerateLLMsTextRequest 生成 llms.txt 格式请求体
type GenerateLLMsTextRequest struct {
	URL          string `json:"url" binding:"required"`
	MaxURLs      int    `json:"maxUrls,omitempty"`
	ShowFullText bool   `json:"showFullText,omitempty"`
	Cache        bool   `json:"cache,omitempty"`
}

// GenerateLLMsTextResponse 提交生成任务响应
type GenerateLLMsTextResponse struct {
	Success bool   `json:"success"`
	ID      string `json:"id,omitempty"`
	URL     string `json:"url,omitempty"`
	Error   string `json:"error,omitempty"`
}

// LLMsTextData 数据返回
type LLMsTextData struct {
	LLMsTxt     string `json:"llmstxt"`
	LLMsFullTxt string `json:"llmsfulltxt,omitempty"`
}

// GenerateLLMsTextStatusResponse 状态查询响应
type GenerateLLMsTextStatusResponse struct {
	Success   bool          `json:"success"`
	Status    string        `json:"status"`
	Data      *LLMsTextData `json:"data,omitempty"`
	Error     string        `json:"error,omitempty"`
	ExpiresAt time.Time     `json:"expiresAt"`
}

// LLMsTextJob 内部维护的 llms.txt 任务
type LLMsTextJob struct {
	ID           string    `json:"id"`
	TeamID       string    `json:"teamId"`
	URL          string    `json:"url"`
	MaxURLs      int       `json:"maxUrls"`
	ShowFullText bool      `json:"showFullText"`
	Cache        bool      `json:"cache"`
	Status       string    `json:"status"`
	GeneratedTxt string    `json:"generatedTxt"`
	FullTxt      string    `json:"fullTxt"`
	Error        string    `json:"error,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	ExpiresAt    time.Time `json:"expiresAt"`
}
