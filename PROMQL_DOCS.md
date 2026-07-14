# Firecrawl 监控 PromQL 查询手册

## 查询入口

```bash
curl 'http://209.33.176.13:30009/prometheus/api/v1/query?query=<PromQL>'
```

或用浏览器访问夜莺 Web UI: `http://209.33.176.13:30009/`

---

## 一、节点监控（Categraf）

| 指标 | PromQL | 说明 |
|------|--------|------|
| CPU 空闲率 | `cpu_usage_idle` | 百分比，值越高越空闲 |
| CPU 使用率 | `cpu_usage_active` | 100 - cpu_usage_idle |
| CPU 用户态 | `cpu_usage_user` | 用户进程占用的 CPU 比例 |
| CPU 内核态 | `cpu_usage_system` | 内核占用的 CPU 比例 |
| CPU I/O 等待 | `cpu_usage_iowait` | 等待 I/O 完成的 CPU 比例，>20% 表示磁盘瓶颈 |
| 内存使用率 | `mem_used_percent` | 百分比 |
| 内存使用量 | `mem_used` | bytes |
| 内存总量 | `mem_total` | bytes |
| 磁盘使用率 | `disk_used_percent` | 百分比，按分区 |
| 磁盘使用量 | `disk_used` | bytes，按分区 |
| 磁盘剩余 | `disk_free` | bytes，按分区 |
| 磁盘总量 | `disk_total` | bytes，按分区 |
| 磁盘读速率 | `rate(diskio_read_bytes[1m])` | bytes/s |
| 磁盘写速率 | `rate(diskio_writes_bytes[1m])` | bytes/s |
| 磁盘 IOPS 读 | `rate(diskio_reads[1m])` | 次数/s |
| 磁盘 IOPS 写 | `rate(diskio_writes[1m])` | 次数/s |
| 网络发送速率 | `rate(net_bytes_sent[1m])` | bytes/s |
| 网络接收速率 | `rate(net_bytes_recv[1m])` | bytes/s |
| 网络连接数 | `net_conntrack_dialer_conn_established_total` | 已建立的 TCP 连接数 |
| 进程总数 | `processes_total` | 系统进程数 |
| 文件描述符 | `process_open_fds` | 进程打开的文件句柄数 |

**按节点查询：**
```promql
cpu_usage_idle{ident="k8s-master"}
cpu_usage_idle{ident=~"k8s-.*"}
mem_used_percent{ident="k8s-node1"}
```

---

## 二、容器监控（kubelet cAdvisor）

| 指标 | PromQL | 说明 |
|------|--------|------|
| 容器 CPU 使用 | `container_cpu_usage_seconds_total{namespace="firecrawl"}` | 累计 CPU 秒数（counter，需 rate） |
| 容器 CPU 速率 | `rate(container_cpu_usage_seconds_total{namespace="firecrawl"}[1m])` | CPU 使用速率（核心数） |
| 容器内存使用 | `container_memory_usage_bytes{namespace="firecrawl"}` | bytes |
| 容器内存 RSS | `container_memory_rss{namespace="firecrawl"}` | 常驻内存，bytes |
| 容器内存工作集 | `container_memory_working_set_bytes{namespace="firecrawl"}` | 实际使用内存，bytes |
| 容器磁盘读 | `rate(container_fs_reads_bytes_total{namespace="firecrawl"}[1m])` | bytes/s |
| 容器磁盘写 | `rate(container_fs_writes_bytes_total{namespace="firecrawl"}[1m])` | bytes/s |
| 容器网络接收 | `rate(container_network_receive_bytes_total{namespace="firecrawl"}[1m])` | bytes/s |
| 容器网络发送 | `rate(container_network_transmit_bytes_total{namespace="firecrawl"}[1m])` | bytes/s |
| OOM 事件 | `container_oom_events_total{namespace="firecrawl"}` | 累计 OOM 次数 |
| 容器重启 | `kube_pod_container_status_restarts_total{namespace="firecrawl"}` | Pod 重启次数 |

