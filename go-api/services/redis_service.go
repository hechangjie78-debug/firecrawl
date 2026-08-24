package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/firecrawl/go-api/models"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

const (
	// RedisKeyPrefixCrawlJob CrawlJob 存入 Redis Hash 的前缀 Key
	RedisKeyPrefixCrawlJob = "firecrawl:crawl_job:"
	// DefaultJobExpiration 默认爬虫任务结果在 Redis 中保存 24 小时
	DefaultJobExpiration = 24 * time.Hour
)

// RedisService 封装基于 Go-Redis 的高并发状态与数据持久化逻辑
type RedisService struct {
	client *redis.Client
}

// NewRedisService 初始化 Redis 客户端并验证网络连通性
func NewRedisService() (*RedisService, error) {
	redisURL := os.Getenv("REDIS_URL")
	var opts *redis.Options
	var err error

	if redisURL != "" {
		opts, err = redis.ParseURL(redisURL)
		if err != nil {
			log.Warn().Err(err).Str("url", redisURL).Msg("解析 REDIS_URL 失败，降级使用默认 localhost 配置")
			opts = &redis.Options{Addr: "localhost:6379"}
		}
	} else {
		opts = &redis.Options{
			Addr: "localhost:6379",
		}
	}

	client := redis.NewClient(opts)

	// 发起 Ping 连通性测试 (超时 3 秒)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Warn().Err(err).Msg("主 Redis (209.33.176.8) 连接失败，尝试自动尝试本地/内置 Redis 实例")
		opts = &redis.Options{Addr: "localhost:6379"}
		client = redis.NewClient(opts)
		ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel2()
		if err2 := client.Ping(ctx2).Err(); err2 != nil {
			log.Warn().Err(err2).Msg("本地 Redis 亦不可用，切换至高效内存模式处理")
			return nil, err2
		}
	}

	log.Info().Str("addr", opts.Addr).Msg("Redis 客户端初始化并成功建立连接池！")
	return &RedisService{client: client}, nil
}

// SaveCrawlJob 序列化保存整个 CrawlJob 至 Redis 且设置 24 小时自动过期
func (r *RedisService) SaveCrawlJob(ctx context.Context, job *models.CrawlJob) error {
	key := RedisKeyPrefixCrawlJob + job.ID
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal crawl job failed: %w", err)
	}

	err = r.client.Set(ctx, key, data, DefaultJobExpiration).Err()
	if err != nil {
		log.Error().Err(err).Str("jobID", job.ID).Msg("保存 CrawlJob 到 Redis 失败")
		return err
	}
	return nil
}

// GetCrawlJob 从 Redis 反序列化读取指定的 CrawlJob 实体
func (r *RedisService) GetCrawlJob(ctx context.Context, jobID string) (*models.CrawlJob, error) {
	key := RedisKeyPrefixCrawlJob + jobID
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // 任务不存在或已过期
	} else if err != nil {
		return nil, err
	}

	var job models.CrawlJob
	if err := json.Unmarshal([]byte(val), &job); err != nil {
		return nil, fmt.Errorf("unmarshal crawl job from redis failed: %w", err)
	}
	return &job, nil
}

// UpdateJobStatus 快速原子更新 Redis 中任务的状态
func (r *RedisService) UpdateJobStatus(ctx context.Context, jobID string, status models.CrawlJobStatus) error {
	job, err := r.GetCrawlJob(ctx, jobID)
	if err != nil || job == nil {
		return fmt.Errorf("job not found for status update: %s", jobID)
	}

	job.Status = status
	return r.SaveCrawlJob(ctx, job)
}

// AppendDocument 线程安全地追加新提取好的网页 Document 到指定 CrawlJob 并保存
func (r *RedisService) AppendDocument(ctx context.Context, jobID string, doc *models.Document) error {
	job, err := r.GetCrawlJob(ctx, jobID)
	if err != nil || job == nil {
		return fmt.Errorf("job not found for appending document: %s", jobID)
	}

	job.Documents = append(job.Documents, doc)
	// 如果抓取的文档数量达到了最大的 Limit，更新状态为 completed
	if len(job.Documents) >= job.Limit {
		job.Status = models.StatusCompleted
	}

	return r.SaveCrawlJob(ctx, job)
}

// slidingWindowLua 滑动窗口限流 Lua 脚本
const slidingWindowLua = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])

-- 1. 清除当前窗口之外的过期记录
local clearBefore = now - window
redis.call('ZREMRANGEBYSCORE', key, 0, clearBefore)

