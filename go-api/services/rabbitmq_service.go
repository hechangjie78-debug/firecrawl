package services

import (
	"context"
	"fmt"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

const (
	// CrawlQueueName RabbitMQ 异步多页爬虫队列名称
	CrawlQueueName = "firecrawl_crawl_jobs"
)

// RabbitMQService 封装 RabbitMQ AMQP 队列发布与消费逻辑
type RabbitMQService struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewRabbitMQService 初始化 RabbitMQ 建立通道并声明高可用 Task 队列
func NewRabbitMQService() (*RabbitMQService, error) {
	amqpURL := os.Getenv("NUQ_RABBITMQ_URL")
	if amqpURL == "" {
		amqpURL = os.Getenv("RABBITMQ_URL")
	}
	if amqpURL == "" {
		amqpURL = "amqp://guest:guest@localhost:5672/"
	}

	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		log.Error().Err(err).Str("url", amqpURL).Msg("建立 RabbitMQ AMQP 连接失败")
		return nil, fmt.Errorf("rabbitmq dial failed: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("rabbitmq create channel failed: %w", err)
	}

	// 声明持久化队列 (Durable = true)
	_, err = ch.QueueDeclare(
		CrawlQueueName, // 队列名称
		true,           // durable (持久化保存磁盘)
		false,          // delete when unused
		false,          // exclusive
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("declare rabbitmq queue failed: %w", err)
	}

	log.Info().Str("url", amqpURL).Str("queue", CrawlQueueName).Msg("RabbitMQ 异步队列初始化建立成功！")
	return &RabbitMQService{
		conn:    conn,
		channel: ch,
	}, nil
}

// PublishCrawlJob 快速将 Crawl Job ID 压入 RabbitMQ 队列 (1 毫秒削峰返回)
func (r *RabbitMQService) PublishCrawlJob(ctx context.Context, jobID string) error {
	err := r.channel.PublishWithContext(
		ctx,
		"",             // exchange
		CrawlQueueName, // routing key / queue name
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent, // 消息持久化
			ContentType:  "text/plain",
			Body:         []byte(jobID),
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		log.Error().Err(err).Str("jobID", jobID).Msg("发布 Crawl Job 消息至 RabbitMQ 失败")
		return err
	}
	log.Debug().Str("jobID", jobID).Msg("已成功将 Crawl 任务压入 RabbitMQ 削峰队列")
	return nil
}

// StartConsumer 在独立的 Goroutine 消费监听队列，拉取任务进行处理
func (r *RabbitMQService) StartConsumer(handler func(jobID string)) error {
	msgs, err := r.channel.Consume(
		CrawlQueueName, // 队列名
		"",             // consumer name
		false,          // auto-ack (设置为 false，手动 Ack 防丢失)
		false,          // exclusive
		false,          // no-local
		false,          // no-wait
		nil,            // args
	)
	if err != nil {
		return fmt.Errorf("start consumer on rabbitmq queue failed: %w", err)
	}

	go func() {
		log.Info().Str("queue", CrawlQueueName).Msg("RabbitMQ 异步抓取消费者协程启动就绪，等待消费 Task...")
		for d := range msgs {
			jobID := string(d.Body)
			log.Info().Str("jobID", jobID).Msg("RabbitMQ 收到异步爬虫任务通知，开始消费处理...")

			// 回调处理爬虫业务
			handler(jobID)

			// 手动确认 Ack 消息处理完成
			_ = d.Ack(false)
		}
	}()

	return nil
}

// Close 安全关闭连接与通道
func (r *RabbitMQService) Close() {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
}
