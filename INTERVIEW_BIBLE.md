# xcrawl 爬虫平台 · 面试圣经

> **定位**：你是一家 B 端 SaaS 公司的 SRE / 高级运维工程师，公司核心产品是 **xcrawl**（网页抓取 API 平台，对标 Firecrawl / Crawlbase / ScrapingBee）。
>
> 你从零搭建了 xcrawl 的生产基础设施、CI/CD、监控告警体系。
> 以下所有话术基于 **2 Master + 6 Worker 的 K8s 集群，日均百万级请求**来展开。
> 每个回答都写得足够长，面试时根据对方反应挑重点说。

---

## 点一：基础设施 — K8s 集群 + API 平台

### 标准话术

> 我负责 xcrawl 平台从 0 到 1 的基础设施建设和日常运维。我们跑在 **8 节点 K8s 集群**上——2 个 Master 节点做控制平面高可用（前面挂 HAProxy + keepalived 做 VIP），6 个 Worker 节点承载业务负载。
>
> **两个环境物理隔离**：
> - **生产环境**：独占 4 台 Worker（32C / 64GB），跑 xcrawl 的核心 API 服务。每天处理数百万次 scrape / crawl / search / map 请求，高峰期并发能到 500+
> - **测试/预发环境**：独占 2 台 Worker（16C / 32GB），用于新功能验证、灰度发布、压力测试
>
> **集群组件选型**：
> - K8s v1.29 + containerd 运行时（当时弃用了 Docker，containerd 更轻量、资源占用更少）
> - Calico CNI 做网络（BGP 模式，Pod 网络直通，不需要 Overlay，性能损耗小）
> - 存储用的 **local-path-provisioner**（测试环境）+ 生产环境用独立 NAS 给 PostgreSQL 和 Redis 做持久化
> - 所有应用通过 **Helm chart** 管理，不同环境用不同的 values 文件，镜像 Tag 也做区分
>
> **xcrawl 在 K8s 里的部署架构**：
>
> 我把 xcrawl 拆成了 8 个微服务组件，每个独立 Deployment：
> | 组件 | 副本数 | 说明 |
> |------|--------|------|
> | xcrawl-api | 3 | 主 API 服务，接收所有 REST 请求 |
> | xcrawl-worker | 5 | 网页抓取执行器 |
> | xcrawl-extract-worker | 2 | 结构化数据提取 |
> | xcrawl-playwright | 3 | 浏览器渲染，处理 SPA 页面 |
> | xcrawl-nuq-worker | 5 | 消息队列消费者，任务调度（用的是 Redis + NUQ 类似的机制） |
> | xcrawl-redis | 2（哨兵） | 缓存 + 速率限制 + 消息队列 |
> | xcrawl-rabbitmq | 3（镜像队列） | API 和 Worker 之间的异步通信 |
> | xcrawl-postgresql | 1（主从） | 队列持久化 + 业务数据 |
>
> API 入口走 **Nginx Ingress Controller**，对外暴露一个公网 LoadBalancer IP。认证层在 API 前面加了一层 **Nginx Lua 脚本**做 API Key 校验和速率限制，通过了才转发到后端 api Pod。

### 面试官追问

**Q: 集群节点挂了怎么办？**

> 分几种情况：
>
> **Master 挂了一台**：HAProxy 会自动把流量切到另一台 Master，kube-apiserver 无感。etcd 是 3 节点的（Master 2 + Worker 上单独部署一个 etcd member），多数节点存活就能正常读写。
>
> **Worker 挂了**：这台节点上的 Pod 会自动漂移到其他 Worker（toleration 设置合适的时间，默认 5 分钟）。xcrawl 的组件都是无状态的（除了 DB），重新调度后自动从队列里拉任务继续处理。有状态的 RabbitMQ 我们做了镜像队列，数据不丢。
>
> **整个集群挂了**：这确实发生过一次——机房断电。恢复流程：先把 NAS 存储挂载回来，然后按依赖顺序启动：PostgreSQL → Redis → RabbitMQ → NUQ → Worker → API。全流程恢复大约 15 分钟。

**Q: 为什么 xcrawl 要拆成这么多组件？直接一个单体不行吗？**

> 单体当然可以，但问题是：① 爬虫场景下，**scrape 请求和 extract 请求的负载特征完全不同**——scrape 吃网络 IO，extract 吃 CPU（跑 LLM）。放在一起互相干扰。② **Worker 需要独立扩缩容**——白天流量大，Worker 从 5 扩到 20，API 不需要跟着扩。拆开以后每个组件可以独立做 HPA。
>
> 而且每个组件挂了不影响其他组件：API 挂了，已经在队列里的任务 Worker 继续处理；NUQ 队列积压了，API 可以限流拒绝新请求，不会雪崩。

**Q: 容器运行时为什么选 containerd 而不是 Docker？**

> 当时调研后决定用 containerd：① K8s 从 1.24 开始弃用 Docker Shim，containerd 是更原生的 CRI 实现 ② containerd 的资源开销比 Docker 小（没有 Docker Engine 那一层 daemon）③ Docker 的很多功能（docker build、docker compose）在 K8s 里用不到，装它只是多一个攻击面。迁移过程确实遇到一些问题，比如 Categraf 默认采集 Docker socket，还要特地去关掉。

**Q: 请求链路是怎样的？从用户发起到返回结果？**

> 一条完整的 scrape 请求链路：
>
> 用户 `POST /v2/scrape {"url": "https://..."}` 
> → 公网 LB → Nginx Ingress（校验 API Key + 速率限制）
> → xcrawl-api Pod（参数校验、Redis 查缓存）
> → 如果缓存没有 → 发消息到 RabbitMQ
> → NUQ Worker 消费消息 → 调用 Playwright 或直接 HTTP Get
> → 抓取结果写回 Redis 缓存 + 返回给用户
>
> 同步请求（scrape）在 3-5 秒内返回，异步请求（crawl）先返回 jobId，用户轮询 GET /v2/crawl/:id 拿结果。

**Q: Calico 网络方案怎么选的？遇到过什么问题？**

> 选 Calico 是因为：① 支持 BGP 模式，Pod 直接路由可达，不需要 Overlay 封包，性能比 Flannel 的 VXLAN 好 ② NetworkPolicy 支持完善，可以精细控制跨命名空间访问 ③ 社区活跃，K8s 主流 CNI 之一。
>
> 踩过的坑：
> ① **BGP 路由泄露**：刚开始 Calico 默认把 Pod 网段广播到上游交换机，导致交换机路由表被 Pod IP 刷爆。解决：配置 `calico-node` 的 `calico_backend: bird`，把 BGP 模式从 `full-mesh`（全互联）改成 `route-reflector` 模式，只和核心交换机建 BGP 会话。
> ② **IPPool 耗尽**：随着 Pod 增多，默认的 `192.168.0.0/16` 不够用了。解决：规划 CIDR 的时候留够余量，直接配一个 `/12` 的网段。
> ③ **Calico 升级断连**：升级 Calico 版本时，节点上的 calico-node Pod 重启期间网络断了几秒。解决：升级前先把业务 Pod 驱赶到其他节点，或者在低峰期操作。

**Q: etcd 怎么备份和恢复的？**

> etcd 是整个集群的命根子，我定期做三件事：
>
> ① **自动备份**：写了个 CronJob，每天凌晨用 `etcdctl snapshot save` 把快照打到 NAS 存储，保留 30 天
> ② **异地备份**：NAS 上的快照再 rsync 到另一台机房服务器，防止单机房故障
> ③ **恢复演练**：每季度在测试环境做一次 etcd 恢复演练，确保备份文件是可用的
>
> 恢复命令：`etcdctl snapshot restore snapshot.db --data-dir=/var/lib/etcd`，然后把数据目录挂回 etcd 成员，重启即可。
>
> 注意：etcd 备份要在**同一个节点**上执行，跨节点恢复需要小心 `--initial-cluster` 和 `--initial-advertise-peer-urls` 参数。

**Q: Node 宕机了，K8s 是怎么处理上面 Pod 的？**

> K8s 的 **Node Controller** 会检测 Node 状态。默认每隔 40s 发一次心跳（`node-monitor-period`），如果连续 5 次没收到（`node-monitor-grace-period: 40s`），Node 被标记为 `Unknown`。再过 5 分钟（`pod-eviction-timeout: 5m`），Master 开始驱逐这台 Node 上的 Pod，将它们调度到其他可用 Node 上。
>
> 实际场景中，如果是短暂网络抖动（1-2 分钟），Node 恢复后 Pod 继续运行不会被驱逐。如果是宕机，5 分钟后 Pod 开始重新调度。StatefulSet 的 Pod 需要手动删除 PVC 锁才能迁移，Deployment 的 Pod 会自动在健康 Node 上重建。
>
> 我遇到过一次：只有 2 台 Worker 在线，第 3 台宕机了，那台上的 Pod 驱逐到剩余两台上，导致剩余 Node 资源不够，部分 Pod 一直 Pending。解决：给关键组件加了 `priorityClassName: high`，确保资源紧张时低优先级 Pod 先被驱逐。

**Q: HPA 在生产环境怎么配的？**

> 核心组件都配置了 HPA：
>
> ```yaml
> apiVersion: autoscaling/v2
> kind: HorizontalPodAutoscaler
> metadata:
>   name: xcrawl-api-hpa
> spec:
>   scaleTargetRef:
>     apiVersion: apps/v1
>     kind: Deployment
>     name: xcrawl-api
>   minReplicas: 3
>   maxReplicas: 20
>   metrics:
>   - type: Resource
>     resource:
>       name: cpu
>       target:
>         type: Utilization
>         averageUtilization: 70
>   - type: Pods
>     pods:
>       metric:
>         name: firecrawl_concurrent_active
>       target:
>         type: AverageValue
>         averageValue: 50
> ```
>
> 两个指标配合：CPU 超过 70% 自动扩容，或者活跃并发数超过 50 也触发扩容。缩容冷却时间设了 `scale-down-stabilization-window: 5m`，避免流量抖动时频繁扩缩。
>
> 踩过的坑：一开始只配了 CPU，但请求量上来的时候 CPU 还没打满，队列先积压了。所以加了一个业务指标（并发数）做 HPA，响应更快。

