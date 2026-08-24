#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Firecrawl Go API 100 超高并发全接口 (P0 + P1 8大端点) 自动化压测脚本
针对目标网站 (https://www.911proxy.com/ 与 https://www.xcrawl.com/) 进行极限高并发吞吐测试。
"""

import json
import time
import urllib.request
import urllib.error
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import dataclass
from typing import List, Dict, Any

# ---------------- 配置区域 ----------------
API_BASE_URL = "http://localhost:3005"
API_KEY = "fc-test-key"
TARGET_URLS = [
    "https://www.911proxy.com/",
    "https://www.xcrawl.com/"
]

# 超高并发配置
CONCURRENCY = 500    # 500 线程极限并发
TEST_ROUNDS = 5      # 重复测试轮次

# 禁用 HTTP 本地代理
proxy_handler = urllib.request.ProxyHandler({})
opener = urllib.request.build_opener(proxy_handler)
urllib.request.install_opener(opener)

@dataclass
class StatResult:
    endpoint: str
    target_url: str
    status_code: int
    duration_ms: float
    success: bool
    error_msg: str = ""

class FirecrawlBenchmark:
    def __init__(self, base_url: str, api_key: str):
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.results: List[StatResult] = []

    def _send_request(self, method: str, path: str, payload: Dict[str, Any] = None) -> (int, Dict[str, Any], float):
        url = f"{self.base_url}{path}"
        headers = {
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json"
        }
        data = json.dumps(payload).encode('utf-8') if payload else None

        req = urllib.request.Request(url, data=data, headers=headers, method=method)
        start_time = time.time()
        try:
            with urllib.request.urlopen(req, timeout=30) as resp:
                elapsed_ms = (time.time() - start_time) * 1000
                res_body = json.loads(resp.read().decode('utf-8'))
                return resp.status, res_body, elapsed_ms
        except urllib.error.HTTPError as e:
            elapsed_ms = (time.time() - start_time) * 1000
            try:
                body = json.loads(e.read().decode('utf-8'))
            except:
                body = {}
            return e.code, body, elapsed_ms
        except Exception as e:
            elapsed_ms = (time.time() - start_time) * 1000
            return 500, {"error": str(e)}, elapsed_ms

    def test_health(self) -> StatResult:
        status, body, ms = self._send_request("GET", "/health")
        ok = (status == 200 and body.get("status") == "ok")
        return StatResult("GET /health", "N/A", status, ms, ok)

    def test_scrape(self, target_url: str) -> StatResult:
        payload = {"url": target_url, "formats": ["markdown"], "onlyMainContent": True}
        status, body, ms = self._send_request("POST", "/v1/scrape", payload)
        ok = (status == 200 and body.get("success") is True)
        return StatResult("POST /v1/scrape", target_url, status, ms, ok, body.get("error", ""))

    def test_map(self, target_url: str) -> StatResult:
        payload = {"url": target_url, "limit": 10}
        status, body, ms = self._send_request("POST", "/v1/map", payload)
        ok = (status == 200 and body.get("success") is True)
        return StatResult("POST /v1/map", target_url, status, ms, ok, body.get("error", ""))

    def test_crawl_pipeline(self, target_url: str) -> List[StatResult]:
        results = []
        payload = {"url": target_url, "limit": 2, "maxDepth": 1}
        status, body, ms = self._send_request("POST", "/v1/crawl", payload)
        ok = (status == 200 and body.get("success") is True)
        results.append(StatResult("POST /v1/crawl", target_url, status, ms, ok, body.get("error", "")))
        
        if ok and "id" in body:
            job_id = body["id"]
            st, b, ms2 = self._send_request("GET", f"/v1/crawl/{job_id}")
            results.append(StatResult("GET /v1/crawl/:id", target_url, st, ms2, st == 200))
            st, b, ms3 = self._send_request("DELETE", f"/v1/crawl/{job_id}")
            results.append(StatResult("DELETE /v1/crawl/:id", target_url, st, ms3, st == 200))
        return results

    def test_batch_pipeline(self, target_url: str) -> List[StatResult]:
        results = []
        payload = {"urls": [target_url], "options": {"onlyMainContent": True}}
        status, body, ms = self._send_request("POST", "/v1/batch/scrape", payload)
        ok = (status == 200 and body.get("success") is True)
        results.append(StatResult("POST /v1/batch/scrape", target_url, status, ms, ok, body.get("error", "")))

        if ok and "id" in body:
            batch_id = body["id"]
            st, b, ms2 = self._send_request("GET", f"/v1/batch/scrape/{batch_id}")
            results.append(StatResult("GET /v1/batch/scrape/:id", target_url, st, ms2, st == 200))
            st, b, ms3 = self._send_request("DELETE", f"/v1/batch/scrape/{batch_id}")
            results.append(StatResult("DELETE /v1/batch/scrape/:id", target_url, st, ms3, st == 200))
        return results

    def run_benchmark(self):
        print(f"🚀 开始 Firecrawl Go API 500 极限并发全端点压测...")
        print(f"📍 目标地址: {self.base_url}")
        print(f"🌐 目标测试网站: {', '.join(TARGET_URLS)}")
        print(f"🔥 并发线程数: {CONCURRENCY} | 每组重复轮次: {TEST_ROUNDS}\n" + "="*75)

        start_all = time.time()
        tasks = []

        with ThreadPoolExecutor(max_workers=CONCURRENCY) as executor:
            for _ in range(TEST_ROUNDS):
                tasks.append(executor.submit(self.test_health))

            for target_url in TARGET_URLS:
                for _ in range(TEST_ROUNDS):
                    tasks.append(executor.submit(self.test_scrape, target_url))
                    tasks.append(executor.submit(self.test_map, target_url))
                    tasks.append(executor.submit(self.test_crawl_pipeline, target_url))
                    tasks.append(executor.submit(self.test_batch_pipeline, target_url))

            for future in as_completed(tasks):
                res = future.result()
                if isinstance(res, list):
                    self.results.extend(res)
                else:
                    self.results.append(res)

        total_time = time.time() - start_all
        self.print_report(total_time)

    def print_report(self, total_time: float):
        print("\n" + "="*75)
        print("📊 500并发全端点压测报告汇总 (500-Concurrency Benchmark Results)")
        print("="*75)

        grouped: Dict[str, List[StatResult]] = {}
        for r in self.results:
            grouped.setdefault(r.endpoint, []).append(r)

        print(f"{'Endpoint 接口':<26} | {'总请求数':<8} | {'成功率':<8} | {'平均耗时(ms)':<12} | {'P95(ms)':<10}")
        print("-" * 75)

        for endpoint, res_list in grouped.items():
            total = len(res_list)
            successes = sum(1 for r in res_list if r.success)
            success_rate = (successes / total) * 100 if total > 0 else 0
            durations = sorted([r.duration_ms for r in res_list])
            avg_ms = sum(durations) / total if total > 0 else 0
            p95_idx = int(total * 0.95)
            p95_ms = durations[min(p95_idx, total - 1)] if total > 0 else 0

            print(f"{endpoint:<26} | {total:<8} | {success_rate:>6.1f}%  | {avg_ms:>10.2f} ms | {p95_ms:>8.2f} ms")

        print("-" * 75)
        qps = len(self.results) / total_time if total_time > 0 else 0
        print(f"⏱️ 压测总耗时: {total_time:.2f} 秒")
        print(f"⚡ 总完成请求量: {len(self.results)} 次")
        print(f"🔥 综合吞吐 QPS: {qps:.2f} 请求/秒")
        print("="*75 + "\n")

if __name__ == "__main__":
    benchmark = FirecrawlBenchmark(API_BASE_URL, API_KEY)
    benchmark.run_benchmark()