**按 Pod 查询：**
```promql
container_memory_usage_bytes{pod=~"firecrawl-firecrawl-api-.*"}
rate(container_cpu_usage_seconds_total{pod=~"firecrawl-firecrawl-worker-.*"}[1m])
```

---

## 三、K8s 集群监控（kube-state-metrics）

| 指标 | PromQL | 说明 |
|------|--------|------|
| 所有 Pod 状态 | `count by (phase) (kube_pod_status_phase)` | Running/Pending/Succeeded/Failed/Unknown 分布 |
| 指定命名空间 Pod | `count by (phase) (kube_pod_status_phase{namespace="firecrawl"})` | Firecrawl 命名空间 |
| Pod 就绪状态 | `kube_pod_status_ready{namespace="firecrawl"}` | 0=未就绪, 1=就绪 |
| Node 就绪 | `kube_node_status_condition{condition="Ready",status="true"}` | 正常运行的 Node 数 |
| Node 信息 | `kube_node_info` | Node 的内核版本、容器运行时等 |
| Deployment 期望副本 | `kube_deployment_spec_replicas{namespace="firecrawl"}` | 期望 Pod 数 |
| Deployment 可用副本 | `kube_deployment_status_replicas_available{namespace="firecrawl"}` | 可用 Pod 数 |
| Deployment 就绪状态 | `kube_deployment_status_condition{namespace="firecrawl"}` | Available/Progressing 等 |
| StatefulSet 副本 | `kube_statefulset_replicas{namespace="n9e"}` | 期望副本数 |
| DaemonSet 就绪 | `kube_daemonset_status_number_ready{namespace="n9e"}` | 已就绪的 DaemonSet Pod |
| 命名空间状态 | `kube_namespace_status_phase` | Active/Terminating |
| Node 资源容量 | `kube_node_status_capacity` | CPU 核数、内存总量等 |
| Node 资源可分配 | `kube_node_status_allocatable` | 可分配 CPU、内存 |
| PVC 用量 | `kube_persistentvolumeclaim_resource_requests_storage_bytes` | PVC 申请容量 |
| PV 状态 | `kube_persistentvolume_status_phase` | Bound/Available/Released |
| 证书过期 | `kube_secret_info` | 配合 metadata 查看 Secret 信息 |
| Service 类型 | `kube_service_spec_type` | ClusterIP/NodePort/LoadBalancer |

---

## 四、Firecrawl 应用监控

### 4.1 并发与限流

| 指标 | PromQL | 说明 |
|------|--------|------|
| 活跃并发数 | `firecrawl_concurrent_active` | 当前正在处理的请求数 |
| 队列任务数 | `concurrency_limit_queue_job_count_total` | 排队等待的任务数 |
| 队列团队数 | `concurrency_limit_queue_team_count` | 排队的团队数 |

### 4.2 请求处理延迟

| 指标 | PromQL | 说明 |
|------|--------|------|
| PDF 异步总耗时 P50 | `histogram_quantile(0.5, rate(firecrawl_fire_pdf_async_total_duration_seconds_bucket[5m]))` | 50% 请求在此时间内完成 |
| PDF 异步总耗时 P90 | `histogram_quantile(0.9, rate(firecrawl_fire_pdf_async_total_duration_seconds_bucket[5m]))` | 90% 请求在此时间内完成 |
| PDF 异步总耗时 P99 | `histogram_quantile(0.99, rate(firecrawl_fire_pdf_async_total_duration_seconds_bucket[5m]))` | 99% 请求在此时间内完成 |
| PDF 异步轮询次数 | `rate(firecrawl_fire_pdf_async_poll_count_sum[5m]) / rate(firecrawl_fire_pdf_async_poll_count_count[5m])` | 平均轮询次数 |
| 缓存读取耗时 P50 | `histogram_quantile(0.5, rate(firecrawl_index_cache_read_duration_seconds_bucket[5m]))` | 缓存读取延迟 |
| 缓存读取耗时 P99 | `histogram_quantile(0.99, rate(firecrawl_index_cache_read_duration_seconds_bucket[5m]))` | 缓存读取延迟 P99 |

### 4.3 NUQ 队列