**Q: 遇到过 Pod 起不来的情况吗？怎么排查？**

> 太常见了，排查套路：
>
> ① `kubectl describe pod <name>` → 看 Events，最常见的是 `ImagePullBackOff`（镜像拉不下来）、`CrashLoopBackOff`（进程启动就挂）、`Pending`（资源不够）
> ② 镜像拉不下来 → 检查 Harbor 是否正常、Secret 是否过期、镜像 tag 是否存在
> ③ CrashLoopBackOff → `kubectl logs <pod>` 看启动日志，常见原因是数据库连不上、环境变量缺失、配置文件错误
> ④ Pending → `kubectl describe pod` 看 `Events` 里有没有 `Insufficient cpu` / `Insufficient memory`，或者 PVC 没绑定
>
> 经典案例：有一次升级后所有 Pod 起不来，`CrashLoopBackOff`。查日志发现是 ConfigMap 里的数据库连接地址写错了，连不上 PostgreSQL。Helm 升级时 values 里的 `DATABASE_URL` 配成了旧地址。回滚 ConfigMap 后恢复。

**Q: StorageClass 和 PVC 怎么管理的？**

> 用的是 **local-path-provisioner**（本地磁盘）+ 生产环境加了外接 NAS。StorageClass 配置：
>
> - `local-path (default)`：临时数据，用了 `WaitForFirstConsumer` 绑定模式（Pod 调度到哪个节点，PVC 就绑到哪个节点的磁盘上），回收策略 `Delete`
> - `nas-storage`：生产数据库用，`Immediate` 绑定模式，回收策略 `Retain`
>
> 踩过的坑：RabbitMQ 用的是 StatefulSet + PVC，Pod 重建后 PVC 还在老节点上，新 Pod 调度到其他节点就绑不上。解决：给 RabbitMQ 加了 `nodeSelector` 锁定到特定节点，或者用 ReadWriteMany 的 NAS 存储。

**Q: K8s 版本升级做过吗？怎么做的？**

> 做过从 1.27 到 1.29 的升级。流程：
>
> ① **规划**：先看 Release Note，确认 API 变更（比如 1.28 弃用了某些 beta API），检查 xcrawl 有没有用到被移除的 API 版本
> ② **测试环境升级**：先在 staging 集群升级 Master → 升级 Worker。跑全量 E2E 测试，确认所有功能正常
> ③ **生产环境升级**：分批升级 Master（先备后用，HAProxy 切流），然后逐个升级 Worker（先 cordon + drain 一台，升级完 uncordon，确认 Pod 正常后再下一台）
> ④ **回滚方案**：如果升级失败，二进制方式（kubeadm）可以 `kubeadm upgrade revert`，并且 etcd 提前做了快照
>
> 注意事项：升级前要看 `kubectl get apiservice` 有没有第三方服务（比如 Calico、Ingress Controller）和新版 K8s 的兼容性。Calico 必须升级到支持新版 K8s 的版本。

**Q: 集群 DNS 出过问题吗？怎么排的？**

> 出过一次。现象是 API 日志报 `getaddrinfo ENOTFOUND rabbitmq`，但别的高可用 Pod 解析正常。
>
> 排查：① 进 Pod 里 `nslookup rabbitmq` → 超时 ② 检查 CoreDNS Pod：`kubectl get pods -n kube-system -l k8s-app=kube-dns` → 一个 CoreDNS Pod 在 CrashLoopBackOff ③ 看 CoreDNS 日志：`[FATAL] plugin/corefile: open /etc/coredns/Corefile: no such file` → ConfigMap 被误删了
>
> 恢复：重新 apply CoreDNS 的 ConfigMap。临时方案：Pod 里加 `dnsPolicy: ClusterFirst` 或直接用 IP 地址绕过 DNS。
>
> 事后：给 CoreDNS 加了监控告警，并且不允许别人手动改 kube-system 的 ConfigMap。

**Q: K8s 安全方面做过什么？**

> 做了三层安全：
>
> ① **RBAC 最小权限**：每个服务有自己的 ServiceAccount，只给需要的权限。比如 xcrawl-api 只需要 get pod 和 list service，不需要 cluster-admin。Prometheus 的 SA 给了只读 ClusterRole。
>
> ② **NetworkPolicy**：默认拒绝跨 Namespace 的流量（Calico 的 GlobalNetworkPolicy），然后显式放行需要的（比如 n9e 到 firecrawl 的 /metrics 访问）。firecrawl 命名空间内部组件之间可以互通，但外部不能直接访问。
>
> ③ **Secret 管理**：K8s Secret 只是 base64 编码，不能算加密。我们用了 **Sealed Secrets**（bitnami/sealed-secrets）——Git 仓库里存的是加密后的 SealedSecret，只有集群里的 controller 能解密。生产环境的 API Key 和数据库密码走 Vault，Pod 启动时通过 CSI 驱动注入。

**Q: K8s 遇到过的最大故障是什么？**

> 有一次某个同事在 Master 节点上跑了 `docker system prune -a`（清空所有镜像），导致 etcd 和 kube-apiserver 依赖的基础镜像被删了，Master 上的关键 Pod 挂了一片，集群控制面几乎瘫痪了半小时。
>
> 教训：① Master 节点应该打 `taint: NoSchedule`，不让业务 Pod 调度上去 ② 禁止在集群节点上手动执行 docker 命令，所有操作走 CI/CD ③ 核心组件的镜像应该拉下来 persist 到本地，或者用 containerd 的镜像预热功能

**Q: 如果给你一个全新的 K8s 集群，部署一套 xcrawl 需要多久？**

> 如果镜像和 Helm Chart 都准备好了，纯部署时间大约 **30 分钟**：
> - 5 分钟：建 Namespace + 配 NetworkPolicy + 配 RBAC
> - 5 分钟：部署 Redis + RabbitMQ + PostgreSQL（先启动基础中间件）
> - 10 分钟：部署 NUQ Worker + Prefetch Worker + Extract Worker
> - 5 分钟：部署 API + Worker + Playwright
> - 5 分钟：配 Ingress + 验证 E2E 测试通过

---

## 点二：CI/CD 流水线

### 标准话术

> 我搭建了从代码到生产的完整 CI/CD 流水线，基于 **GitLab CI + Harbor 镜像仓库 + Helm + K8s**，整个流程不需要人工干预：
>
> **① 代码提交阶段**
>
> 开发 push 代码到 GitLab，触发 CI Pipeline。我们分几个分支：`feature/*`（功能开发）、`develop`（集成分支）、`release/*`（发布候选）、`main`（生产）。只有 release 和 main 分支会触发生产构建。
>
> **② 镜像构建阶段**
>
> GitLab Runner（我们自建的 K8s Runner）执行 Dockerfile 构建，打上两个 Tag：`git-${COMMIT_SHA}`（唯一标识）和 `latest`（默认最新）。镜像推送到公司内的 **Harbor** 私有仓库，配置了镜像扫描（Trivy），高危漏洞直接阻断 Pipeline。
>
> **③ 自动化测试阶段**
>
> 镜像构建完后自动部署到 staging 环境，触发 E2E 测试套件。测试覆盖：
> - 核心 API 功能测试：scrape 正常 URL / 404 / 超时、crawl 创建 → 轮询 → 取消、search 结果校验
> - Playwright 渲染测试：SPA 页面、登录态、无限滚动
> - 稳定性测试：并发 50 请求持续 5 分钟，看错误率和延迟是否超标
>
> E2E 全部通过后，自动打上 `release-${VERSION}` 的 Git Tag，触发生产部署。
>
> **④ 灰度发布阶段**
>
> 生产部署分两步走。先灰度：
> - Helm 新起一个灰度 Deployment（`xcrawl-api-canary`），副本数 1，image tag 指向新版本
> - Ingress 切 10% 流量到 canary Pod，观察 15 分钟
> - 观察指标：API 错误率 < 0.1%、P99 延迟不高于基线 20%、`nuq_queue_scrape_job_count` 无突刺
> - 任何指标超标，自动停止灰度，钉钉通知值班人
>
> **⑤ 全量上线阶段**
>
> 灰度验证通过后，执行 `helm upgrade` 滚动更新生产 Deployment。K8s 滚动更新策略：
> - `maxSurge: 1`（每次多起 1 个新 Pod）
> - `maxUnavailable: 0`（保证最少有全部 Pod 在服务）
> - `progressDeadlineSeconds: 300`（5 分钟内 Pod 没 Ready 就回滚）
>
> 滚动更新完，自动删除 canary Deployment。整个流程从 push 到全量上线大约 **20-30 分钟**，支持 **每周 2-3 次** 的迭代频率。

### 面试官追问

**Q: 回滚怎么做的？**

> 三种回滚手段：
>
> **最快**：`helm rollback xcrawl-api <版本号>`，30 秒内回退到上一个 Release 版本。Helm 会保留最近 10 个 Release 历史。
>
> **精确回滚**：指定具体的 image tag 重新部署。比如新版本是 `abc123`，回滚到 `def456`，直接改 Helm values 里的 `image.tag` 重新 deploy。
>
> **一键切流**：如果问题严重，Ingress 层可以直接把流量全部切回旧集群（xcrawl 的旧版本仍然保留 Deployment，只是 replicaCount 设为 0），做到秒级切流。

**Q: 镜像构建优化做过吗？**

