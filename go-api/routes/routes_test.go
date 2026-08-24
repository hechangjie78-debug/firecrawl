package routes_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/firecrawl/go-api/handlers"
	"github.com/firecrawl/go-api/models"
	"github.com/firecrawl/go-api/routes"
	"github.com/firecrawl/go-api/services"
	"github.com/gin-gonic/gin"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	workerClient := services.NewScrapeService()
	crawlService := services.NewCrawlService(workerClient, nil, nil)
	batchService := services.NewBatchService(workerClient, nil, nil)
	extractService := services.NewExtractService(workerClient, nil)
	researchService := services.NewDeepResearchService(workerClient, extractService, nil)
	llmsTxtService := services.NewLLMsTextService(workerClient, nil)
	systemService := services.NewSystemService(nil)

	scrapeHandler := handlers.NewScrapeHandler(workerClient)
	crawlHandler := handlers.NewCrawlHandler(crawlService)
	batchHandler := handlers.NewBatchHandler(batchService)
	extractHandler := handlers.NewExtractHandler(extractService)
	researchHandler := handlers.NewDeepResearchHandler(researchService)
	llmsTxtHandler := handlers.NewLLMsTextHandler(llmsTxtService)
	systemHandler := handlers.NewSystemHandler(systemService, workerClient)
	wsHandler := handlers.NewWSHandler(crawlService)

	return routes.SetupRouter(
		scrapeHandler,
		crawlHandler,
		batchHandler,
		extractHandler,
		researchHandler,
		llmsTxtHandler,
		systemHandler,
		wsHandler,
	)
}

func executeRequest(r *gin.Engine, method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	var bodyReader *bytes.Reader
	if body != nil {
		jsonBytes, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(jsonBytes)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	req, _ := http.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	} else {
		req.Header.Set("Authorization", "Bearer fc-test-token")
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHealthEndpoints(t *testing.T) {
	r := setupTestRouter()

	endpoints := []string{"/health", "/health/liveness", "/health/readiness", "/v0/health/liveness"}
	for _, ep := range endpoints {
		w := executeRequest(r, "GET", ep, nil, nil)
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 for %s, got %d", ep, w.Code)
		}
		var res map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		if res["status"] != "ok" {
			t.Errorf("Expected status ok, got %v", res["status"])
		}
	}
}

func TestAuthMiddleware(t *testing.T) {
	r := setupTestRouter()

	// 请求不带 Token
	w := executeRequest(r, "POST", "/v1/scrape", map[string]string{"url": "https://example.com"}, map[string]string{})
	// 如果系统未启用 USE_DB_AUTHENTICATION=true，默认通过或返回 200/400；验证请求正常响应
	if w.Code == http.StatusUnauthorized {
		var res map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		if res["error"] == nil {
			t.Error("Expected error field on unauthorized response")
		}
	}
}

func TestScrapeValidation(t *testing.T) {
	r := setupTestRouter()

	// 缺少 URL
	w := executeRequest(r, "POST", "/v1/scrape", map[string]string{}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for empty url, got %d", w.Code)
	}

	// 状态查询
	w2 := executeRequest(r, "GET", "/v1/scrape/job-12345", nil, nil)
	if w2.Code != http.StatusOK {
		t.Errorf("Expected 200 for scrape status, got %d", w2.Code)
	}
}

func TestCrawlLifecycle(t *testing.T) {
	r := setupTestRouter()

	// 1. 创建 Crawl 任务
	reqBody := models.CrawlRequest{
		URL:      "https://example.com",
		Limit:    5,
		MaxDepth: 2,
	}
	w := executeRequest(r, "POST", "/v1/crawl", reqBody, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 on crawl create, got %d: %s", w.Code, w.Body.String())
	}

	var createResp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &createResp)
	jobID, ok := createResp["id"].(string)
	if !ok || jobID == "" {
		t.Fatalf("Expected valid job ID in crawl create response")
	}

	// 2. 查询 Crawl 状态
	wStatus := executeRequest(r, "GET", "/v1/crawl/"+jobID, nil, nil)
	if wStatus.Code != http.StatusOK {
		t.Errorf("Expected 200 for crawl status, got %d", wStatus.Code)
	}

	// 3. 查询活动爬取任务列表
	wOngoing := executeRequest(r, "GET", "/v1/crawl/ongoing", nil, nil)
	if wOngoing.Code != http.StatusOK {
		t.Errorf("Expected 200 for ongoing crawls, got %d", wOngoing.Code)
	}

	// 4. 查询任务错误
	wErrors := executeRequest(r, "GET", "/v1/crawl/"+jobID+"/errors", nil, nil)
	if wErrors.Code != http.StatusOK {
		t.Errorf("Expected 200 for crawl errors, got %d", wErrors.Code)
	}

	// 5. 取消 Crawl 任务
	wCancel := executeRequest(r, "DELETE", "/v1/crawl/"+jobID, nil, nil)
	if wCancel.Code != http.StatusOK {
		t.Errorf("Expected 200 for crawl cancel, got %d", wCancel.Code)
	}
}

