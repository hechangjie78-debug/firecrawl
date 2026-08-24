<template>
  <div class="studio-layout">
    <!-- 顶部状态与全局指标栏 Top Bar -->
    <header class="top-bar glass-card">
      <div class="brand-section">
        <div class="brand-logo">
          <Zap class="icon-zap text-cyan" :size="24" />
        </div>
        <div class="brand-text">
          <div class="brand-header">
            <span class="brand-name">Firecrawl</span>
            <span class="badge-tag">Go 2.0 Engine</span>
            <span class="badge-speed">⚡ &lt;1ms</span>
          </div>
          <p class="brand-desc">新一代超高并发网页爬虫、Sitemap 拓扑、智能搜索 & AI 深度研判控制台</p>
        </div>
      </div>

      <!-- 右侧指标与连接配置 -->
      <div class="top-actions">
        <!-- 实时指标胶囊 -->
        <div class="metrics-capsules">
          <div class="metric-pill" :title="'活跃并发 / 最大并发限制'">
            <Cpu :size="14" class="text-indigo" />
            <span class="metric-label">并发:</span>
            <span class="metric-val text-indigo">{{ systemStats.concurrency }} / {{ systemStats.maxConcurrency }}</span>
          </div>
          <div class="metric-pill" :title="'剩余可用 Credit 积分'">
            <Database :size="14" class="text-emerald" />
            <span class="metric-label">积分:</span>
            <span class="metric-val text-emerald">{{ formatNumber(systemStats.remainingCredits) }}</span>
          </div>
        </div>

        <!-- API 连接配置 -->
        <div class="config-box">
          <div class="config-input-wrap">
            <Globe :size="14" class="input-icon" />
            <input v-model="baseUrl" class="config-input" placeholder="http://localhost:3002" />
          </div>
          <div class="config-input-wrap">
            <ShieldCheck :size="14" class="input-icon" />
            <input v-model="apiKey" type="password" class="config-input key" placeholder="Bearer Token" />
          </div>
          <button class="health-btn" @click="checkHealth" :disabled="loadingHealth">
            <span class="pulse-dot" :class="healthOk ? 'bg-emerald' : 'bg-rose'"></span>
            <span>{{ loadingHealth ? '检测中' : (healthOk ? `${healthMs}ms` : '离线') }}</span>
          </button>
        </div>
      </div>
    </header>

    <div class="main-body">
      <!-- 左侧功能导航 Sidebar -->
      <aside class="sidebar glass-card">
        <div class="sidebar-title">核心微服务能力</div>
        <nav class="nav-menu">
          <button 
            v-for="tab in tabs" 
            :key="tab.id" 
            class="nav-item" 
            :class="{ active: activeTab === tab.id }"
            @click="activeTab = tab.id"
          >
            <component :is="tab.icon" :size="18" class="nav-icon" />
            <div class="nav-text">
              <span class="nav-label">{{ tab.name }}</span>
              <span class="nav-sub">{{ tab.sub }}</span>
            </div>
            <span class="method-badge" :class="tab.method">{{ tab.method }}</span>
          </button>
        </nav>
      </aside>

      <!-- 右侧主工作区 Main Workspace -->
      <main class="workspace">
        <!-- 1. 单页智能提取 (Scrape) -->
        <section v-if="activeTab === 'scrape'" class="tab-panel glass-card">
          <div class="panel-head">
            <div>
              <h2 class="panel-title">📄 单页抓取与内容清洗 (Scrape)</h2>
              <p class="panel-desc">极速抓取指定网页并自动清洗转码为精美 Markdown、HTML 或纯文本</p>
            </div>
            <div class="preset-group">
              <span class="preset-label">快速体验:</span>
              <button class="btn-preset" @click="scrapeUrl = 'https://www.xcrawl.com/'">XCrawl</button>
              <button class="btn-preset" @click="scrapeUrl = 'https://www.911proxy.com/'">911Proxy</button>
              <button class="btn-preset" @click="scrapeUrl = 'https://example.com'">Example</button>
            </div>
          </div>

          <div class="input-row">
            <input v-model="scrapeUrl" class="input-field" placeholder="输入目标网页完整 URL (例如: https://www.xcrawl.com/)" />
            <button class="btn-primary" @click="handleScrape" :disabled="loadingScrape">
              <RefreshCw v-if="loadingScrape" class="animate-spin" :size="16" />
              <Play v-else :size="16" />
              <span>{{ loadingScrape ? '极速提取中...' : '开始抓取' }}</span>
            </button>
          </div>

          <div class="options-bar">
            <label class="checkbox-label">
              <input type="checkbox" v-model="scrapeOnlyMain" />
              <span>智能去除广告与导航，只保留主体内容 (onlyMainContent)</span>
            </label>
            <div class="format-select">
              <span class="format-label">输出格式:</span>
              <label class="radio-tag"><input type="radio" value="markdown" v-model="scrapeFormat" /> Markdown</label>
              <label class="radio-tag"><input type="radio" value="html" v-model="scrapeFormat" /> HTML</label>
              <label class="radio-tag"><input type="radio" value="rawHtml" v-model="scrapeFormat" /> Raw DOM</label>
            </div>
          </div>

          <!-- 结果展示 -->
          <div v-if="scrapeResult" class="result-box">
            <div class="result-head">
              <div class="result-tabs">
                <button :class="{ active: scrapeViewTab === 'preview' }" @click="scrapeViewTab = 'preview'">格式化预览</button>
                <button :class="{ active: scrapeViewTab === 'raw' }" @click="scrapeViewTab = 'raw'">文本源码</button>
                <button :class="{ active: scrapeViewTab === 'json' }" @click="scrapeViewTab = 'json'">原始 JSON</button>
              </div>
              <div class="result-actions">
                <button class="btn-secondary" @click="copyContent(scrapeResult.data?.markdown || JSON.stringify(scrapeResult))">
                  <Copy :size="14" /> 复制结果
                </button>
                <button class="btn-secondary" @click="downloadFile(scrapeResult.data?.markdown || '', 'scraped.md')">
                  <Download :size="14" /> 导出 .md
                </button>
              </div>
            </div>

            <div class="result-body">
              <div v-if="scrapeViewTab === 'preview'" class="markdown-view">
                <div class="doc-meta" v-if="scrapeResult.data?.metadata">
                  <div class="meta-item"><span class="meta-k">标题:</span> <span class="meta-v">{{ scrapeResult.data.metadata.title }}</span></div>
                  <div class="meta-item" v-if="scrapeResult.data.metadata.description"><span class="meta-k">描述:</span> <span class="meta-v">{{ scrapeResult.data.metadata.description }}</span></div>
                </div>
                <div class="rendered-content">{{ scrapeResult.data?.markdown }}</div>
              </div>
              <pre v-else-if="scrapeViewTab === 'raw'" class="code-view">{{ scrapeResult.data?.markdown || scrapeResult.data?.html || scrapeResult.data?.rawHtml }}</pre>
              <pre v-else class="code-view">{{ JSON.stringify(scrapeResult, null, 2) }}</pre>
            </div>
          </div>
        </section>

        <!-- 2. 全站递归爬虫 (Crawl) -->
        <section v-if="activeTab === 'crawl'" class="tab-panel glass-card">
          <div class="panel-head">
            <div>
              <h2 class="panel-title">🕷️ 全站/多页面递归爬虫 (Crawl & WS Stream)</h2>
              <p class="panel-desc">广度优先 (BFS) 自动化全站链路发现，支持 WebSocket 实时增量推送</p>
            </div>
          </div>

          <div class="input-grid">
            <div class="input-group flex-2">
              <label>起始 URL</label>
              <input v-model="crawlUrl" class="input-field" placeholder="https://www.xcrawl.com/" />
            </div>
            <div class="input-group">
              <label>爬取上限 (Limit)</label>
              <input v-model.number="crawlLimit" type="number" class="input-field" placeholder="10" />
            </div>
            <div class="input-group">
              <label>最大深度 (Depth)</label>
              <input v-model.number="crawlDepth" type="number" class="input-field" placeholder="2" />
            </div>
            <div class="input-group btn-col">
              <label>&nbsp;</label>
              <button class="btn-primary" @click="handleCreateCrawl" :disabled="loadingCrawl">
                <Play :size="16" /> 启动爬虫
              </button>
            </div>
          </div>

          <!-- 任务状态与实时 WebSocket 流 -->
          <div v-if="crawlJobId" class="stream-card">
            <div class="stream-header">
              <div class="stream-meta">
                <span class="status-badge" :class="crawlStatus?.status || 'scraping'">{{ crawlStatus?.status || 'scraping' }}</span>
                <span class="mono text-muted">ID: {{ crawlJobId }}</span>
                <span v-if="wsConnected" class="badge-ws"><Wifi :size="12" /> WebSocket 实时增量流</span>
              </div>
              <div class="stream-actions">
                <button class="btn-secondary" @click="fetchCrawlStatus"><RefreshCw :size="14" /> 刷新</button>
                <button class="btn-secondary" @click="fetchCrawlErrors"><AlertCircle :size="14" /> 错误分析</button>
                <button class="btn-danger" @click="cancelCrawl"><XCircle :size="14" /> 取消任务</button>
              </div>
            </div>

            <!-- 实时进度条 -->
            <div class="progress-container">
              <div class="progress-bar">
                <div class="progress-fill" :style="{ width: `${Math.min(100, ((crawlStreamDocs.length || crawlStatus?.completed || 0) / (crawlLimit || 1)) * 100)}%` }"></div>
              </div>
              <div class="progress-text">
                <span>已抓取: {{ crawlStreamDocs.length || crawlStatus?.completed || 0 }} / {{ crawlLimit }} 页</span>
                <span>进度: {{ Math.round(((crawlStreamDocs.length || crawlStatus?.completed || 0) / (crawlLimit || 1)) * 100) }}%</span>
              </div>
            </div>

            <!-- 实时收到的页面流列表 -->
            <div class="stream-doc-list">
              <div v-for="(doc, idx) in (crawlStreamDocs.length ? crawlStreamDocs : crawlStatus?.data || [])" :key="idx" class="doc-item-card">
                <div class="doc-item-head">
                  <span class="doc-num">#{{ idx + 1 }}</span>
                  <a :href="doc.metadata?.sourceURL || doc.url" target="_blank" class="doc-url">{{ doc.metadata?.sourceURL || doc.url }}</a>
                  <span class="doc-title">{{ doc.metadata?.title }}</span>
                </div>
                <details class="doc-details">
                  <summary>查看提取的 Markdown 提词 ({{ doc.markdown?.length || 0 }} 字符)</summary>
                  <pre class="doc-markdown-pre">{{ doc.markdown }}</pre>
                </details>
              </div>
            </div>

            <!-- 错误展示框 (如果有) -->
            <div v-if="crawlErrors && crawlErrors.errors?.length" class="error-box margin-top">
              <div class="error-title">⚠️ 爬取过程中记录的异常 ({{ crawlErrors.errors.length }} 条):</div>
              <div v-for="(err, eidx) in crawlErrors.errors" :key="eidx" class="error-item">
                <span class="error-url">{{ err.url }}</span>
                <span class="error-msg">{{ err.error }}</span>
              </div>
            </div>
          </div>
        </section>

        <!-- 3. 批量高并发抓取 (Batch Scrape) -->
        <section v-if="activeTab === 'batch'" class="tab-panel glass-card">
          <div class="panel-head">
            <div>
              <h2 class="panel-title">📦 多 URLs 批量高并发抓取 (Batch Scrape)</h2>
              <p class="panel-desc">一次性提交数百上千个目标网址，Go API 并发协程池极速拉取并归档</p>
            </div>
          </div>

          <div class="form-group">
            <label class="form-label">输入目标 URLs 列表 (每行一条):</label>
            <textarea v-model="batchUrlsText" class="input-field textarea" rows="4" placeholder="https://www.911proxy.com/&#10;https://www.xcrawl.com/&#10;https://example.com"></textarea>
          </div>

          <div class="action-row margin-top">
            <button class="btn-primary" @click="handleCreateBatch" :disabled="loadingBatch">
              <Play :size="16" /> 提交批量任务
            </button>
          </div>

          <div v-if="batchJobId" class="stream-card margin-top">
            <div class="stream-header">
              <div class="stream-meta">
                <span class="status-badge" :class="batchStatus?.status || 'scraping'">{{ batchStatus?.status || 'scraping' }}</span>
                <span class="mono text-muted">ID: {{ batchJobId }}</span>
                <span class="metric-val text-cyan">完成: {{ batchStatus?.completed || 0 }} / {{ batchStatus?.total || 0 }}</span>
              </div>
              <div class="stream-actions">
                <button class="btn-secondary" @click="fetchBatchStatus"><RefreshCw :size="14" /> 刷新状态</button>
                <button class="btn-danger" @click="cancelBatch"><XCircle :size="14" /> 取消</button>
              </div>
            </div>

            <div v-if="batchStatus?.data" class="stream-doc-list">
              <div v-for="(doc, idx) in batchStatus.data" :key="idx" class="doc-item-card">
                <div class="doc-item-head">
                  <span class="doc-num">#{{ idx + 1 }}</span>
                  <span class="doc-url">{{ doc.metadata?.sourceURL || doc.url }}</span>
                  <span class="doc-title">{{ doc.metadata?.title }}</span>
                </div>
                <details class="doc-details">
                  <summary>预览内容</summary>
                  <pre class="doc-markdown-pre">{{ doc.markdown }}</pre>
                </details>
              </div>
            </div>
          </div>
        </section>

        <!-- 4. Sitemap 站点地图拓扑 (Map) -->
        <section v-if="activeTab === 'map'" class="tab-panel glass-card">
          <div class="panel-head">
            <div>
              <h2 class="panel-title">🗺️ Sitemap 站点地图拓扑图谱 (Map)</h2>
              <p class="panel-desc">秒级扫描目标网站的全站拓扑链接，支持精准关键词过滤与全量导出</p>
            </div>
          </div>

          <div class="input-row">
            <input v-model="mapUrl" class="input-field" placeholder="输入目标网站 (例如: https://www.xcrawl.com/)" />
            <input v-model.number="mapLimit" type="number" class="input-field mini" placeholder="数量上限(留空全部)" />
            <button class="btn-primary" @click="handleMap" :disabled="loadingMap">
              <Compass :size="16" /> 导出 Sitemap 拓扑
            </button>
          </div>

          <div v-if="mapResult" class="result-box margin-top">
            <div class="result-head">
              <div class="filter-wrap">
                <Search :size="14" class="input-icon" />
                <input v-model="mapSearch" class="search-mini" placeholder="实时过滤链接 (例如: proxy, blog, api)..." />
              </div>
              <div class="result-actions">
                <span class="count-pill">共计 {{ filteredMapLinks.length }} 个超链接</span>
                <button class="btn-secondary" @click="copyContent(filteredMapLinks.join('\n'))">
                  <Copy :size="14" /> 复制全部
                </button>
              </div>
            </div>

            <div class="links-scroll">
              <div v-for="(link, idx) in filteredMapLinks" :key="idx" class="link-row">
                <span class="link-index">{{ idx + 1 }}</span>
                <a :href="link" target="_blank" class="link-url">{{ link }}</a>
                <ExternalLink :size="12" class="text-dim" />
              </div>
            </div>
          </div>
        </section>

        <!-- 5. 智能搜索聚合 (Search) -->
        <section v-if="activeTab === 'search'" class="tab-panel glass-card">
          <div class="panel-head">
            <div>
              <h2 class="panel-title">🔍 网页智能搜索与多源并发提取 (Search)</h2>
              <p class="panel-desc">全网 SERP 搜索与底层页面实时多协程抓取清洗的一体化聚合接口</p>
            </div>
          </div>

          <div class="input-row">
            <input v-model="searchQuery" class="input-field" placeholder="输入搜索关键词 (例如: 911proxy residential proxy, xcrawl ai scraper)" />
            <input v-model.number="searchLimit" type="number" class="input-field mini" placeholder="数量 (5)" />
            <button class="btn-primary" @click="handleSearch" :disabled="loadingSearch">
              <Search :size="16" /> 开始搜索提取
            </button>
          </div>

          <div v-if="searchResult" class="result-box margin-top">
            <div class="result-head">
              <span class="count-pill text-cyan">成功抓取并解析 {{ searchResult.data?.length || 0 }} 个结果页</span>
            </div>

            <div class="search-list">
              <div v-for="(item, idx) in searchResult.data" :key="idx" class="search-card">
                <div class="search-card-top">
                  <span class="badge-num">#{{ idx + 1 }}</span>
                  <a :href="item.url" target="_blank" class="search-card-title">{{ item.title }}</a>
                </div>
                <p class="search-card-desc">{{ item.description || '无元数据描述摘要' }}</p>
                <div class="search-card-meta">
                  <span class="mono text-dim">{{ item.url }}</span>
                </div>
                <details class="search-details">
                  <summary>展开提取的高清 Markdown 内容</summary>
                  <pre class="search-pre">{{ item.markdown }}</pre>
                </details>
              </div>
            </div>
          </div>
        </section>

        <!-- 6. JSON 结构化提取 (Extract) -->
        <section v-if="activeTab === 'extract'" class="tab-panel glass-card">
          <div class="panel-head">
            <div>
              <h2 class="panel-title">🧩 基于 Prompt 的 JSON 结构化提取 (Extract)</h2>
              <p class="panel-desc">将非结构化网页通过智能规则/大模型直接提炼成类型安全的 JSON 对象</p>
            </div>
          </div>

          <div class="form-group">
            <label class="form-label">目标网页 URLs (每行一条):</label>
            <textarea v-model="extractUrlsText" class="input-field textarea" rows="2" placeholder="https://www.911proxy.com/"></textarea>
          </div>

          <div class="form-group margin-top">
            <label class="form-label">提取需求 Prompt 提示词:</label>
            <input v-model="extractPrompt" class="input-field" placeholder="例如: 提取该代理服务商的价格定价、套餐规格与支持的地区范围" />
          </div>

          <button class="btn-primary margin-top" @click="handleCreateExtract" :disabled="loadingExtract">
            <Play :size="16" /> 提交提取任务
          </button>

          <div v-if="extractJobId" class="stream-card margin-top">
            <div class="stream-header">
              <div class="stream-meta">
                <span class="status-badge" :class="extractStatus?.status || 'scraping'">{{ extractStatus?.status || 'scraping' }}</span>
                <span class="mono text-muted">Job ID: {{ extractJobId }}</span>
              </div>
              <button class="btn-secondary" @click="fetchExtractStatus"><RefreshCw :size="14" /> 刷新提取结果</button>
            </div>

            <div v-if="extractStatus" class="result-box margin-top">
              <pre class="code-view json-view">{{ JSON.stringify(extractStatus.data || extractStatus, null, 2) }}</pre>
            </div>
          </div>
        </section>

        <!-- 7. AI 深度研究研判 (Deep Research) -->
        <section v-if="activeTab === 'deepResearch'" class="tab-panel glass-card">
          <div class="panel-head">
            <div>
              <h2 class="panel-title">🧠 AI 深度研究与智能研判 (Deep Research Agent)</h2>
              <p class="panel-desc">多层深度自主探索智能体：多轮检索、溯源交叉验证、事实沉淀并合成最终研判分析报告</p>
            </div>
          </div>

          <div class="input-grid">
            <div class="input-group flex-2">
              <label>研究课题 / Topic / Query</label>
              <input v-model="researchQuery" class="input-field" placeholder="例如: 2026年 AI 智能体与高并发爬虫技术发展路线分析" />
            </div>
            <div class="input-group">
              <label>研究深度 (Depth 1-7)</label>
              <input v-model.number="researchDepth" type="number" class="input-field" placeholder="3" />
            </div>
            <div class="input-group">
              <label>分析信源上限</label>
              <input v-model.number="researchMaxUrls" type="number" class="input-field" placeholder="10" />
            </div>
            <div class="input-group btn-col">
              <label>&nbsp;</label>
              <button class="btn-primary" @click="handleCreateResearch" :disabled="loadingResearch">
                <Sparkles :size="16" /> 启动研判 Agent
              </button>
            </div>
          </div>

          <div v-if="researchJobId" class="stream-card margin-top">
            <div class="stream-header">
              <div class="stream-meta">
                <span class="status-badge" :class="researchStatus?.status || 'processing'">{{ researchStatus?.status || 'processing' }}</span>
                <span class="mono text-muted">Research ID: {{ researchJobId }}</span>
                <span class="metric-val text-purple">当前探索深度: {{ researchStatus?.currentDepth || 0 }} / {{ researchDepth }}</span>
              </div>
              <button class="btn-secondary" @click="fetchResearchStatus"><RefreshCw :size="14" /> 刷新研判状态</button>
            </div>

            <!-- Agent 行动动态轨迹流 Timeline -->
            <div class="timeline-box" v-if="researchStatus?.activities && researchStatus.activities.length">
              <div class="timeline-title"><Activity :size="14" /> Agent 思考与行动轨迹流 (Activities):</div>
              <div class="timeline-list">
                <div v-for="(act, aidx) in researchStatus.activities" :key="aidx" class="timeline-item">
                  <span class="timeline-dot"></span>
                  <span class="timeline-text">{{ act }}</span>
                </div>
              </div>
            </div>

            <!-- 最终综合研判报告 -->
            <div v-if="researchStatus?.data?.finalAnalysis" class="result-box margin-top">
              <div class="result-head">
                <span class="count-pill text-purple">📑 最终综合研判分析报告</span>
                <div class="result-actions">
                  <button class="btn-secondary" @click="copyContent(researchStatus.data.finalAnalysis)">
                    <Copy :size="14" /> 复制报告
                  </button>
                  <button class="btn-secondary" @click="downloadFile(researchStatus.data.finalAnalysis, 'research-report.md')">
                    <Download :size="14" /> 导出报告
                  </button>
                </div>
              </div>
              <div class="result-body markdown-view">
                <pre class="rendered-content">{{ researchStatus.data.finalAnalysis }}</pre>
              </div>
            </div>
          </div>
        </section>

        <!-- 8. LLMs.txt 站点生成 (LLMs.txt) -->
        <section v-if="activeTab === 'llmstxt'" class="tab-panel glass-card">
          <div class="panel-head">
            <div>
              <h2 class="panel-title">📜 LLMs.txt 站点标准化大纲生成 (LLMs.txt)</h2>
              <p class="panel-desc">自动解析目标站点并生成符合标准规范的 /llms.txt 导航索引与 /llms-full.txt 完整知识库</p>
            </div>
          </div>

          <div class="input-row">
            <input v-model="llmsTxtUrl" class="input-field" placeholder="输入文档站或官方网站 URL (例如: https://www.xcrawl.com/)" />
            <button class="btn-primary" @click="handleCreateLLMsTxt" :disabled="loadingLLMsTxt">
              <BookOpen :size="16" /> 生成 LLMs.txt
            </button>
          </div>

          <div class="options-bar">
            <label class="checkbox-label">
              <input type="checkbox" v-model="llmsTxtFull" />
              <span>同时生成完整版 Markdown 内容 (showFullText / llms-full.txt)</span>
            </label>
          </div>

          <div v-if="llmsTxtJobId" class="stream-card margin-top">
            <div class="stream-header">
              <div class="stream-meta">
                <span class="status-badge" :class="llmsTxtStatus?.status || 'processing'">{{ llmsTxtStatus?.status || 'processing' }}</span>
                <span class="mono text-muted">ID: {{ llmsTxtJobId }}</span>
              </div>
              <button class="btn-secondary" @click="fetchLLMsTxtStatus"><RefreshCw :size="14" /> 刷新状态</button>
            </div>

            <div v-if="llmsTxtStatus?.data" class="result-box margin-top">
              <div class="result-head">
                <span class="count-pill text-cyan">生成的标准 llms.txt 内容</span>
                <button class="btn-secondary" @click="copyContent(llmsTxtStatus.data.llmstxt)"><Copy :size="14" /> 复制</button>
              </div>
              <pre class="code-view">{{ llmsTxtStatus.data.llmstxt }}</pre>
            </div>
          </div>
        </section>

        <!-- 9. 系统监控与配额面板 (System Monitor) -->
        <section v-if="activeTab === 'system'" class="tab-panel glass-card">
          <div class="panel-head">
            <div>
              <h2 class="panel-title">📊 引擎监控、配额账期与系统状态</h2>
              <p class="panel-desc">实时观测 Go API 网关并发限制、Redis 状态持久化、RabbitMQ 积压与 Token 用量</p>
            </div>
            <button class="btn-secondary" @click="loadSystemStats"><RefreshCw :size="14" /> 刷新全部指标</button>
          </div>

          <div class="stats-cards-grid">
            <div class="stat-card glass-card">
              <div class="stat-card-head">
                <span class="stat-card-title">并发限制与占用 (Concurrency)</span>
                <Cpu class="text-indigo" :size="20" />
              </div>
              <div class="stat-card-val text-indigo">{{ systemStats.concurrency }} <span class="stat-sub">/ {{ systemStats.maxConcurrency }} Max</span></div>
              <div class="stat-card-desc">基于 Redis 滑动窗口精准并发限流算法</div>
            </div>

            <div class="stat-card glass-card">
              <div class="stat-card-head">
                <span class="stat-card-title">剩余 Credit 积分额度</span>
                <Database class="text-emerald" :size="20" />
              </div>
              <div class="stat-card-val text-emerald">{{ formatNumber(systemStats.remainingCredits) }}</div>
              <div class="stat-card-desc">当前账期配额总量: 500,000</div>
            </div>

            <div class="stat-card glass-card">
              <div class="stat-card-head">
                <span class="stat-card-title">LLM Token 消耗额度</span>
                <Sparkles class="text-purple" :size="20" />
              </div>
              <div class="stat-card-val text-purple">{{ formatNumber(systemStats.remainingTokens) }}</div>
              <div class="stat-card-desc">支持多模型 AI 提取与深度研判</div>
            </div>

            <div class="stat-card glass-card">
              <div class="stat-card-head">
                <span class="stat-card-title">RabbitMQ 任务队列积压</span>
                <Layers class="text-cyan" :size="20" />
              </div>
              <div class="stat-card-val text-cyan">{{ systemStats.jobsInQueue }}</div>
              <div class="stat-card-desc">异步爬取与批量抓取任务消费顺畅</div>
            </div>
          </div>
        </section>
      </main>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import {
  Zap, Globe, ShieldCheck, Cpu, Database, Play, RefreshCw, Copy, Download,
  ExternalLink, Layers, Search, BookOpen, Sparkles, Activity, AlertCircle,
  XCircle, Wifi, Compass
} from 'lucide-vue-next'