> 做过几层优化：
>
> ① **多阶段构建**：builder 阶段装 devDependencies + 编译 TypeScript，runtime 阶段只保留 dist 和 production dependencies，镜像从 1.2GB 降到 280MB。
>
> ② **分层缓存**：`package.json` 和 `yarn.lock` 单独 COPY 并 `yarn install`，业务代码放后面。这样只有依赖变更时才重新 install，Pipelines 从 8 分钟降到 3 分钟。
>
> ③ **基础镜像固定**：不用 `node:latest`，固定到 `node:20-alpine` 的具体 SHA，避免基础镜像更新导致非预期的构建结果。

**Q: Helm Chart 是怎么管理的？**

> 公司内部一个 `helm-charts` 仓库，每个应用一个目录。生产环境和测试环境用不同的 values 文件：
>
> ```
> xcrawl-api/
> ├── Chart.yaml
> ├── templates/
> │   ├── deployment.yaml
> │   ├── service.yaml
> │   ├── ingress.yaml
> │   ├── hpa.yaml
> │   └── configmap.yaml
> └── values/
>     ├── production.yaml     # 3 副本 + 持久化 + 资源 limits
>     ├── staging.yaml         # 1 副本 + 不开持久化
>     └── dev.yaml            # 1 副本 + 资源不限制
> ```
>
> 部署命令：`helm upgrade --install xcrawl-api ./xcrawl-api -f values/production.yaml`

**Q: 分支策略是什么？Git Flow 还是 Trunk Based？**

> 我们用的 **Git Flow** 变体：
> - `main`：生产分支，只从 release 合并，禁止直接 push
> - `develop`：集成分支，feature 合并到这里跑自动化测试
> - `release/*`：发布候选，从 develop 切出来，只修 Bug，测试通过后合并到 main
> - `feature/*`：功能开发，从 develop 切出来，开发完合并回 develop
> - `hotfix/*`：紧急修复，从 main 切出来，修完合并回 main 和 develop
>
> 为什么不用 Trunk Based？因为我们团队 10 多人并行开发，release 周期是周级别，Git Flow 更适合版本管理和回溯。如果是 3-5 人的小团队快速迭代，Trunk Based 更合适。

**Q: 怎么做自动化测试的？测试覆盖到什么程度？**

> 测试分三层：
>
> **单元测试**：Jest，每个函数/工具的单元测试。CI 中 `yarn test:unit` 必须通过，否则阻断。覆盖率要求 > 80%。
>
> **集成测试**：staging 环境部署完后，自动跑 Postman/Newman 集合。覆盖 xcrawl 的全部核心 API：scrape（正常/404/超时/超大页面）、crawl（创建→轮询→取消→errors）、search（结果校验/分页/空结果）、map（sitemap 解析/子域名发现）。每个接口至少覆盖正常和异常两个用例。
>
> **E2E 测试**：Firecrawl 的 harness（snips）框架，模拟用户使用场景。比如：先 scrape 一个页面 → 提取链接 → 用这些链接发起 crawl → 等 crawl 完成 → 校验结果完整性。跑完大约 10 分钟。
>
> 测试不过的版本，自动阻断生产发布，钉钉通知开发。

**Q: 镜像安全和漏洞扫描怎么做的？**

> 用了 **Trivy**（aquasecurity/trivy）做镜像扫描，集成在 GitLab CI 中：
>
> ```yaml
> scan:
>   script:
>     - trivy image --severity CRITICAL,HIGH harbor.internal.com/xcrawl/api:${CI_COMMIT_SHA}
>   only:
>     - main
>     - release/*
> ```
>
> 扫描规则：① CRITICAL 漏洞发现 1 个就阻断 Pipeline ② HIGH 漏洞超过 3 个阻断 ③ MEDIUM/LOW 只报告不阻断
>
> 基础镜像每两周更新一次（`node:20-alpine`），修复已知 CVE。如果遇到阻塞性的 CVE，走紧急通道更新。

**Q: 版本号和制品怎么管理的？**

> 遵循 **SemVer**（语义化版本）：`major.minor.patch`。
> - `major`：不兼容的 API 变更
> - `minor`：向下兼容的新功能
> - `patch`：Bug 修复
>
> 每次 CI 构建生成的镜像 tag 是 `v${VERSION}-${CI_COMMIT_SHORT_SHA}`。同时 Git 打对应 tag：
> ```
> v2.3.0-a1b2c3d
> ```
>
> Harbor 保留策略：最近 10 个 tag 保留，其余自动清理。生产环境固定到具体 tag 部署，staging 用 `latest`。

**Q: 有没有做过蓝绿部署或 A/B 测试？**

> 灰度发布其实就是简化版的 **金丝雀发布**（Canary Release）。更完整的蓝绿部署我们也做过：
>
> 蓝绿部署：同时维护两个完全相同的环境（Blue: 当前生产、Green: 新版本）。切换时改 Ingress 指向，一键切流。回滚也一样方便——切回 Blue 就行。
>
> A/B 测试：Ingress 可以根据 Header（比如 `X-Canary-Version: v2`）或者 Cookie 分流。我们用来测试新引擎的抓取效果——把 5% 的流量导到新引擎，对比新老引擎的抓取成功率和结果质量。这种方式对新功能上线的信心提升很大。

**Q: 如果上线后出问题了，怎么快速止血？**

> 分三个等级：
>
> **P0（全挂）**：立即回滚。`helm rollback` 30 秒内回到上一版本。同时恢复旧版本的 Deployment，Ingress 切流。
>
> **P1（部分功能异常）**：限流降级。在 Nginx 层限制异常接口的速率，保证核心功能正常。比如 extract 出问题就先限流，scrape 正常提供服务。
>
> **P2（少量错误）**：观察 5 分钟，如果错误率持续上升就自动回滚。HPA 会自动扩容分担压力，我们先排查根因再决定是否回滚。
>
> 核心原则：**回滚优先于修复**。先保证用户能用，再慢慢修。

**Q: 怎么做蓝绿部署的？helm 怎么配合？**

> 蓝绿部署的核心是两套独立的 Deployment + 一个 Service 指向当前活跃的版本：
>
> ```yaml
> # Blue (当前)
> apiVersion: apps/v1
> kind: Deployment
> metadata:
>   name: xcrawl-api-blue
> spec:
>   replicas: 3
>   template:
>     spec:
>       containers:
>       - image: xcrawl/api:v2.2.0
> 
> # Green (新版本)  
> apiVersion: apps/v1
> kind: Deployment
> metadata:
>   name: xcrawl-api-green
> spec:
>   replicas: 3
>   template:
>     spec:
>       containers:  
>       - image: xcrawl/api:v2.3.0
> 
> # Service 指向 Blue
> apiVersion: v1
> kind: Service
> spec:
>   selector:
>     app: xcrawl-api
>     version: blue
> ```
>
> 部署流程：先部署 Green → 等 Pod 全部 Ready → E2E 测试 → 修改 Service selector 指向 `version: green` → 观察 10 分钟 → 缩容 Blue。
>
> 回滚只需要把 Service selector 改回 `version: blue`，秒级完成。

**Q: 有没有做过自动化缩容？比如夜间流量低的时候？**

> 做了基于时间的 **CronHPA**。K8s 原生的 HPA 只支持基于指标扩缩，不支持定时。我们用了一个开源组件 `k8s-advanced-hpa` / 或者自己写了一个 CronJob 来改 replicaCount：
>
> ```yaml
> apiVersion: batch/v1
> kind: CronJob
> metadata:
>   name: scale-down-night
> spec:
>   schedule: "0 22 * * *"    # 每晚 22:00
>   jobTemplate:
>     spec:
>       template:
>         spec:
>           containers:
>           - image: bitnami/kubectl
>             command:
>             - kubectl
>             - scale
>             - deployment/xcrawl-worker
>             - --replicas=3
>             - -n
>             - xcrawl-prod
> ```
>
> 早 8 点扩到 8 副本（`0 8 * * *`），晚 10 点缩到 3 副本。一个月算下来节省了约 40% 的资源成本。

**Q: CI Pipeline 跑得慢怎么办？做过哪些优化？**

> 做过一系列优化，把 Pipeline 从 15 分钟降到了 5 分钟：
>
> ① **并行 Stage**：单元测试、lint、类型检查三个任务并行运行，而不是串行。从 8 分钟降到 3 分钟。
>
> ② **Docker 层缓存**：GitLab Runner 挂载了 Docker 的 layer cache，只有 `package.json` 变更时才重新 `yarn install`，否则直接用缓存层。从 5 分钟降到 20 秒。
>
> ③ **按需构建**：只有 `apps/api/` 或其依赖文件变更时才构建 API 镜像，`apps/worker/` 变更只构建 Worker 镜像，互不影响。用 GitLab CI 的 `changes` 条件判断。
>
> ④ **测试切片**：单元测试分 4 个并行 job 跑（test:1、test:2、test:3、test:4），每个跑 25% 的用例。从 3 分钟降到 45 秒。

---

## 点三：后端架构运维与性能优化

### 标准话术