-- 2. 获取当前窗口内的请求数
local currentRequests = redis.call('ZCARD', key)

-- 3. 判断是否超过限额
if currentRequests < limit then
    redis.call('ZADD', key, now, now)
    redis.call('EXPIRE', key, math.ceil(window / 1000))
    return 1
else
    return 0
end
`

// AllowRateLimit 使用 Redis + Lua 脚本计算滑动窗口限流
func (r *RedisService) AllowRateLimit(ctx context.Context, domain string, limit int, windowDuration time.Duration) (bool, error) {
	if r == nil || r.client == nil {
		return true, nil // Redis 未连接时优雅降级允许通过
	}
	now := time.Now().UnixNano() / int64(time.Millisecond)
	windowMs := windowDuration.Milliseconds()
	key := fmt.Sprintf("firecrawl:rate_limit:%s", domain)

	res, err := r.client.Eval(ctx, slidingWindowLua, []string{key}, now, windowMs, limit).Result()
	if err != nil {
		return false, err
	}
	return res.(int64) == 1, nil
}

// AcquireLock 获取带 TTL 的 Redis 分布式锁 (使用 SETNX + 随机 UUID 防止误删)
func (r *RedisService) AcquireLock(ctx context.Context, lockKey string, lockValue string, ttl time.Duration) (bool, error) {
	if r == nil || r.client == nil {
		return true, nil // 降级处理
	}
	key := fmt.Sprintf("firecrawl:lock:%s", lockKey)
	return r.client.SetNX(ctx, key, lockValue, ttl).Result()
}

// releaseLockLua 释放分布式锁 Lua 脚本：防止误删其他 Goroutine/节点的锁
const releaseLockLua = `
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("del", KEYS[1])
else
    return 0
end
`

// ReleaseLock 校验 lockValue 并安全释放分布式锁
func (r *RedisService) ReleaseLock(ctx context.Context, lockKey string, lockValue string) (bool, error) {
	if r == nil || r.client == nil {
		return true, nil
	}
	key := fmt.Sprintf("firecrawl:lock:%s", lockKey)
	res, err := r.client.Eval(ctx, releaseLockLua, []string{key}, lockValue).Int64()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}

// renewLockLua 看门狗续期 Lua 脚本：当 lockValue 匹配时为锁延长 TTL
const renewLockLua = `
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("pexpire", KEYS[1], ARGV[2])
else
    return 0
end
`

// WatchdogLock 带有看门狗 (Watchdog) 自动续期功能的分布式锁高级封装
type WatchdogLock struct {
	redisSvc *RedisService
	lockKey  string
	lockVal  string
	ttl      time.Duration
	cancel   context.CancelFunc
}

// NewWatchdogLock 创建 WatchdogLock 实例
func (r *RedisService) NewWatchdogLock(lockKey string, lockVal string, ttl time.Duration) *WatchdogLock {
	return &WatchdogLock{
		redisSvc: r,
		lockKey:  lockKey,
		lockVal:  lockVal,
		ttl:      ttl,
	}
}

// Lock 获取分布式锁并启动看门狗协程自动续期
func (l *WatchdogLock) Lock(ctx context.Context) (bool, error) {
	ok, err := l.redisSvc.AcquireLock(ctx, l.lockKey, l.lockVal, l.ttl)
	if err != nil || !ok {
		return false, err
	}

	// 启动后台看门狗 (Watchdog) 自动续期协程
	watchCtx, cancel := context.WithCancel(context.Background())
	l.cancel = cancel
	go l.startWatchdog(watchCtx)

	return true, nil
}

// startWatchdog 后台看门狗协程：每隔 1/3 TTL 周期自动续期
func (l *WatchdogLock) startWatchdog(ctx context.Context) {
	ticker := time.NewTicker(l.ttl / 3)
	defer ticker.Stop()

	key := fmt.Sprintf("firecrawl:lock:%s", l.lockKey)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			res, err := l.redisSvc.client.Eval(ctx, renewLockLua, []string{key}, l.lockVal, l.ttl.Milliseconds()).Int64()
			if err != nil || res == 0 {
				return // 锁已被主动释放或失效，看门狗退出
			}
		}
	}
}

// Unlock 停止看门狗并安全释放锁
func (l *WatchdogLock) Unlock(ctx context.Context) (bool, error) {
	if l.cancel != nil {
		l.cancel() // 停止看门狗续期
	}
	return l.redisSvc.ReleaseLock(ctx, l.lockKey, l.lockVal)
}