const baseUrl = ref('http://localhost:3002')
const apiKey = ref('fc-test-key')

const healthOk = ref(false)
const healthMs = ref(0)
const loadingHealth = ref(false)

const activeTab = ref('scrape')

const tabs = [
  { id: 'scrape', name: '单页智能提取', sub: 'Markdown / HTML 转码', method: 'POST', icon: Zap },
  { id: 'crawl', name: '全站递归爬虫', sub: 'BFS 拓扑与实时 WS 流', method: 'POST', icon: Layers },
  { id: 'batch', name: '批量多源抓取', sub: '并发批量 URL 调度', method: 'POST', icon: Database },
  { id: 'map', name: 'Sitemap 拓扑', sub: '秒级站点超链接扫描', method: 'POST', icon: Compass },
  { id: 'search', name: '智能搜索聚合', sub: 'SERP 全网检索与提取', method: 'POST', icon: Search },
  { id: 'extract', name: 'JSON 结构化提取', sub: 'Schema / Prompt 提炼', method: 'POST', icon: Sparkles },
  { id: 'deepResearch', name: 'AI 深度研究研判', sub: '多轮深度研判 Agent', method: 'POST', icon: Activity },
  { id: 'llmstxt', name: 'LLMs.txt 站点生成', sub: '标准化 LLM 知识库', method: 'POST', icon: BookOpen },
  { id: 'system', name: '系统监控与配额', sub: '并发 / 队列 / 性能', method: 'GET', icon: Cpu },
]

