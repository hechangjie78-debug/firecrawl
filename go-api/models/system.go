package models


// ConcurrencyCheckResponse 并发检查返回
type ConcurrencyCheckResponse struct {
	Success        bool `json:"success"`
	Concurrency    int  `json:"concurrency"`
	MaxConcurrency int  `json:"maxConcurrency"`
}

// CreditUsageData 积分使用数据
type CreditUsageData struct {
	RemainingCredits   int     `json:"remaining_credits"`
	PlanCredits        int     `json:"plan_credits"`
	BillingPeriodStart *string `json:"billing_period_start"`
	BillingPeriodEnd   *string `json:"billing_period_end"`
}

// CreditUsageResponse 积分查询返回
type CreditUsageResponse struct {
	Success bool             `json:"success"`
	Data    *CreditUsageData `json:"data,omitempty"`
	Error   string           `json:"error,omitempty"`
}

// TokenUsageData Token 使用数据
type TokenUsageData struct {
	RemainingTokens    int     `json:"remaining_tokens"`
	PlanTokens         int     `json:"plan_tokens"`
	BillingPeriodStart *string `json:"billing_period_start"`
	BillingPeriodEnd   *string `json:"billing_period_end"`
}

// TokenUsageResponse Token 查询返回
type TokenUsageResponse struct {
	Success bool            `json:"success"`
	Data    *TokenUsageData `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// QueueStatusResponse 任务队列状态返回
type QueueStatusResponse struct {
	Success            bool    `json:"success"`
	JobsInQueue        int     `json:"jobsInQueue"`
	ActiveJobsInQueue  int     `json:"activeJobsInQueue"`
	WaitingJobsInQueue int     `json:"waitingJobsInQueue"`
	MaxConcurrency     int     `json:"maxConcurrency"`
	MostRecentSuccess  *string `json:"mostRecentSuccess"`
	Error              string  `json:"error,omitempty"`
}

// CrawlErrorItem 单个爬取失败错误明细
type CrawlErrorItem struct {
	ID        string `json:"id,omitempty"`
	URL       string `json:"url"`
	Error     string `json:"error"`
	Code      string `json:"code,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

// CrawlErrorsResponse 爬取错误与拦截返回
type CrawlErrorsResponse struct {
	Success       bool              `json:"success"`
	Errors        []*CrawlErrorItem `json:"errors"`
	RobotsBlocked []string          `json:"robotsBlocked"`
	Error         string            `json:"error,omitempty"`
}

// OngoingCrawlItem 正在运行的活动爬虫简要信息
type OngoingCrawlItem struct {
	ID        string        `json:"id"`
	TeamID    string        `json:"teamId"`
	URL       string        `json:"url"`
	CreatedAt string        `json:"created_at"`
	Options   *CrawlRequest `json:"options,omitempty"`
}

// OngoingCrawlsResponse 正在运行的任务列表返回
type OngoingCrawlsResponse struct {
	Success bool                `json:"success"`
	Crawls  []*OngoingCrawlItem `json:"crawls"`
	Error   string              `json:"error,omitempty"`
}

// FireclawRequest Fireclaw 专属请求
type FireclawRequest struct {
	URL string `json:"url" binding:"required"`
}

// FireclawResponse Fireclaw 专属响应
type FireclawResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