> 我虽然不直接写业务代码，但在运维过程中深度参与了后端的性能排查和架构优化。xcrawl 后端基于 **Node.js + TypeScript**（Express 框架），核心引擎是自研的分布式爬虫调度系统。以下是我处理过的几类典型生产问题：
>
> **案例 1：NUQ 队列积压导致 API 响应缓慢**
>
> **现象**：用户反馈 scrape 请求等待时间变长，有些请求超时。Prometheus 看 `nuq_queue_scrape_job_count` 从几百飙到了 2 万+。
>
> **排查过程**：
> ① 先看队列消费速度：`rate(nuq_queue_processed_total[5m])` 发现消费速率没有下降，但生产速率是消费的 3 倍 → 说明流量突增，不是 Worker 出问题。
> ② 再看连接池：`nuq_pool_waiting_count` 很高 → NUQ 连接池打满了，新任务在排队等连接。
> ③ 看 Worker 资源：`container_cpu_usage_seconds_total` 和 `container_memory_working_set_bytes` 都在正常范围内，说明 Worker 本身不饱和，瓶颈在连接池。
>
> **解决方案**：
> ① 临时：扩容 NUQ Worker 从 3 副本到 8 副本，`NUM_WORKERS_PER_QUEUE` 从 8 调到 16（每个 Pod 内部线程数），队列深度 15 分钟内从 2 万降到 200。
> ② 长期：优化 NUQ 的连接池配置，加了连接池预热（应用启动时预先建立 50 个连接而不是按需创建）。改了 RabbitMQ 的 prefetch count 从 1 调到 5，每次批量拉取 5 个消息减少网络往返。
>
> **案例 2：Playwright 内存泄漏导致 Pod 频繁 OOM 重启**
>
> **现象**：xcrawl-playwright Pod 运行 8-12 小时后内存持续增长，直到被 K8s OOMKill 重启，然后周而复始。业务影响是 SPA 页面抓取成功率从 99% 掉到 92%。
>
> **排查过程**：
> ① `container_memory_working_set_bytes` 呈线性增长，没有平台趋势 → 典型的内存泄漏特征。
> ② `container_oom_events_total` 有计数 → 确认 OOM 发生。
> ③ 进 Pod 看日志，发现每次请求都会 `const browser = await puppeteer.launch()`，但请求结束后 `browser.close()` 没有保证被执行。特别是当页面超时或网络异常时，finally 块没执行到，browser 进程成了僵尸进程，内存不释放。
>
> **解决方案**：
> ① 短期止血：给 playwright Pod 加 `resources.limits.memory: 4Gi`（原来 2Gi），同时设置 `restartPolicy: Always`，OOM 后自动重启，影响降到最低。
> ② 长期修复：和开发一起定位代码，把 `browser.close()` 放到 `try...catch...finally` 的 finally 块里，确保无论成功还是异常都能释放。同时加了浏览器实例池（预先创建 5 个浏览器实例复用，而不是每次请求都 launch），内存稳定在 800MB 不再增长。
>
> **案例 3：数据库慢查询导致任务调度死锁**
>
> **现象**：NUQ 任务偶尔卡死，新任务不派发，但是 Pod 都正常。重启 NUQ Worker 后恢复，但几小时后复现。
>
> **排查过程**：
> ① 看 NUQ Worker 日志，发现大量 `timeout: acquiring connection from pool` 和 `deadlock detected`。
> ② 连到 PostgreSQL 查 `pg_stat_activity`，发现大量 `SELECT ... FOR UPDATE SKIP LOCKED` 在等待锁，有些已经等了 30 秒+。
> ③ 查 `pg_locks`，发现多个 Worker 同时抢占队列任务时，PostgreSQL 的行级锁冲突导致死锁。
>
> **解决方案**：
> ① 加 PgBouncer 做连接池中间件：限制最大连接数 50，减少数据库连接压力。
> ② 调整 NUQ 的轮询间隔从 100ms 改成 500ms，降低数据库争抢频率。
> ③ 任务表加索引，`FOR UPDATE SKIP LOCKED` 的查询条件加 `status = 'pending'` 的 partial index，查询效率提升 10 倍。

### 面试官追问

**Q: 你是怎么做容量规划的？怎么决定什么时候扩容？**

> 基于趋势 + 阈值两个维度：
>
> **趋势预测**：Prometheus 的历史数据 + 业务增长曲线（比如每周增长 10%），提前 2 周规划资源。比如当前 Worker 单副本处理能力是 100 req/s，当前流量 300 req/s（3 副本），预计 2 周后到 500 req/s，那提前加到 5 副本。
>
> **阈值告警**：HPA 配置了 CPU 超过 70% 自动扩，同时 Prometheus 告警规则里 `nuq_queue_scrape_job_count > 500` 就通知值班人，`> 2000` 自动触发扩容脚本。
>
> **压测数据**：上线前做压力测试，得到每个组件的性能基线。比如单个 xcrawl-api Pod QPS 上限是 200（P99 < 2s），内存单 Pod 稳定在 500MB，据此估算不同流量水平下的副本数。

**Q: 遇到过缓存穿透/击穿吗？怎么处理的？**

> 遇到过。某个客户连续 scrape 一个不存在的页面，每次都穿透到下游引擎，导致那个引擎的延迟飙升影响其他客户。
>
> 解决：① **布隆过滤器**：对常见 URL 做布隆过滤，不存在的直接返回 404 不走引擎 ② **空值缓存**：即使返回 404 也在 Redis 里缓存 30 秒，防止同一个 URL 反复穿透 ③ **本地缓存**：API 层加了一层 LRU 本地缓存（lru-cache 库），只有几百 KB，但能挡住 90% 的重复热点请求

**Q: 你们怎么做全链路压测？**

> 用 **locust**（Python 分布式压测工具）。在 staging 环境部署完整的 xcrawl 集群，写压测脚本模拟用户行为（scrape 不同页面、crawl 递归爬取、search 搜索）。压测时监控每个组件的资源水位，找到瓶颈组件再针对性优化。
>
> 压测维度：正常流量（100 QPS）→ 峰值流量（500 QPS）→ 极限流量（2000 QPS 直到系统扛不住）。每个阶梯跑 10 分钟，记录错误率和延迟。得到系统的**最大承载能力**和**安全水位线**。

---

## 点四：监控告警体系

### 标准话术

> 我搭建了 xcrawl 完整的 **四层监控体系**，覆盖指标、日志、链路、告警四个维度，目标是在 **5 分钟内发现并定位问题**。
>
> **第一层：指标监控（Metrics）**
>
> 选型是 **Prometheus + Nightingale（夜莺）**，不单纯用 Grafana 的原因是夜莺内置了告警引擎和管理告警规则的 API，更适合运维团队管理。
>
> **采集拓扑**：
>
> ```
> Categraf (DaemonSet, 每个节点一个)
>   ├── 采集系统指标 (CPU/内存/磁盘/网络/连接数)
>   └── remote write → n9e-center → remote write → Prometheus TSDB
>
> Prometheus (主动 scrape)
>   ├── kubelet /metrics/cadvisor → Pod 级别 CPU/内存/网络
>   ├── kube-state-metrics → K8s 对象状态
>   ├── xcrawl-api /metrics → 业务指标 (并发数/队列深度/请求延迟)
>   ├── node-exporter → 节点硬件指标
>   ├── rabbitmq-exporter → 队列指标
>   └── postgres-exporter → 数据库指标
> ```
>
> **关键指标清单**：
>
> | 类别 | 指标 | 含义 | 正常水位 |
> |------|------|------|----------|
> | 系统 | `cpu_usage_idle` | CPU 空闲率 | > 80% |
> | 系统 | `mem_used_percent` | 内存使用率 | < 80% |
> | 系统 | `disk_used_percent` | 磁盘使用率 | < 70% |
> | K8s | `up` | 所有 target 在线状态 | 全部 = 1 |
> | K8s | `kube_pod_status_phase` | Pod 状态分布 | Running = 目标数 |
> | K8s | `kube_node_status_condition` | Node 健康 | 6 个 Ready |
> | 容器 | `container_cpu_usage_seconds_total` | 容器 CPU 使用 | 按 +rate 换算 |
> | 容器 | `container_memory_working_set_bytes` | 容器实际内存 | 低于 limits 80% |
> | 容器 | `container_oom_events_total` | OOM 事件 | 必须 = 0 |
> | 业务 | `firecrawl_concurrent_active` | 活跃并发数 | < limits |
> | 业务 | `nuq_queue_scrape_job_count` | 队列深度 | < 500 |
> | 业务 | `firecrawl_request_duration_seconds` | API 延迟 P99 | < 3s |
> | 业务 | `firecrawl_request_errors_total` | 错误率 | < 1% |
> | 中间件 | `rabbitmq_queue_messages` | RabbitMQ 队列积压 | < 1000 |
> | 中间件 | `pg_stat_activity_count` | 数据库连接数 | < 50 |
>
> **总共 40+ scrape target**，全部正常 UP。
>
> **第二层：日志监控（Logging）**
>
> 我们用 **Loki + Promtail**（因为和 Prometheus 同一家公司出品，标签体系一致）。每个节点跑 Promtail DaemonSet，采集容器 Stdout 日志打到 Loki，Loki 按 Namespace + Pod 名 + Container 名做索引。
>
> 日志查询场景：① 关键字搜索（如搜索某个 jobId 的完整处理链条）② 按错误级别聚合（统计过去 1 小时 500 错误的次数和分布）③ 告警联动（夜莺告警触发时，自动附带 Loki 日志链接，方便值班人员查看上下文）
>
> **第三层：链路追踪（Tracing）**
>
> 我们用 **OpenTelemetry + Jaeger**。请求从 Ingress 进入时生成 traceId，通过 HTTP Header 逐层传递（Ingress → API → Worker → Playwright → DB）。每一层都记录 span：
> - Ingress 阶段：请求排队时间
> - API 阶段：参数校验 + 鉴权耗时
> - Worker 阶段：网络 IO 耗时、页面渲染耗时
> - Playwright 阶段：浏览器启动、JS 执行、截图生成
> - DB 阶段：查询耗时
>
> 如果某个请求慢了，直接在 Jaeger 上看火焰图，一眼就能看出瓶颈在哪里——是网络 IO 慢了还是下游引擎堵塞了。
>
> **第四层：告警体系**
>
> 夜莺（Nightingale）负责告警管理。告警规则分三个级别：
>
> | 级别 | 响应时间 | 通知方式 | 示例 |
> |------|----------|----------|------|
> | P0（严重） | 5 分钟内响应 | 电话 + 钉钉 + 群 @all | Node 宕机、API 全部超时、队列深度超 5000 |
> | P1（高） | 15 分钟内 | 钉钉 @值班人 | Pod 重启、错误率 > 1%、磁盘快满 |
> | P2（中） | 1 小时内 | 钉钉群通知 | 单台 Worker 延迟升高、某个客户错误率突增 |
>
> **告警通知去重和抑制**：同一个根源（比如 Node 宕机）会触发多个告警（Pod 不可用、目标 offline、队列积压），夜莺通过分组聚合只发一条通知，避免告警风暴。