// 1. Scrape
const scrapeUrl = ref('https://www.xcrawl.com/')
const scrapeOnlyMain = ref(true)
const scrapeFormat = ref('markdown')
const loadingScrape = ref(false)
const scrapeResult = ref(null)
const scrapeViewTab = ref('preview')

// 2. Crawl
const crawlUrl = ref('https://www.xcrawl.com/')
const crawlLimit = ref(5)
const crawlDepth = ref(2)
const loadingCrawl = ref(false)
const crawlJobId = ref('')
const crawlStatus = ref(null)
const crawlErrors = ref(null)
const crawlStreamDocs = ref([])
const wsConnected = ref(false)
let activeWs = null

// 3. Batch
const batchUrlsText = ref("https://www.911proxy.com/\nhttps://www.xcrawl.com/")
const loadingBatch = ref(false)
const batchJobId = ref('')
const batchStatus = ref(null)

// 4. Map
const mapUrl = ref('https://www.xcrawl.com/')
const mapLimit = ref(null)
const mapSearch = ref('')
const loadingMap = ref(false)
const mapResult = ref(null)

const filteredMapLinks = computed(() => {
  if (!mapResult.value?.links) return []
  if (!mapSearch.value) return mapResult.value.links
  return mapResult.value.links.filter(l => l.toLowerCase().includes(mapSearch.value.toLowerCase()))
})