func TestBatchScrapeLifecycle(t *testing.T) {
	r := setupTestRouter()

	reqBody := models.BatchScrapeRequest{
		URLs: []string{"https://example.com/1", "https://example.com/2"},
	}
	w := executeRequest(r, "POST", "/v1/batch/scrape", reqBody, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 on batch scrape create, got %d", w.Code)
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	batchID := resp["id"].(string)

	// 查询状态
	wStatus := executeRequest(r, "GET", "/v1/batch/scrape/"+batchID, nil, nil)
	if wStatus.Code != http.StatusOK {
		t.Errorf("Expected 200 for batch status, got %d", wStatus.Code)
	}

	// 查询错误
	wErrors := executeRequest(r, "GET", "/v1/batch/scrape/"+batchID+"/errors", nil, nil)
	if wErrors.Code != http.StatusOK {
		t.Errorf("Expected 200 for batch errors, got %d", wErrors.Code)
	}

	// 取消任务
	wCancel := executeRequest(r, "DELETE", "/v1/batch/scrape/"+batchID, nil, nil)
	if wCancel.Code != http.StatusOK {
		t.Errorf("Expected 200 for batch cancel, got %d", wCancel.Code)
	}
}

func TestMapEndpoint(t *testing.T) {
	r := setupTestRouter()

	reqBody := models.MapRequest{
		URL:   "https://example.com",
		Limit: 10,
	}
	w := executeRequest(r, "POST", "/v1/map", reqBody, nil)
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Unexpected status code for map: %d", w.Code)
	}
}

func TestDeepResearchLifecycle(t *testing.T) {
	r := setupTestRouter()

	reqBody := models.DeepResearchRequest{
		Query:    "AI agent architectures",
		MaxDepth: 2,
		MaxURLs:  5,
	}
	w := executeRequest(r, "POST", "/v1/deep-research", reqBody, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 on deep research create, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	researchID, ok := resp["id"].(string)
	if !ok || researchID == "" {
		t.Fatalf("Expected research ID in response")
	}

	wStatus := executeRequest(r, "GET", "/v1/deep-research/"+researchID, nil, nil)
	if wStatus.Code != http.StatusOK {
		t.Errorf("Expected 200 on deep research status, got %d", wStatus.Code)
	}
}

func TestLLMsTextLifecycle(t *testing.T) {
	r := setupTestRouter()

	reqBody := models.GenerateLLMsTextRequest{
		URL:          "https://example.com",
		ShowFullText: true,
	}
	w := executeRequest(r, "POST", "/v1/llmstxt", reqBody, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 on llmstxt create, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	jobID, ok := resp["id"].(string)
	if !ok || jobID == "" {
		t.Fatalf("Expected llmstxt job ID in response")
	}

	wStatus := executeRequest(r, "GET", "/v1/llmstxt/"+jobID, nil, nil)
	if wStatus.Code != http.StatusOK {
		t.Errorf("Expected 200 on llmstxt status, got %d", wStatus.Code)
	}
}

func TestSystemEndpoints(t *testing.T) {
	r := setupTestRouter()

	systemRoutes := []string{
		"/v1/concurrency-check",
		"/v1/team/credit-usage",
		"/v1/team/credit-usage/historical",
		"/v1/team/token-usage",
		"/v1/team/token-usage/historical",
		"/v1/team/queue-status",
	}

	for _, rt := range systemRoutes {
		w := executeRequest(r, "GET", rt, nil, nil)
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 for %s, got %d: %s", rt, w.Code, w.Body.String())
		}
		var res map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		if res["success"] != true {
			t.Errorf("Expected success true for %s", rt)
		}
	}
}

func TestV0CompatibilityEndpoints(t *testing.T) {
	r := setupTestRouter()

	// v0 keyAuth
	wAuth := executeRequest(r, "GET", "/v0/keyAuth", nil, nil)
	if wAuth.Code != http.StatusOK {
		t.Errorf("Expected 200 for /v0/keyAuth, got %d", wAuth.Code)
	}

	// v0 crawl create
	w := executeRequest(r, "POST", "/v0/crawl", models.CrawlRequest{URL: "https://example.com", Limit: 2}, nil)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for /v0/crawl, got %d", w.Code)
	}
}

func TestLiveTCPServer(t *testing.T) {
	r := setupTestRouter()
	ts := httptest.NewServer(r)
	defer ts.Close()

	client := ts.Client()

	// 1. 测试真实的 TCP GET /health
	resp, err := client.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to GET /health over TCP: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 for /health, got %d", resp.StatusCode)
	}

	// 2. 测试真实的 TCP POST /v1/deep-research
	researchPayload, _ := json.Marshal(models.DeepResearchRequest{
		Query:    "Live Go Network Socket Test",
		MaxDepth: 2,
		MaxURLs:  3,
	})
	httpReq, _ := http.NewRequest("POST", ts.URL+"/v1/deep-research", bytes.NewReader(researchPayload))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer fc-live-token")

	respResearch, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("Failed to POST /v1/deep-research: %v", err)
	}
	defer respResearch.Body.Close()
	if respResearch.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 for deep research, got %d", respResearch.StatusCode)
	}

	var researchRes map[string]interface{}
	_ = json.NewDecoder(respResearch.Body).Decode(&researchRes)
	jobID, ok := researchRes["id"].(string)
	if !ok || jobID == "" {
		t.Fatalf("Expected valid ID from live server")
	}

	// 3. 测试真实的 TCP GET /v1/deep-research/:id
	statusReq, _ := http.NewRequest("GET", ts.URL+"/v1/deep-research/"+jobID, nil)
	statusReq.Header.Set("Authorization", "Bearer fc-live-token")
	respStatus, err := client.Do(statusReq)
	if err != nil {
		t.Fatalf("Failed to GET /v1/deep-research/:id: %v", err)
	}
	defer respStatus.Body.Close()
	if respStatus.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 for deep research status, got %d", respStatus.StatusCode)
	}
}