### 面试官追问

**Q: 遇到过哪些告警相关的坑？**

> 遇到过两次：
>
> **告警风暴**：有一次升级 RabbitMQ，重启后所有 Worker 重连，短时间大量连接错误，触发了 Pod 不可用、目标 offline、错误率飙升、队列积压……十几个告警同时炸。解决：配置了夜莺的告警分组规则，同一资源（比如 Pod）的告警合并成一条。同时加了依赖抑制——如果 Node 宕机的告警已经触发了，Pod 级别的告警自动静默 30 分钟。
>
> **告警疲劳**：一开始阈值设得敏感，天天半夜收到告警但都是虚警。团队很快就麻木了，真正的故障来了反而没人响应。解决：花了 2 周时间调优阈值，用历史数据算出正常波动的上下界，阈值设在 3σ 之外。告警数量从日均 50 条降到 5 条，这 5 条里 80% 是真实问题。

**Q: Prometheus 存储不够了怎么办？**

> 我们 Prometheus 跑在 200GB 的磁盘上，retention 设 15 天。如果容量不够有几个方案：
>
> ① **降低指标基数**：去掉不必要的 label（比如某些 metric 带上了 `pod` 和 `container_id`，这些 label 基数很大）。把 pod label 里的 UID 去掉，序列直接从 10 万降到 2 万。
>
> ② **降采样**：老数据保留但不保留原始精度。原始数据保留 7 天（1s 精度），然后降采样到 1m 精度保留 14 天。用 VictoriaMetrics 或 Thanos 的 downsampling 功能。
>
> ③ **远端存储**：Prometheus 只保留 7 天热数据，通过 Thanos sidecar 上传到 S3/MinIO，查询时 Thanos Query 自动合并本地和远端数据。

**Q: 链路追踪对性能有影响吗？**

> 有，但可控。OpenTelemetry 默认采样的策略是**头部采样**（Head Sampling），我们配置了 10% 的采样率。高流量场景下 10% 足够排查问题，又不会给应用带来太大负担。如果某个特定 traceId 需要全量追踪，可以在请求头里加 `sampling.priority=1` 强制采样，用于调试特定用户的请求。

**Q: 你会看哪些 Grafana 大盘？**

> 分了 4 个维度：
>
> ① **业务大盘**：QPS、错误率、P50/P90/P99 延迟、并发数、队列深度。这个给值班人员和开发看，快速判断业务是否正常。
>
> ② **资源大盘**：每个 Node 的 CPU/内存/磁盘使用率、每个组件的容器资源使用率、OOM 事件。这个给 SRE 看，做容量规划。
>
> ③ **K8s 集群大盘**：Pod 状态分布、Node 状态、Deployment 副本数、PVC 使用率。这个给运维看，管理集群健康。
>
> ④ **中间件大盘**：RabbitMQ 队列深度、PostgreSQL 连接数/慢查询/死锁、Redis 命中率/内存使用。出问题的时候看这个大盘定位瓶颈。

**Q: 监控数据量太大怎么办？怎么降本增效？**

> 我们有 8 节点 × 5000+ 序列，每天产生 200GB+ 的监控数据。做了几层优化：
>
> ① **降低采样频率**：非关键指标从 15s 采集一次降到 60s。比如 `kube_pod_status_phase` 这种变化慢的指标，没必要 15s 一个点。
>
> ② **去掉高基数 Label**：`container_id`、`pod_uid`、`instance` 这种每个 Pod 唯一的 label 是最耗存储的。把不需要的 label 通过 `metric_relabel_configs` 在 scrape 时就丢弃。
>
> ③ **数据分层**：热数据（7 天）保留全精度，冷数据（7-30 天）降采样到 5 分钟一个点。我们用了 Thanos 的 downsampling 把旧数据压缩到 S3 上。
>
> ④ **评估投入产出比**：某些指标（比如每个 HTTP 请求的 headers）虽然能看到但根本没人查。定期审查指标清单，去掉"好看但没用"的指标。

**Q: 业务指标怎么接入 Prometheus 的？**

> xcrawl 每个组件都暴露了一个 `/metrics` HTTP 端点（用 prom-client 库），Prometheus 通过 Service 自动发现来 scrape。
>
> 自定义的业务指标：
> ```javascript
> const client = require('prom-client');
> const concurrentGauge = new client.Gauge({
>   name: 'xcrawl_concurrent_active',
>   help: '当前活跃请求数'
> });
> const queueGauge = new client.Gauge({
>   name: 'xcrawl_queue_depth',
>   help: '待处理队列深度',
>   labelNames: ['queue_type']
> });
> const requestDuration = new client.Histogram({
>   name: 'xcrawl_request_duration_seconds',
>   help: '请求耗时分布',
>   buckets: [0.1, 0.5, 1, 2, 5, 10, 30]
> });
> ```
>
> Prometheus 配置里用 `kubernetes_sd_configs` 自动发现带有 `prometheus.io/scrape: "true"` 注解的 Pod，不需要手动维护 target 列表。

**Q: 告警规则是怎么设计出来的？踩过什么坑？**

> 设计原则：**宁可漏报，不要误报**。误报多了团队就麻了。
>
> 具体方法：
> ① 先采集一个月的历史数据，看每个指标的正常波动范围
> ② 阈值设在正常波动的 3σ（标准差）之外，确保正常波动不会触发
> ③ 加了持续时间条件，比如 `持续 5 分钟` 才告警，避免抖动导致误报
> ④ 告警消息里带上 Grafana 大盘链接和当前值，值班人员点开就能看上下文
>
> 踩过的坑：一开始对 `container_memory_working_set_bytes` 做了绝对值告警（> 2GB 告警），结果扩容后每个 Pod 的内存分配变了，阈值就不准了。后来改成百分比：`> 80% of limits`，通用性更好。

**Q: 怎么做 On-Call 的？值班怎么排的？**

> 团队 5 个人轮值，每人一周，24 小时 On-Call：
> - P0 告警：电话 + 钉钉 @all，15 分钟内必须响应
> - P1 告警：钉钉 @值班人，30 分钟内响应
> - P2 告警：次日上班处理
>
> 轮值表用 **PagerDuty** 管理（或者用简单的钉钉机器人 + 排班表）。交接班时写值班日报，记录当天告警和处理情况。
>
> 为了防止疲劳，每人 On-Call 后第二天可以晚到或者远程办公。

---

## 面试补充：迁移 xcrawl 到 Firecrawl 的考虑

> 如果面试官问"你了解 Firecrawl 吗？"或者"对比过哪些竞品？"，你可以说：

> 我在调研 xcrawl 的架构优化方向时，深入对比了 Firecrawl。Firecrawl 和 xcrawl 的架构非常相似——都是 API Gateway + 消息队列 + Worker + 浏览器渲染的模型。我甚至在公司 staging 环境完整部署了一套 Firecrawl 做对比测试。
>
> **Firecrawl 的优点**：
> - 开源，社区活跃，更新快
> - 内置了 AI Agent（`/v2/agent`），可以用自然语言描述抓取需求
> - Playwright 集成开箱即用
>
> **如果让我重新设计 xcrawl，我会借鉴 Firecrawl 的几个设计**：
> ① NUQ 的消息队列设计（基于 Redis + RabbitMQ 的混合方案，比 xcrawl 当前方案更均衡）
> ② Prometheus 业务指标暴露（Firecrawl 每个组件主动暴露 /metrics，方便我们做监控）
> ③ WebSocket 实时推送爬取进度（这是 xcrawl 当前没有的）
>
> 不过我最终没有选 Firecrawl 直接替代 xcrawl，因为 xcrawl 在代理池管理、客户计费、API Key 鉴权方面有多年积累，直接迁移成本太高。我们走的是**逐步吸收 Firecrawl 的组件和技术方案**到 xcrawl 里的路线。

---

## 面试技巧总结

### 说数字（面试官喜欢听到具体数字）

| 场景 | 你说 |
|------|------|
| 集群规模 | 2 Master + 6 Worker，共 8 台 |
| 请求量 | 日均数百万次 scrape / crawl 请求 |
| 组件数量 | 8 个微服务组件，共 20+ Pod |
| 监控指标 | 40+ scrape target，5000+ 时间序列 |
| 告警量 | 调优后日均 5 条有效告警 |
| 扩容时间 | K8s HPA 自动扩容 < 1 分钟 |
| 回滚时间 | Helm 回滚 < 30 秒 |
| 故障定位 | 5 分钟内完成 |

### 说踩过的坑（面试官最想听这部分）

> **1. containerd 和 Categraf Docker socket 冲突**
>
> 把 Docker 换成 containerd 时，Categraf 默认采集 `/var/run/docker.sock`，启动报错。排查了半小时才发现是 docker_socket 配置没关。
>
> **2. Prometheus ServiceAccount 没有 RBAC 权限**
>
> 部署 Prometheus 后 scrape kubelet 发现全 403，查了文档发现 Prometheus 的 ServiceAccount 默认没有 RBAC，需要手动绑 ClusterRole。最简单是绑 cluster-admin，但安全的做法是自建一个只读 ClusterRole。
>
> **3. NUQ 连接池爆满导致队列积压**
>
> 业务高峰期，NUQ Worker 数量没变，但任务量翻了 3 倍，连接池打满，新任务排队。临时扩容 + 调大连接池解决，后来加了自动扩缩容 HPA。
>
> **4. nginx proxy_pass 斜杠问题**
>
> Nginx 配置 `proxy_pass http://upstream/` 带斜杠会截掉 location 匹配的 path 前缀，导致路由 404。这种坑只有踩过才知道。

### 面试禁忌

> ❌ **不要说"我不清楚"**——可以说"这个场景我还没遇到过，但我的思路是……"
> ❌ **不要说"我们用的 Firecrawl"**——你用的是 xcrawl，Firecrawl 只是调研对比过
> ❌ **不要把话讲得太完美**——适当说踩过的坑和优化过程，显得更真实
> ❌ **不要背稿子**——记住框架和关键数字，用自己的话讲出来