// 5. Search
const searchQuery = ref('911proxy residential proxy')
const searchLimit = ref(5)
const loadingSearch = ref(false)
const searchResult = ref(null)

// 6. Extract
const extractUrlsText = ref('https://www.911proxy.com/')
const extractPrompt = ref('提取页面上的价格套餐与支持的代理类型')
const loadingExtract = ref(false)
const extractJobId = ref('')
const extractStatus = ref(null)

// 7. Deep Research
const researchQuery = ref('2026年 AI Agent 架构演进与高并发抓取')
const researchDepth = ref(3)
const researchMaxUrls = ref(10)
const loadingResearch = ref(false)
const researchJobId = ref('')
const researchStatus = ref(null)

// 8. LLMs.txt
const llmsTxtUrl = ref('https://www.xcrawl.com/')
const llmsTxtFull = ref(true)
const loadingLLMsTxt = ref(false)
const llmsTxtJobId = ref('')
const llmsTxtStatus = ref(null)

// 9. System Stats
const systemStats = ref({
  concurrency: 0,
  maxConcurrency: 10,
  remainingCredits: 500000,
  remainingTokens: 7500000,
  jobsInQueue: 0
})

// Methods
const formatNumber = (num) => {
  if (!num) return '0'
  return num.toLocaleString()
}

