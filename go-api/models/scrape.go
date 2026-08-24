package models

// ScrapeAction 表示在抓取网页前需要执行的自动化动作（如点击、输入、等待等）
type ScrapeAction struct {
	Type         string `json:"type" binding:"required"` // 动作类型: click, write, wait, press, scroll, screenshot
	Selector     string `json:"selector,omitempty"`     // CSS 选择器或 XPath
	Text         string `json:"text,omitempty"`         // 输入的文本内容 (仅用于 write 动作)
	Milliseconds int    `json:"milliseconds,omitempty"` // 等待毫秒数 (仅用于 wait 动作)
	Key          string `json:"key,omitempty"`          // 按键名称 (仅用于 press 动作)
	Direction    string `json:"direction,omitempty"`    // 滚动方向: up / down (仅用于 scroll 动作)
}

// LocationOptions 地理位置配置
type LocationOptions struct {
	Country   string   `json:"country,omitempty"`   // 国家代码，如 US, CN, JP
	Languages []string `json:"languages,omitempty"` // 浏览器语言配置
}

// ChangeTrackingOptions 变更追踪配置
type ChangeTrackingOptions struct {
	Modes []string `json:"modes,omitempty"` // 追踪模式，如 json, markdown
}

// ScrapeRequest 代表 POST /v1/scrape 请求的载体参数
type ScrapeRequest struct {
	// Target URL to scrape
	URL string `json:"url,omitempty"` // 目标网页 URL

	// Output format choices
	Formats []string `json:"formats,omitempty"` // 输出格式列表, 例如 ["markdown", "html", "rawHtml", "links", "json"]

	// Filtering options
	OnlyMainContent bool     `json:"onlyMainContent,omitempty"` // 是否只提取网页主内容（去除导航栏、页脚、广告等）
	IncludeTags     []string `json:"includeTags,omitempty"`     // 专门包含的 HTML 标签列表，如 ["article", "main"]
	ExcludeTags     []string `json:"excludeTags,omitempty"`     // 需要排除的 HTML 标签列表，如 ["nav", "footer", "script"]

	// Execution parameters
	Timeout int  `json:"timeout,omitempty"` // 抓取超时时间 (毫秒)，默认 30000
	WaitFor int  `json:"waitFor,omitempty"` // 页面加载完成后额外等待的毫秒数
	Mobile  bool `json:"mobile,omitempty"`  // 是否使用移动端 User-Agent 与视口模拟

	// Advanced options
	Headers               map[string]string      `json:"headers,omitempty"`               // 传递给目标网站的自定义 HTTP 请求头
	Actions               []ScrapeAction         `json:"actions,omitempty"`               // 抓取前执行的交互动作序列
	Location              *LocationOptions       `json:"location,omitempty"`              // 地理位置/代理语言选择
	ZeroDataRetention     bool                   `json:"zeroDataRetention,omitempty"`     // 是否启用零数据留存模式
	ChangeTrackingOptions *ChangeTrackingOptions `json:"changeTrackingOptions,omitempty"` // 网页内容变更追踪选项
}

// DocumentMetadata 抓取结果文档的元数据信息
type DocumentMetadata struct {
	Title                      string `json:"title,omitempty"`                      // 网页标题
	Description                string `json:"description,omitempty"`                // Meta Description 描述
	Language                   string `json:"language,omitempty"`                   // 页面语言
	SourceURL                  string `json:"sourceURL,omitempty"`                  // 原始请求 URL
	StatusCode                 int    `json:"statusCode,omitempty"`                 // 目标网站响应状态码
	Error                      string `json:"error,omitempty"`                      // 单个抓取错误信息
	OGTitle                    string `json:"ogTitle,omitempty"`                    // OpenGraph 标题
	OGDescription              string `json:"ogDescription,omitempty"`              // OpenGraph 描述
	OGImage                    string `json:"ogImage,omitempty"`                    // OpenGraph 缩略图 URL
	ConcurrencyLimited         bool   `json:"concurrencyLimited,omitempty"`         // 是否触及团队并发上限受限
	ConcurrencyQueueDurationMs int64  `json:"concurrencyQueueDurationMs,omitempty"` // 并发队列排队等待毫秒数
}

// Document 抓取成功后返回的文档对象
type Document struct {
	Markdown   string                 `json:"markdown,omitempty"`   // 转化后的 clean Markdown 文本
	HTML       string                 `json:"html,omitempty"`       // 经过清洗处理后的 HTML
	RawHTML    string                 `json:"rawHtml,omitempty"`    // 网页原始未经清洗的 HTML
	Links      []string               `json:"links,omitempty"`      // 页面中提取出的所有超链接 URL 数组
	Screenshot string                 `json:"screenshot,omitempty"` // 网页截图 (Base64 或图片链接)
	JSON       map[string]interface{} `json:"json,omitempty"`       // 结构化提取出的 JSON 数据
	Summary    string                 `json:"summary,omitempty"`    // 页面总结
	Metadata   DocumentMetadata       `json:"metadata"`             // 元数据
}

// ScrapeResponse 代表 POST /v1/scrape 响应的统一 JSON 结构
type ScrapeResponse struct {
	Success  bool      `json:"success"`            // 请求是否成功标志
	Data     *Document `json:"data,omitempty"`     // 抓取成功时返回的数据文档
	Error    string    `json:"error,omitempty"`    // 发生错误时的可读提示信息
	Code     string    `json:"code,omitempty"`     // 错误码分类 (例如 SCRAPE_TIMEOUT, INVALID_URL)
	ScrapeID string    `json:"scrape_id,omitempty"` // 唯一请求任务 ID (UUIDv7)
}

// TeamAuth 存储解析后的团队鉴权信息，放入 Context
type TeamAuth struct {
	TeamID   string `json:"team_id"`    // 团队 ID
	APIKeyID string `json:"api_key_id"` // 使用的 API Key ID
	Plan     string `json:"plan"`       // 计费套餐/权限等级
}