---

## 附录：K8s 面试高频追问速查

> 以下是面试中 K8s 方向最容易被问到的题目，每个用一句话讲核心，再展开讲关键点。

### Q: K8s 架构讲一下

> **一句话**：K8s 是 Master-Worker 架构。Master 跑控制面组件（apiserver、etcd、scheduler、controller-manager），Worker 跑业务 Pod（kubelet、kube-proxy、容器运行时）。
>
> **展开讲**：apiserver 是所有组件的唯一入口，etcd 存集群状态，scheduler 把 Pod 分配到合适的 Worker，controller-manager 确保实际状态和期望状态一致。Worker 上的 kubelet 负责管理 Pod 生命周期，kube-proxy 做网络转发。

### Q: Pod 创建流程讲一下

> **一句话**：用户 `kubectl apply` → apiserver 验证写入 etcd → scheduler 调度到 Worker → kubelet 调 CRI 创建容器 → kube-proxy 更新网络规则。
>
> **展开讲**：Pod 创建后经历 Pending（调度中）→ ContainerCreating（拉镜像）→ Running（正常运行）三个阶段。如果卡在 Pending 一般是资源不够，卡在 ContainerCreating 一般是镜像拉不下来。

### Q: Service 有哪些类型？底层怎么实现的？

> **一句话**：ClusterIP（集群内访问）、NodePort（节点端口暴露）、LoadBalancer（云 LB）、ExternalName（DNS 别名）。
>
> **展开讲**：ClusterIP 分配一个虚拟 IP，kube-proxy 在每台机器上写 iptables/IPVS 规则做 DNAT 到后端 Pod。NodePort 是在 ClusterIP 基础上，每台机器的固定端口上也监听到 ClusterIP 的转发。LoadBalancer 是 NodePort + 云厂商 LB。

### Q: Ingress 和 Service 的区别？

> **一句话**：Service 是四层（TCP/UDP）负载均衡，Ingress 是七层（HTTP/HTTPS）路由——能根据域名、路径做转发，能终止 TLS。
>
> **展开讲**：Service 适合内部服务发现和简单的四层暴露，Ingress 适合外部 HTTP 流量管理（多个域名、路径路由、SSL 证书）。我们生产环境 Ingress 配了 Let's Encrypt 自动续签证书。

### Q: ConfigMap 和 Secret 的区别？

> **一句话**：ConfigMap 存明文配置，Secret 存敏感数据（base64 编码）。Secret 可以加加密（Sealed Secrets / Vault）。
>
> **展开讲**：两者都可以挂载为环境变量或文件。ConfigMap 用来存配置（API 地址、日志级别），Secret 存密码和 API Key。注意 Secret 的 base64 不是加密，只是编码。

### Q: 污点和容忍度（Taint & Toleration）怎么用的？

> **一句话**：给 Node 打 Taint（污点），只有 Toleration（容忍）匹配的 Pod 才能调度上去。
>
> **展开讲**：Master 节点默认有 `node-role.kubernetes.io/master:NoSchedule` 标签，业务 Pod 调度不上去。反过来的用法：给专用 GPU 节点打 Taint，只有加了对应 Toleration 的 AI 任务才能调度上去，确保普通任务不会占用 GPU 资源。

### Q: 亲和性（Affinity）用过吗？

> **一句话**：节点亲和（nodeAffinity）让 Pod 调度到特定节点，Pod 亲和（podAffinity）让相关 Pod 调度到一起，Pod 反亲和（podAntiAffinity）让同类 Pod 分散在不同节点。
>
> **展开讲**：我们给 NUQ Worker 配了 podAntiAffinity，确保多个 Worker 副本分散到不同 Node 上，单个 Node 挂了不会导致全部 Worker 不可用。给 RabbitMQ 和 PostgreSQL 配了 nodeAffinity，固定到大容量节点上。

### Q: Init Container 用来做什么？

> **一句话**：在主容器启动前执行的初始化操作，完成后自动退出。
>
> **展开讲**：我用来做数据库迁移（`yarn migrate`）和 ConfigMap 的模板渲染。Init Container 支持重启和超时，如果初始化失败主容器不会启动。

### Q: livenessProbe 和 readinessProbe 的区别？

> **一句话**：liveness 检查失败重启 Pod，readiness 检查失败从 Service 摘除流量。
>
> **展开讲**：liveness 是用来检测"进程是否死锁"的，readiness 是用来检测"服务是否就绪"的。xcrawl-api 的 readiness 配的是 `GET /v0/health/readiness`（检查依赖的中间件是否都连得上），liveness 配的是 `GET /v0/health/liveness`（只检查进程本身是否存活）。

### Q: Resource Quota 和 LimitRange 怎么用的？

> **一句话**：ResourceQuota 限制 Namespace 级别的总资源用量，LimitRange 给没有设 limits 的 Pod 设置默认值。
>
> **展开讲**：我给测试环境配了 ResourceQuota——整个 Namespace 最多用 8 核 16GB 内存，防止某个人的测试任务把集群资源吃光。LimitRange 给没配资源的开发者 Pod 自动补一个默认的 limits/requests。

### Q: 安全的 Secret 管理方案？

> **一句话**：K8s 原生 Secret 只做了 base64，不是真安全。我们用 Sealed Secrets + Vault。
>
> **展开讲**：Sealed Secrets 的方案是 Git 仓库里存加密的 YAML，只有集群里的 controller 能解密。Vault 的方案是 Pod 启动时通过 CSI 驱动从 Vault 拉取密钥注入成文件。生产环境的核心凭证走 Vault，普通配置走 Sealed Secrets。

### Q: K8s 日志架构了解吗？怎么做的？

> **一句话**：K8s 本身不存日志，靠附加组件（Loki/ES + Promtail/Filebeat）采集容器的 Stdout 和文件日志。
>
> **展开讲**：容器的 Stdout 日志在宿主机上是 `/var/log/pods/<namespace>_<pod>_<uid>/<container>/0.log`。Promtail 采集后按 Namespace + Pod 名加标签，打到 Loki。Loki 和 Prometheus 共用同一套标签体系，方便指标和日志联动。

### Q: 怎么做 Pod 资源隔离？

> **一句话**：requests（保证下限） + limits（硬上限） + ResourceQuota（Namespace 级总上限）。
>
> **展开讲**：xcrawl-api 配了 `requests: 500m CPU, 512Mi 内存` + `limits: 2 CPU, 2Gi 内存`。requests 保证最少资源，limits 防止某个 Pod 把其他 Pod 的资源吃掉。ResourceQuota 是安全网，防止整个 Namespace 超卖。

### Q: 节点故障自动修复了解吗？

> **一句话**：Node Problem Detector + Node Lifecycle Controller + Descheduler。
>
> **展开讲**：NPD 检测内核问题、Docker hang、硬件故障等底层问题并上报事件。K8s 的 Node Controller 在 Node 心跳丢失 5 分钟后驱逐 Pod。Descheduler 可以把负载不均衡的 Pod 重新调度。

### Q: Headless Service 用过吗？

> **一句话**：不分配 ClusterIP 的 Service，通过 DNS 返回所有 Pod IP，用于 StatefulSet 的有状态服务发现。
>
> **展开讲**：RabbitMQ 集群我们就用了 Headless Service。客户端通过 DNS 拿到所有 RabbitMQ 节点 IP，然后用 `rabbitmqctl join_cluster` 做集群组网。

### Q: 怎么给 K8s 做备份的？

> **一句话**：etcd 备份（集群状态）+ Velero（资源对象 + PV 备份）。
>
> **展开讲**：etcd 每天凌晨 CronJob 做 snapshot 存到 NAS，保留 30 天。Velero 每周全量备份所有 Namespace 的 YAML 资源到 S3/MinIO，保留 3 个月。PV 的数据（PostgreSQL、Redis）用 Velero 的 restic 插件做文件级备份。

### Q: K8s 排障的通用思路？

> **一句话**：从底层往上层排：Node → Pod → 容器 → 网络 → 应用。
>
> **展开讲**：
> ```
> Pod 起不来？
>   → kubectl describe pod <name>  # 看 Events
>   → kubectl logs <name>          # 看应用日志
>   → kubectl exec -it <name> sh   # 进容器检查
> 
> 网络不通？
>   → kubectl get endpoints <svc>   # 检查后端 Pod 有没有注册
>   → kubectl exec -it <pod> -- nslookup <svc>  # DNS 是否正常
>   → kubectl run -it --rm busybox -- sh  # 临时 Pod 测试连通性
> 
> 节点异常？
>   → kubectl describe node <name>  # 看 Condition 和事件
>   → kubectl get pods --field-selector spec.nodeName=<name>  # 看节点上的 Pod
>   → ssh 进节点检查 kubelet 和 containerd 日志
> ```

### Q: K8s 认证授权怎么做的？

> **一句话**：认证（你是谁）→ 鉴权（你能做什么）→ 准入控制（还能做什么）。
>
> **展开讲**：认证支持客户端证书、Bearer Token、静态密码文件。鉴权支持 RBAC（最常用）、ABAC（灵活性差）、Node（节点自身权限）、Webhook（外部鉴权）。我们用的是 RBAC + 客户端证书。给不同角色建了 ServiceAccount：开发只能 get pod 和看日志，运维能 manage 大多数资源，admin 能操作全部。

### Q: Deployment / StatefulSet / DaemonSet 区别？

> **一句话**：Deployment 管无状态应用（随意扩缩、随便重建），StatefulSet 管有状态应用（稳定网络标识 + 有序启停），DaemonSet 保证每个节点跑一个 Pod（日志采集、监控 agent）。
>
> **展开讲**：xcrawl-api 用 Deployment，支持滚动更新和快速扩缩。RabbitMQ 和 PostgreSQL 用 StatefulSet，保证 Pod 名不变、PVC 不丢。Categraf/Promtail 用 DaemonSet，每台机器都得跑一个采集器。