const checkHealth = async () => {
  loadingHealth.value = true
  const start = Date.now()
  try {
    const res = await fetch(`${baseUrl.value}/health`)
    const data = await res.json()
    healthMs.value = Date.now() - start
    healthOk.value = (res.status === 200 && data.status === 'ok')
  } catch (err) {
    healthOk.value = false
  } finally {
    loadingHealth.value = false
  }
}

const loadSystemStats = async () => {
  try {
    const [cRes, crRes, tRes, qRes] = await Promise.all([
      fetch(`${baseUrl.value}/v1/concurrency-check`, { headers: { 'Authorization': `Bearer ${apiKey.value}` } }),
      fetch(`${baseUrl.value}/v1/team/credit-usage`, { headers: { 'Authorization': `Bearer ${apiKey.value}` } }),
      fetch(`${baseUrl.value}/v1/team/token-usage`, { headers: { 'Authorization': `Bearer ${apiKey.value}` } }),
      fetch(`${baseUrl.value}/v1/team/queue-status`, { headers: { 'Authorization': `Bearer ${apiKey.value}` } }),
    ])
    if (cRes.ok) {
      const c = await cRes.json()
      systemStats.value.concurrency = c.concurrency || 0
      systemStats.value.maxConcurrency = c.maxConcurrency || 10
    }
    if (crRes.ok) {
      const cr = await crRes.json()
      systemStats.value.remainingCredits = cr.data?.remaining_credits || 500000
    }
    if (tRes.ok) {
      const t = await tRes.json()
      systemStats.value.remainingTokens = t.data?.remaining_tokens || 7500000
    }
    if (qRes.ok) {
      const q = await qRes.json()
      systemStats.value.jobsInQueue = q.jobsInQueue || 0
    }
  } catch (e) {
    console.error("加载系统统计失败:", e)
  }
}

const handleScrape = async () => {
  if (!scrapeUrl.value) return alert('请输入抓取网址')
  loadingScrape.value = true
  scrapeResult.value = null
  try {
    const res = await fetch(`${baseUrl.value}/v1/scrape`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${apiKey.value}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        url: scrapeUrl.value,
        onlyMainContent: scrapeOnlyMain.value,
        formats: [scrapeFormat.value]
      })
    })
    scrapeResult.value = await res.json()
  } catch (err) {
    alert('抓取失败: ' + err.message)
  } finally {
    loadingScrape.value = false
  }
}

