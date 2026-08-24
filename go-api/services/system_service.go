package services

import (
	"context"
	"fmt"
	"time"

	"github.com/firecrawl/go-api/models"
	"github.com/redis/go-redis/v9"
)

// SystemService 负责团队并发、积分额度与系统状态管理
type SystemService struct {
	redisService *RedisService
}

// NewSystemService 初始化 SystemService
func NewSystemService(redisService *RedisService) *SystemService {
	return &SystemService{
		redisService: redisService,
	}
}

// GetConcurrencyCheck 获取团队当前并发度
func (s *SystemService) GetConcurrencyCheck(ctx context.Context, teamID string) *models.ConcurrencyCheckResponse {
	maxConcurrency := 10
	currentConcurrency := 0

	if s.redisService != nil && s.redisService.client != nil {
		key := "concurrency-limiter:" + teamID
		now := time.Now().UnixMilli()
		val, err := s.redisService.client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
			Min: fmt.Sprintf("%d", now),
			Max: "+inf",
		}).Result()
		if err == nil {
			currentConcurrency = len(val)
		}
	}

	return &models.ConcurrencyCheckResponse{
		Success:        true,
		Concurrency:    currentConcurrency,
		MaxConcurrency: maxConcurrency,
	}
}

// GetCreditUsage 获取团队 Credit 余额与账期
func (s *SystemService) GetCreditUsage(ctx context.Context, teamID string) *models.CreditUsageResponse {
	start := time.Now().AddDate(0, 0, -time.Now().Day()+1).Format("2006-01-02T15:04:05Z07:00")
	end := time.Now().AddDate(0, 1, -time.Now().Day()).Format("2006-01-02T15:04:05Z07:00")

	return &models.CreditUsageResponse{
		Success: true,
		Data: &models.CreditUsageData{
			RemainingCredits:   500000,
			PlanCredits:        500000,
			BillingPeriodStart: &start,
			BillingPeriodEnd:   &end,
		},
	}
}

// GetTokenUsage 获取团队 Token 余额与用量
func (s *SystemService) GetTokenUsage(ctx context.Context, teamID string) *models.TokenUsageResponse {
	start := time.Now().AddDate(0, 0, -time.Now().Day()+1).Format("2006-01-02T15:04:05Z07:00")
	end := time.Now().AddDate(0, 1, -time.Now().Day()).Format("2006-01-02T15:04:05Z07:00")

	return &models.TokenUsageResponse{
		Success: true,
		Data: &models.TokenUsageData{
			RemainingTokens:    7500000,
			PlanTokens:         7500000,
			BillingPeriodStart: &start,
			BillingPeriodEnd:   &end,
		},
	}
}

// GetQueueStatus 获取任务队列积压与并发状态
func (s *SystemService) GetQueueStatus(ctx context.Context, teamID string) *models.QueueStatusResponse {
	nowStr := time.Now().Format(time.RFC3339)
	return &models.QueueStatusResponse{
		Success:            true,
		JobsInQueue:        0,
		ActiveJobsInQueue:  0,
		WaitingJobsInQueue: 0,
		MaxConcurrency:     10,
		MostRecentSuccess:  &nowStr,
	}
}