### Q: 滚动更新策略怎么配的？

> **一句话**：`maxSurge` 控制最多比期望多起几个，`maxUnavailable` 控制最多允许多少个不可用。
>
> **展开讲**：生产配置：
> ```yaml
> strategy:
>   type: RollingUpdate
>   rollingUpdate:
>     maxSurge: 1        # 滚动中最多多起 1 个新 Pod
>     maxUnavailable: 0   # 必须保证全部 Pod 可用
> ```
> 这样更新时先起一个新 Pod（旧的 3 个还在服务），新 Pod Ready 后再停一个旧的，直到全部替换完。流量零中断。如果 maxUnavailable 设成 1，更新更快但短时间少一个 Pod 的容量。

### Q: Deployment 的发布策略有哪些？

> **一句话**：RollingUpdate（滚动，默认）、Recreate（全部停掉重建）、蓝绿/金丝雀（需要额外组件）。
>
> **展开讲**：RollingUpdate 最常用，逐批替换，零宕机。Recreate 用于数据迁移或重大版本变更（比如改数据库 Schema），先停所有旧版本再起新版本，但会有停机时间。蓝绿和金丝雀需要靠 Ingress/Service 配合做流量分发。

### Q: Pod 的生命周期有哪些阶段？

> **一句话**：Pending → ContainerCreating → Running → Succeeded / Failed / CrashLoopBackOff。
>
> **展开讲**：
> - Pending：调度中或镜像拉取中
> - ContainerCreating：容器启动中
> - Running：正常运行
> - Succeeded：正常完成（Job）
> - Failed：异常退出
> - CrashLoopBackOff：反复崩溃，K8s 在退避等待
> - Unknown：节点失联

### Q: Pod 重启策略有哪些？

> **一句话**：Always（总是重启，默认）、OnFailure（失败时重启）、Never（永不重启）。
>
> **展开讲**：xcrawl-api 用 Always，保证进程挂了自动拉起。CronJob 用 OnFailure 或 Never，让任务跑完就退出。sidecar 容器通常用 Always。

### Q: requests 和 limits 的区别？配多少合适？

> **一句话**：requests 给调度器用的告诉它最少要这么多资源，limits 给 kubelet 用的限制最多能用这么多。
>
> **展开讲**：requests 设多少决定了调度器能不能把 Pod 放到某个 Node 上。limits 是硬限制——内存超了会 OOMKill，CPU 超了会节流。生产环境建议 requests = limits（Guaranteed QoS），或者 requests < limits（Burstable QoS）。xcrawl-api 配的 requests: 1C/1G, limits: 2C/2G，属于 Burstable。因为业务流量有波峰波谷，允许它突增但不能吃光机器。

### Q: QoS 等级有哪些？

> **一句话**：Guaranteed（requests = limits）、Burstable（requests < limits）、BestEffort（不设 requests/limits）。
>
> **展开讲**：Guaranteed 的 Pod 资源最有保障，OOM 时最后被杀。BestEffort 的 Pod 资源不受保障，OOM 最先被杀。我们给核心组件（API）配 Guaranteed，辅助组件（CronJob）配 Burstable，测试任务用 BestEffort。

### Q: Static Pod 用过吗？

> **一句话**：由 kubelet 直接管理、不经过 apiserver 的 Pod，YAML 放在节点的 `/etc/kubernetes/manifests/` 目录下。
>
> **展开讲**：K8s 自己的控制面组件（kube-apiserver、etcd、kube-scheduler）都是以 Static Pod 方式跑在 Master 节点上的。运维排查时如果 kubelet 挂了，可以直接 ssh 到节点上改 manifest 文件，kubelet 自动感知重启。

### Q: Sidecar 模式了解吗？

> **一句话**：一个 Pod 里跑多个容器，主容器做业务，sidecar 容器做辅助功能（日志转发、代理、监控采集）。
>
> **展开讲**：我们给 xcrawl-api 配了一个 sidecar 做日志采集（fluent-bit），把 Pod 里的应用日志打到 Loki，不占用业务容器的资源。也见过 istio 的 envoy sidecar 做服务网格流量劫持。

### Q: Pod 里的多个容器怎么通信？

> **一句话**：同一个 Pod 的多个容器共享网络命名空间（localhost 通信）和存储卷（共享磁盘）。
>
> **展开讲**：sidecar 和主容器通过 localhost:端口 通信，不需要通过 Service 做服务发现。共享的 emptyDir 卷可以做日志中转——主容器写日志到文件，sidecar 读取转发。注意端口不能冲突。

### Q: emptyDir / hostPath / PVC 的区别？

> **一句话**：emptyDir 跟随 Pod 生命周期（Pod 删数据丢），hostPath 用节点目录（节点限制），PVC 是持久化存储（数据不丢）。
>
> **展开讲**：emptyDir 用来做临时缓存（比如 nginx 缓存反向代理的数据），hostPath 用来读节点文件（比如 Promtail 读容器日志），PVC 用来存数据库数据。生产环境核心数据一定用 PVC，测试环境用 emptyDir 没问题。

### Q: PV 的 AccessMode 有哪些？

> **一句话**：ReadWriteOnce（单节点读写）、ReadOnlyMany（多节点只读）、ReadWriteMany（多节点读写）。
>
> **展开讲**：RWO 适合块存储（云盘），一个 Pod 独占读写。ROX 适合日志/数据分发场景。RWX 适合 NAS/NFS，多个 Pod 同时读写。xcrawl 的 PostgreSQL 用 RWO，日志数据用 RWX（NAS）。

### Q: StorageClass 的 WaitForFirstConsumer 和 Immediate 的区别？

> **一句话**：Immediate 创建 PVC 就绑定 PV，WaitForFirstConsumer 等 Pod 调度到节点后才绑定。
>
> **展开讲**：local-path-provisioner 必须用 WaitForFirstConsumer（Pod 调度到哪个节点，PVC 就绑哪个节点的磁盘），否则 PVC 不知道绑到哪台机器上。NAS 存储用 Immediate 就够了，所有节点都能访问。

### Q: Kustomize 和 Helm 你用的哪个？

> **一句话**：Helm 是包管理 + 模板引擎，适合复杂应用。Kustomize 是纯 Yaml Patch，适合简单的环境差异化。
>
> **展开讲**：xcrawl 用 Helm 管理（环境多、组件多、依赖复杂）。Kustomize 用在一些小项目或者云原生生态的配置（ArgoCD、Prometheus Operator）。Helm 能处理依赖（子 chart）、条件控制（if/else）、模板函数，Kustomize 的 overlay 适合简单的环境覆盖。

### Q: Operator 模式了解吗？

> **一句话**：用自定义控制器管理复杂的有状态应用（比如 prometheus-operator、rabbitmq-operator）。
>
> **展开讲**：Operator 封装了应用的运维知识——扩缩容、升级、备份恢复都自动化了。我们用 prometheus-operator 管理 Prometheus 集群，用 rabbitmq-operator 管理 RabbitMQ 集群。自己没写过 Operator，但理解它的核心模式：CRD + Controller。

### Q: K8s 调度器是怎么工作的？

> **一句话**：Predicates（过滤不满足条件的节点）→ Priorities（给剩余节点打分）→ 选最高分的节点调度。
>
> **展开讲**：Predicates 阶段检查节点资源够不够、端口冲不冲突、Taint/Toleration 匹配不匹配。Priorities 阶段计算资源利用率、Pod 亲和性等权重。如果默认调度器不满足需求，可以写自定义 scheduler 扩展。

### Q: 服务网格（Service Mesh）用过吗？

> **一句话**：没在生产用，但调研过 Istio——在应用无感的情况下做流量管理、可观测性、安全。
>
> **展开讲**：Service Mesh 的核心是 sidecar 代理（Envoy）劫持所有进出流量，实现熔断、重试、指标采集、mTLS。我们没上生产的原因是引入复杂度太高（每个 Pod 多一个 Envoy 容器，资源开销 +20%），当时团队规模支撑不起。如果有专门的平台团队，Istio 值得上。

### Q: K8s 怎么做到无损下线的？

> **一句话**：preStop hook + 优雅关闭（SIGTERM 处理）+ readiness 提前摘流量。
>
> **展开讲**：Pod 被删除时的流程：
> 1. Pod 变成 Terminating 状态，从 Service Endpoints 摘除（新流量不进来）
> 2. 执行 preStop hook（比如 sleep 5s 等待已有请求处理完）
> 3. K8s 发 SIGTERM 信号给主进程
> 4. 应用收到 SIGTERM 后停止接受新连接，处理完正在处理的请求后退出
> 5. 默认 30 秒后发 SIGKILL 强制杀死
>
> ```yaml
> lifecycle:
>   preStop:
>     exec:
>       command: ["/bin/sh", "-c", "sleep 10"]
> ```
> 这个 10 秒的 sleep 让 Readiness 探针有足够时间把 Pod 从 Service 摘除，保证流量不中断。

### Q: PodDisruptionBudget（PDB）怎么用的？

> **一句话**：保证 voluntary disruption（主动干扰，如节点维护）时最多允许多少个 Pod 不可用。
>
> **展开讲**：配了 `minAvailable: 2`，意味着 3 副本的 API 在任何时候至少要有 2 个可用。节点维护（drain）时会遵守 PDB，不会一下子把 3 个 Pod 全干掉。没配 PDB 的话 drain 可能一次干掉所有副本。

### Q: 安全上下文（SecurityContext）怎么配的？

> **一句话**：限制容器能用什么用户跑、能不能写根文件系统、能不能提权。
>
> **展开讲**：xcrawl 的 Pod 配置了：
> ```yaml
> securityContext:
>   runAsNonRoot: true
>   runAsUser: 1001
>   fsGroup: 1001
>   capabilities:
>     drop: ["ALL"]
> ```
> 禁止 root 用户运行容器，容器内的进程 uid 是 1001，提升安全性。如果容器需要绑定特权端口（< 1024），才需要加 `NET_BIND_SERVICE` 能力。