const handleCreateCrawl = async () => {
  if (!crawlUrl.value) return alert('请输入爬虫起始网址')
  loadingCrawl.value = true
  crawlStatus.value = null
  crawlErrors.value = null
  crawlStreamDocs.value = []
  if (activeWs) activeWs.close()

  try {
    const res = await fetch(`${baseUrl.value}/v1/crawl`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${apiKey.value}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        url: crawlUrl.value,
        limit: crawlLimit.value || 10,
        maxDepth: crawlDepth.value || 2
      })
    })
    const data = await res.json()
    if (data.id) {
      crawlJobId.value = data.id
      connectCrawlWS(data.id)
      fetchCrawlStatus()
    }
  } catch (err) {
    alert('创建爬虫任务失败: ' + err.message)
  } finally {
    loadingCrawl.value = false
  }
}

const connectCrawlWS = (id) => {
  const wsUrl = baseUrl.value.replace(/^http/, 'ws') + `/v1/crawl/${id}/ws`
  try {
    activeWs = new WebSocket(wsUrl)
    activeWs.onopen = () => { wsConnected.value = true }
    activeWs.onclose = () => { wsConnected.value = false }
    activeWs.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data)
        if (msg.type === 'document' && msg.data) {
          crawlStreamDocs.value.push(msg.data)
        } else if (msg.type === 'done') {
          fetchCrawlStatus()
        }
      } catch (e) {}
    }
  } catch (err) {
    console.warn("WebSocket 连接不可用，回退至 HTTP 轮询", err)
  }
}

const fetchCrawlStatus = async () => {
  if (!crawlJobId.value) return
  try {
    const res = await fetch(`${baseUrl.value}/v1/crawl/${crawlJobId.value}`, {
      headers: { 'Authorization': `Bearer ${apiKey.value}` }
    })
    crawlStatus.value = await res.json()
  } catch (err) {
    console.error(err)
  }
}

const fetchCrawlErrors = async () => {
  if (!crawlJobId.value) return
  try {
    const res = await fetch(`${baseUrl.value}/v1/crawl/${crawlJobId.value}/errors`, {
      headers: { 'Authorization': `Bearer ${apiKey.value}` }
    })
    crawlErrors.value = await res.json()
  } catch (err) {
    console.error(err)
  }
}

const cancelCrawl = async () => {
  if (!crawlJobId.value) return
  try {
    await fetch(`${baseUrl.value}/v1/crawl/${crawlJobId.value}`, {
      method: 'DELETE',
      headers: { 'Authorization': `Bearer ${apiKey.value}` }
    })
    fetchCrawlStatus()
  } catch (err) {
    console.error(err)
  }
}

const handleCreateBatch = async () => {
  const urls = batchUrlsText.value.split('\n').map(s => s.trim()).filter(Boolean)
  if (!urls.length) return alert('请输入至少一条 URL')
  loadingBatch.value = true
  batchStatus.value = null
  try {
    const res = await fetch(`${baseUrl.value}/v1/batch/scrape`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${apiKey.value}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ urls })
    })
    const data = await res.json()
    if (data.id) {
      batchJobId.value = data.id
      fetchBatchStatus()
    }
  } catch (err) {
    alert('提交批量任务失败: ' + err.message)
  } finally {
    loadingBatch.value = false
  }
}

const fetchBatchStatus = async () => {
  if (!batchJobId.value) return
  try {
    const res = await fetch(`${baseUrl.value}/v1/batch/scrape/${batchJobId.value}`, {
      headers: { 'Authorization': `Bearer ${apiKey.value}` }
    })
    batchStatus.value = await res.json()
  } catch (err) {
    console.error(err)
  }
}

const cancelBatch = async () => {
  if (!batchJobId.value) return
  try {
    await fetch(`${baseUrl.value}/v1/batch/scrape/${batchJobId.value}`, {
      method: 'DELETE',
      headers: { 'Authorization': `Bearer ${apiKey.value}` }
    })
    fetchBatchStatus()
  } catch (err) {
    console.error(err)
  }
}

const handleMap = async () => {
  if (!mapUrl.value) return alert('请输入目标网址')
  loadingMap.value = true
  mapResult.value = null
  try {
    const body = { url: mapUrl.value }
    if (mapLimit.value && mapLimit.value > 0) body.limit = mapLimit.value
    const res = await fetch(`${baseUrl.value}/v1/map`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${apiKey.value}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(body)
    })
    mapResult.value = await res.json()
  } catch (err) {
    alert('导出 Sitemap 失败: ' + err.message)
  } finally {
    loadingMap.value = false
  }
}

const handleSearch = async () => {
  if (!searchQuery.value) return alert('请输入搜索关键词')
  loadingSearch.value = true
  searchResult.value = null
  try {
    const res = await fetch(`${baseUrl.value}/v1/search`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${apiKey.value}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        query: searchQuery.value,
        limit: searchLimit.value || 5
      })
    })
    searchResult.value = await res.json()
  } catch (err) {
    alert('搜索提取失败: ' + err.message)
  } finally {
    loadingSearch.value = false
  }
}

const handleCreateExtract = async () => {
  const urls = extractUrlsText.value.split('\n').map(s => s.trim()).filter(Boolean)
  if (!urls.length) return alert('请至少填写一条目标 URL')
  loadingExtract.value = true
  extractStatus.value = null
  try {
    const res = await fetch(`${baseUrl.value}/v1/extract`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${apiKey.value}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        urls,
        prompt: extractPrompt.value
      })
    })
    const data = await res.json()
    if (data.id) {
      extractJobId.value = data.id
      fetchExtractStatus()
    }
  } catch (err) {
    alert('提交提取失败: ' + err.message)
  } finally {
    loadingExtract.value = false
  }
}

const fetchExtractStatus = async () => {
  if (!extractJobId.value) return
  try {
    const res = await fetch(`${baseUrl.value}/v1/extract/${extractJobId.value}`, {
      headers: { 'Authorization': `Bearer ${apiKey.value}` }
    })
    extractStatus.value = await res.json()
  } catch (err) {
    console.error(err)
  }
}

const handleCreateResearch = async () => {
  if (!researchQuery.value) return alert('请输入研究主题')
  loadingResearch.value = true
  researchStatus.value = null
  try {
    const res = await fetch(`${baseUrl.value}/v1/deep-research`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${apiKey.value}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        query: researchQuery.value,
        maxDepth: researchDepth.value || 3,
        maxUrls: researchMaxUrls.value || 10
      })
    })
    const data = await res.json()
    if (data.id) {
      researchJobId.value = data.id
      fetchResearchStatus()
    }
  } catch (err) {
    alert('启动研判 Agent 失败: ' + err.message)
  } finally {
    loadingResearch.value = false
  }
}