| 指标 | PromQL | 说明 |
|------|--------|------|
| 队列深度 | `nuq_queue_scrape_job_count` | 待处理的任务数 |
| 空闲连接数 | `nuq_pool_idle_count` | 空闲的 NUQ 连接 |
| 等待连接数 | `nuq_pool_waiting_count` | 等待获取连接的请求数 |
| 总连接数 | `nuq_pool_total_count` | NUQ 连接池总量 |

### 4.4 信号量

| 指标 | PromQL | 说明 |
|------|--------|------|
| 活跃信号量 | `noq_semaphore_active` | 当前活跃的信号量 |
| 信号量获取耗时 P50 | `histogram_quantile(0.5, rate(noq_semaphore_acquire_duration_seconds_bucket[5m]))` | 获取信号量延迟 |
| 信号量持有耗时 P50 | `histogram_quantile(0.5, rate(noq_semaphore_hold_duration_seconds_bucket[5m]))` | 持有信号量时长 |

### 4.5 Credits & 业务

| 指标 | PromQL | 说明 |
|------|--------|------|
| 计费团队数 | `billed_teams_count` | 当前计费的团队数 |

---

## 五、Prometheus 自身监控

| 指标 | PromQL | 说明 |
|------|--------|------|
| 所有目标状态 | `up` | 1=正常, 0=异常 |
| 目标抓取耗时 | `scrape_duration_seconds` | 每次抓取耗时 |
| 抓取样本数 | `scrape_samples_scraped` | 每次抓取的样本数 |
| TSDB 序列数 | `prometheus_tsdb_head_series` | 当前存储的时间序列数 |
| TSDB 存储量 | `prometheus_tsdb_storage_blocks_bytes` | 存储占用 bytes |
| 内存使用 | `process_resident_memory_bytes{job="prometheus"}` | Prometheus 进程内存 |
| CPU 使用 | `rate(process_cpu_seconds_total{job="prometheus"}[1m])` | Prometheus CPU 使用率 |

**健康巡检：**
```promql
# 检查所有 target 是否在线
count by (job) (up == 0)

# 检查抓取错误
scrape_samples_scraped == 0

# 检查 Prometheus 配置重载是否成功
prometheus_config_last_reload_successful == 0
```

---

## 六、常用组合查询

### 6.1 按命名空间汇总资源使用

```promql
# 各命名空间 CPU 使用
sum by (namespace) (rate(container_cpu_usage_seconds_total{container!=""}[5m]))

# 各命名空间内存使用
sum by (namespace) (container_memory_working_set_bytes{container!=""})
```

### 6.2 按 Pod 汇总

```promql
# 单个 Pod 的 CPU 使用
sum(rate(container_cpu_usage_seconds_total{pod="firecrawl-firecrawl-api-xxxxx"}[1m]))

# 单个 Pod 的内存使用
sum(container_memory_working_set_bytes{pod="firecrawl-firecrawl-api-xxxxx"})
```

### 6.3 节点资源利用率

```promql
# Node CPU 利用率（排除空闲）
100 - avg by (instance) (rate(container_cpu_usage_seconds_total{container!=""}[5m])) * 100

# Node 内存利用率
(1 - sum(container_memory_working_set_bytes{container!=""}) / sum(machine_memory_bytes)) * 100
```

---

## 七、curl 查询示例

```bash
# 查询所有目标在线状态
curl 'http://209.33.176.13:30009/prometheus/api/v1/query?query=up'

# 查询 Firecrawl 命名空间中所有 Pod 状态
curl 'http://209.33.176.13:30009/prometheus/api/v1/query?query=kube_pod_status_phase{namespace="firecrawl"}'

# 查询 Firecrawl API 内存使用
curl 'http://209.33.176.13:30009/prometheus/api/v1/query?query=container_memory_usage_bytes{namespace="firecrawl",container="api"}'

# 查询 CPU 空闲率（带节点标识）
curl 'http://209.33.176.13:30009/prometheus/api/v1/query?query=cpu_usage_idle'

# 查询 Firecrawl 并发数
curl 'http://209.33.176.13:30009/prometheus/api/v1/query?query=firecrawl_concurrent_active'
```