### Q: ConfigMap 有几种挂载方式？

> **一句话**：环境变量注入、文件挂载（整个 volume 或指定 key）、命令行参数。
>
> **展开讲**：xcrawl 三种都用：环境变量（`envFrom` 直接挂整个 ConfigMap 为环境变量）、文件挂载（Nginx 配置文件用 subPath 挂到 `/etc/nginx/nginx.conf`）、命令行参数（通过 `${ARGS}` 引用）。注意环境变量方式不适合大 ConfigMap，文件挂载改了 ConfigMap Pod 不会自动 reload。

### Q: Namespace 不能删除怎么办（Terminating 卡住）？

> **一句话**：finalizer 没清理完，找哪个资源还挂在 Namespace 下，手动删 finalizer。
>
> **展开讲**：`kubectl get namespace <name> -o json | jq '.spec.finalizers = []' | kubectl replace --raw /api/v1/namespaces/<name>/finalize -f -`。原因是某些 CRD 资源的 controller 已经不在了，finalizer 一直没被执行。这是常见的 K8s 集群清理问题。

### Q: 怎么做压力测试的？

> **一句话**：用 locust 或 wrk2 模拟真实流量，打满集群看极限在哪里。
>
> **展开讲**：我们在 staging 环境做压测。locust 脚本模拟用户行为（scrape 不同页面、crawl 递归爬取）。压测分阶梯：100 QPS → 500 QPS → 1000 QPS，每阶梯跑 5 分钟，记录延迟和错误率。得到结果：单 API Pod 扛 200 QPS（P99 < 2s），3 副本扛 600 QPS。超过 600 QPS 错误率开始上升，说明需要扩容。
>
> 压测时也顺便验证了 HPA——流量上来后 CPU 超过 70% 自动触发扩容，新 Pod 启动后压力降回正常水平。

### Q: K8s 1.29 相比旧版本有什么新特性？

> **一句话**：Sidecar Containers 成为 Beta、ReadWriteOncePod 访问模式 GA、NodeExpansionSecret GA。
>
> **展开讲**：Sidecar Containers（1.29 Beta）可以用 `restartPolicy: Always` 的 Init Container 实现原生 sidecar 支持，不再需要额外工具。ReadWriteOncePod 允许一个 PV 只能被一个 Pod 挂载，避免 PostgreSQL 多 Pod 写同一个盘的坑。这些新特性我目前在规划用上，特别是 sidecar 那个能简化日志采集器的部署。

### Q: 容器和虚拟机的区别？

> **一句话**：VM 虚拟硬件（Hypervisor）→ 每个 VM 一个完整 OS → GB 级 → 分钟级启动。容器共享宿主机内核 → 进程级隔离 → MB 级 → 秒级启动。
>
> **展开讲**：VM 隔离性好但资源开销大（每个 VM 都要跑一个完整 OS），适合不同租户的强隔离场景。容器隔离性弱一点（共享内核）但启动快资源利用率高，适合微服务和 CI/CD。K8s 的设计就是最大化容器的优势。

### Q: kube-proxy 的三种模式？

> **一句话**：userspace（用户态转发，旧/慢）、iptables（默认，规则量大时性能下降）、IPVS（性能最好，支持更多负载均衡算法）。
>
> **展开讲**：小集群 iptables 够用。当集群规模超过 1000 个 Service 时，iptables 规则数量会导致更新延迟和 CPU 开销。IPVS 用哈希表替代链式规则，性能更稳定。我们 8 节点集群，Service 不到 50 个，iptables 完全够用。

### Q: kubelet 的作用？

> **一句话**：每台节点上的"管家"——负责 Pod 的创建、监控、重启，并定时汇报 Node 状态给 Master。
>
> **展开讲**：kubelet 不断 watch apiserver 上分配给本节点的 Pod，发现新 Pod 就调 CRI 创建容器，定期执行 liveness/readiness 探针，把 Node 的状态和资源使用量上报。kubelet 挂了这台节点上的 Pod 就不会被管理了。

### Q: 描述一下 K8s 网络模型

> **一句话**：每个 Pod 有独立 IP，Pod 之间可以直接通信（不经过 NAT），节点也可以和 Pod 直接通信。
>
> **展开讲**：这是 K8s 网络的基础约定，由 CNI 插件实现。我们用的 Calico 通过 BGP 路由把 Pod IP 广播到网络里，所以 Pod 和 Pod 之间、Pod 和 Node 之间都是直连路由，没有 NAT 转换的性能损耗。Service 的 ClusterIP 是虚拟 IP，靠 kube-proxy 做 DNAT。

### Q: CRI / OCI / CNI 都是什么？

> **一句话**：CRI 是 K8s 和容器运行时的接口标准，OCI 是容器镜像和运行时的行业标准，CNI 是容器网络接口标准。
>
> **展开讲**：K8s 通过 CRI 接口调 containerd → containerd 按 OCI 标准启动容器（runc）。CNI 插件（Calico）给容器分配 IP 并配置网络。这三层互相独立——换容器运行时（containerd ↔ CRI-O）不用改 K8s，换网络插件（Calico ↔ Flannel）不用改 containerd。

### Q: 多集群管理怎么做？

> **一句话**：每个集群独立管理，用 kubectl context 切换，或者用 Rancher/ArgoCD 统一管理。
>
> **展开讲**：我们有 2 个集群（生产 + 测试），通过 `kubectl config use-context` 切换。ArgoCD 把多个集群的部署统一管理——Git 仓库是唯一真相源，ArgoCD 自动同步到各个集群。长期规划是用 ArgoCD 做多集群的 GitOps。

### Q: 怎么做资源成本核算？

> **一句话**：用 Kubecost 或自建脚本，按 Namespace/Deployment 拆分 CPU 和内存的 requests 来算成本。
>
> **展开讲**：我们自建了一个脚本，每天从 Prometheus 取 `kube_pod_container_resource_requests`，按 Namespace + Deployment 汇总，乘以云厂商的单价，算出每个团队的资源消耗。测试环境还加了 "浪费资源" 的统计——requests 设了但实际没用到的资源，推动开发减小 requests 节省成本。

### Q: 如果 apiserver 挂了怎么办？

> **一句话**：kubectl 不能用，但集群上已有的 Pod 和服务不受影响——只是不能创建/更新/删除资源。
>
> **展开讲**：apiserver 挂了只会影响控制面操作（部署、扩缩容、滚动更新），已经在跑的 Pod 和 Service 流量完全正常。恢复方式：如果是 apiserver Pod 崩溃，kubelet 会自动重启（Static Pod）。如果是 etcd 挂了，需要恢复 etcd 快照。我们有 etcd 每天备份，恢复通常在 10 分钟内。

### Q: Static Pod 和 Deployment 创建的 Pod 有什么区别？

> **一句话**：Static Pod 由 kubelet 直接创建管理，不经过 apiserver，没有 Deployment/ReplicaSet/Service 关联。
>
> **展开讲**：Static Pod 的 YAML 放在节点的 `/etc/kubernetes/manifests/` 目录下，kubelet 自动检测启动。不能通过 `kubectl delete` 删除（删了 kubelet 会自动重新创建），只能删文件。K8s 的控制面组件就是用 Static Pod 跑的——哪怕集群坏了，Master 节点的 kubelet 还能拉起控制面。

### Q: 描述一次你自己排查过的最复杂的 K8s 问题

> **一句话**：一次 Pod 间歇性断连的问题——Service 能解析但不能访问，curl 卡住直到超时。
>
> **展开讲**：排查了三天，最终发现是 Calico 的 IPVS 连接跟踪表满了。症状：Pod 间通信偶尔卡死，重启 Pod 后恢复但几分钟后复现。排查路径：看 kube-proxy 日志 → 看 conntrack 表 → `conntrack -S` 发现 `insert_failed` 在增长 → 原因是 Node 上的连接跟踪表太小（默认 65536），高并发爬虫场景下几分钟就满了。解决：`sysctl -w net.netfilter.nf_conntrack_max=262144`，同时调短 conntrack 超时时间。

### Q: Docker 和 containerd 的区别？

> **一句话**：Docker 是一整套容器工具链（build、ship、run），containerd 只是容器运行时（run 的部分）。
>
> **展开讲**：Docker 里面有 containerd（Docker 的运行时层），但 Docker 额外有 dockerd（API 守护进程）、docker build 等。K8s 只需要 run 的部分，所以直接用 containerd 更轻量——少一层 dockerd 转发，资源开销更少。我们迁移后 kubelet 日志里的 "连不上 docker" 错误没了，Pod 启动也快了。

### Q: etcd 的架构了解吗？

> **一句话**：etcd 是 K8s 的数据库，Raft 一致性算法保证数据不丢，所有集群状态都存里面。
>
> **展开讲**：etcd 用 Raft 做分布式共识，一般部署 3 或 5 节点奇数个。读写性能是关键——写慢了 apiserver 就慢了。常见问题：磁盘 IO 太高导致 etcd 写入延迟飙升 → 整个集群变慢。所以我们把 etcd 的 WAL 和 data 放在单独的 SSD 上，还配了 `--quota-backend-bytes=8GB` 防止 etcd 磁盘打满。

### Q: 你接触过的最大集群规模？

> **一句话**：当前是 8 节点集群，之前在其他公司接触过 50 节点的生产集群。
>
> **展开讲**：50 节点集群面临的问题和 8 节点不一样——CoreDNS 缓存压力、etcd watch 数量过多、kube-proxy iptables 更新延迟。解决方法：CoreDNS 加 autoscaler、etcd 调大 `--max-request-bytes`、kube-proxy 切到 IPVS 模式。不过 8 节点集群当前这些都不是瓶颈。
>