const fetchResearchStatus = async () => {
  if (!researchJobId.value) return
  try {
    const res = await fetch(`${baseUrl.value}/v1/deep-research/${researchJobId.value}`, {
      headers: { 'Authorization': `Bearer ${apiKey.value}` }
    })
    researchStatus.value = await res.json()
  } catch (err) {
    console.error(err)
  }
}

const handleCreateLLMsTxt = async () => {
  if (!llmsTxtUrl.value) return alert('请输入文档站 URL')
  loadingLLMsTxt.value = true
  llmsTxtStatus.value = null
  try {
    const res = await fetch(`${baseUrl.value}/v1/llmstxt`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${apiKey.value}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        url: llmsTxtUrl.value,
        showFullText: llmsTxtFull.value
      })
    })
    const data = await res.json()
    if (data.id) {
      llmsTxtJobId.value = data.id
      fetchLLMsTxtStatus()
    }
  } catch (err) {
    alert('生成 LLMs.txt 失败: ' + err.message)
  } finally {
    loadingLLMsTxt.value = false
  }
}

const fetchLLMsTxtStatus = async () => {
  if (!llmsTxtJobId.value) return
  try {
    const res = await fetch(`${baseUrl.value}/v1/llmstxt/${llmsTxtJobId.value}`, {
      headers: { 'Authorization': `Bearer ${apiKey.value}` }
    })
    llmsTxtStatus.value = await res.json()
  } catch (err) {
    console.error(err)
  }
}

const copyContent = (text) => {
  if (!text) return
  navigator.clipboard.writeText(text)
  alert('已成功复制到剪贴板！')
}

const downloadFile = (content, filename) => {
  if (!content) return
  const blob = new Blob([content], { type: 'text/markdown;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

onMounted(() => {
  checkHealth()
  loadSystemStats()
})
</script>

<style scoped>
.studio-layout {
  max-width: 1440px;
  margin: 0 auto;
  padding: 20px;
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.top-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
}

.brand-section {
  display: flex;
  align-items: center;
  gap: 14px;
}

.brand-logo {
  background: rgba(6, 182, 212, 0.12);
  border: 1px solid rgba(6, 182, 212, 0.3);
  padding: 8px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.brand-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.brand-name {
  font-size: 20px;
  font-weight: 800;
  letter-spacing: -0.5px;
}

.badge-tag {
  font-size: 11px;
  font-weight: 700;
  background: linear-gradient(135deg, var(--accent-indigo), var(--accent-cyan));
  color: white;
  padding: 2px 8px;
  border-radius: 6px;
}

.badge-speed {
  font-size: 11px;
  background: rgba(16, 185, 129, 0.15);
  color: var(--accent-emerald);
  border: 1px solid rgba(16, 185, 129, 0.3);
  padding: 2px 6px;
  border-radius: 6px;
  font-weight: 600;
}

.brand-desc {
  font-size: 12px;
  color: var(--text-muted);
  margin-top: 2px;
}

.top-actions {
  display: flex;
  align-items: center;
  gap: 16px;
}

.metrics-capsules {
  display: flex;
  gap: 8px;
}

.metric-pill {
  display: flex;
  align-items: center;
  gap: 6px;
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid var(--border-color);
  padding: 6px 12px;
  border-radius: 20px;
  font-size: 12px;
}

.metric-val {
  font-weight: 700;
}

.config-box {
  display: flex;
  align-items: center;
  gap: 8px;
}

.config-input-wrap {
  position: relative;
  display: flex;
  align-items: center;
}

.input-icon {
  position: absolute;
  left: 10px;
  color: var(--text-dim);
}

.config-input {
  background: var(--bg-input);
  border: 1px solid var(--border-color);
  color: var(--text-main);
  padding: 6px 10px 6px 30px;
  border-radius: 8px;
  font-size: 12px;
  outline: none;
  width: 170px;
}

.config-input.key {
  width: 120px;
}

.health-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  background: var(--bg-input);
  border: 1px solid var(--border-color);
  color: var(--text-main);
  padding: 6px 12px;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
}

.bg-emerald { background: var(--accent-emerald); }
.bg-rose { background: var(--accent-rose); }
.text-cyan { color: var(--accent-cyan); }
.text-indigo { color: var(--accent-indigo); }
.text-emerald { color: var(--accent-emerald); }
.text-purple { color: var(--accent-purple); }

.main-body {
  display: grid;
  grid-template-columns: 280px 1fr;
  gap: 20px;
}

.sidebar {
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  height: fit-content;
}

.sidebar-title {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--text-dim);
  padding: 4px 12px;
  font-weight: 700;
}

.nav-menu {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 10px;
  background: transparent;
  border: 1px solid transparent;
  color: var(--text-muted);
  cursor: pointer;
  text-align: left;
  transition: all 0.2s;
}

.nav-item:hover {
  background: var(--bg-card-hover);
  color: var(--text-main);
}

.nav-item.active {
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.15), rgba(6, 182, 212, 0.15));
  border-color: rgba(6, 182, 212, 0.4);
  color: #ffffff;
  box-shadow: 0 4px 14px var(--glow-cyan);
}

.nav-text {
  flex: 1;
}

.nav-label {
  display: block;
  font-size: 13px;
  font-weight: 600;
}

.nav-sub {
  display: block;
  font-size: 10px;
  color: var(--text-dim);
}

.method-badge {
  font-size: 9px;
  font-weight: 800;
  padding: 2px 5px;
  border-radius: 4px;
}

.method-badge.POST {
  background: rgba(99, 102, 241, 0.2);
  color: var(--accent-indigo);
}

.method-badge.GET {
  background: rgba(16, 185, 129, 0.2);
  color: var(--accent-emerald);
}

.workspace {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.tab-panel {
  padding: 24px;
}

.panel-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 20px;
}

.panel-title {
  font-size: 18px;
  font-weight: 700;
}

.panel-desc {
  font-size: 13px;
  color: var(--text-muted);
  margin-top: 4px;
}

.preset-group {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
}

.preset-label {
  color: var(--text-dim);
}

.btn-preset {
  background: var(--bg-input);
  border: 1px solid var(--border-color);
  color: var(--accent-cyan);
  padding: 3px 8px;
  border-radius: 6px;
  font-size: 11px;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-preset:hover {
  background: rgba(6, 182, 212, 0.15);
  border-color: var(--accent-cyan);
}

.input-row {
  display: flex;
  gap: 10px;
}

.input-row .input-field.mini {
  width: 180px;
}

.input-grid {
  display: flex;
  gap: 12px;
}

.input-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 13px;
  color: var(--text-muted);
  flex: 1;
}

.input-group.flex-2 { flex: 2; }
.input-group.btn-col { flex: 0.8; justify-content: flex-end; }

.options-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 14px;
  font-size: 13px;
  color: var(--text-muted);
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.format-select {
  display: flex;
  align-items: center;
  gap: 10px;
}

.radio-tag {
  display: flex;
  align-items: center;
  gap: 4px;
  cursor: pointer;
}

.result-box {
  margin-top: 20px;
  background: var(--bg-input);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  overflow: hidden;
}

.result-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 16px;
  background: rgba(255, 255, 255, 0.02);
  border-bottom: 1px solid var(--border-color);
}

.result-tabs {
  display: flex;
  gap: 6px;
}

.result-tabs button {
  background: transparent;
  border: none;
  color: var(--text-dim);
  padding: 5px 10px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
}

.result-tabs button.active {
  background: rgba(255, 255, 255, 0.08);
  color: var(--text-main);
}

.result-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.result-body {
  padding: 16px;
  max-height: 520px;
  overflow-y: auto;
}

.markdown-view {
  font-size: 14px;
  line-height: 1.6;
}

.doc-meta {
  background: rgba(99, 102, 241, 0.08);
  border-left: 3px solid var(--accent-indigo);
  padding: 10px 14px;
  border-radius: 6px;
  margin-bottom: 16px;
  font-size: 13px;
}

.meta-k {
  color: var(--text-dim);
  font-weight: 600;
}

.meta-v {
  color: var(--text-main);
  margin-left: 6px;
}

.code-view {
  font-size: 13px;
  color: #e2e8f0;
  white-space: pre-wrap;
  word-break: break-all;
}

.rendered-content {
  white-space: pre-wrap;
  font-family: inherit;
}

/* Stream / Crawl Cards */
.stream-card {
  background: var(--bg-input);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 16px;
}

.stream-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 14px;
}

.stream-meta {
  display: flex;
  align-items: center;
  gap: 10px;
}

.status-badge {
  font-size: 11px;
  text-transform: uppercase;
  font-weight: 800;
  padding: 3px 8px;
  border-radius: 6px;
  background: rgba(6, 182, 212, 0.15);
  color: var(--accent-cyan);
}

.status-badge.completed { background: rgba(16, 185, 129, 0.15); color: var(--accent-emerald); }
.status-badge.cancelled { background: rgba(244, 63, 94, 0.15); color: var(--accent-rose); }

.badge-ws {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: var(--accent-emerald);
  background: rgba(16, 185, 129, 0.1);
  padding: 2px 6px;
  border-radius: 4px;
}

.progress-container {
  margin-bottom: 16px;
}

.progress-bar {
  height: 6px;
  background: rgba(255, 255, 255, 0.08);
  border-radius: 3px;
  overflow: hidden;
  margin-bottom: 6px;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--accent-indigo), var(--accent-cyan));
  transition: width 0.3s ease;
}

.progress-text {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
  color: var(--text-dim);
}

.stream-doc-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 400px;
  overflow-y: auto;
}

.doc-item-card {
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 10px 14px;
}

.doc-item-head {
  display: flex;
  align-items: center;
  gap: 8px;
}

.doc-num {
  font-size: 11px;
  font-weight: 800;
  color: var(--accent-cyan);
}

.doc-url {
  font-size: 12px;
  color: var(--text-main);
  text-decoration: none;
  font-weight: 600;
}

.doc-url:hover { text-decoration: underline; }

.doc-title {
  font-size: 12px;
  color: var(--text-dim);
  margin-left: auto;
}

.doc-details {
  margin-top: 6px;
  font-size: 12px;
  color: var(--text-muted);
}

.doc-details summary { cursor: pointer; }
.doc-markdown-pre {
  margin-top: 6px;
  background: #030712;
  padding: 10px;
  border-radius: 6px;
  font-size: 11px;
  white-space: pre-wrap;
}

.margin-top { margin-top: 16px; }

/* Timeline for Deep Research */
.timeline-box {
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 12px 16px;
  margin-top: 14px;
}

.timeline-title {
  font-size: 12px;
  font-weight: 700;
  color: var(--accent-purple);
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 10px;
}

.timeline-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.timeline-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--text-main);
}

.timeline-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--accent-purple);
}

/* System Stats Grid */
.stats-cards-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
  margin-top: 16px;
}

.stat-card {
  padding: 20px;
}

.stat-card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.stat-card-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-muted);
}

.stat-card-val {
  font-size: 32px;
  font-weight: 800;
  margin: 10px 0 6px 0;
  letter-spacing: -1px;
}

.stat-sub {
  font-size: 14px;
  color: var(--text-dim);
  font-weight: normal;
}

.stat-card-desc {
  font-size: 12px;
  color: var(--text-dim);
}

.count-pill {
  font-size: 12px;
  background: rgba(255, 255, 255, 0.05);
  padding: 4px 10px;
  border-radius: 20px;
  font-weight: 600;
}

.links-scroll {
  max-height: 420px;
  overflow-y: auto;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.link-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  background: rgba(255, 255, 255, 0.02);
  border-radius: 6px;
  font-size: 13px;
}

.link-index {
  color: var(--text-dim);
  font-size: 11px;
  width: 24px;
}

.link-url {
  color: var(--accent-cyan);
  text-decoration: none;
  flex: 1;
}

.link-url:hover { text-decoration: underline; }

.search-mini {
  background: transparent;
  border: none;
  color: var(--text-main);
  font-size: 12px;
  outline: none;
  width: 220px;
}

.filter-wrap {
  display: flex;
  align-items: center;
  gap: 6px;
  background: var(--bg-input);
  border: 1px solid var(--border-color);
  padding: 4px 10px;
  border-radius: 6px;
}

.search-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
}

.search-card {
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 14px;
}

.search-card-top {
  display: flex;
  align-items: center;
  gap: 8px;
}

.badge-num {
  font-size: 11px;
  font-weight: 800;
  color: var(--accent-cyan);
}

.search-card-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--text-main);
  text-decoration: none;
}

.search-card-desc {
  font-size: 13px;
  color: var(--text-muted);
  margin: 6px 0;
}

.search-details {
  margin-top: 10px;
  font-size: 12px;
  color: var(--accent-cyan);
  cursor: pointer;
}

.search-pre {
  margin-top: 6px;
  background: #030712;
  padding: 12px;
  border-radius: 6px;
  font-size: 12px;
  white-space: pre-wrap;
  color: var(--text-main);
}
</style>
