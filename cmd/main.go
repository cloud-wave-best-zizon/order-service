package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloud-wave-best-zizon/order-service/internal/events"
	"github.com/cloud-wave-best-zizon/order-service/internal/handler"
	"github.com/cloud-wave-best-zizon/order-service/internal/repository"
	"github.com/cloud-wave-best-zizon/order-service/internal/service"
	"github.com/cloud-wave-best-zizon/order-service/pkg/config"
	"github.com/cloud-wave-best-zizon/order-service/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}
	// Logger 초기화
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Config 로드
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	logger.Info("Service configuration",
		zap.String("port", cfg.Port),
		zap.String("kafka_brokers", cfg.KafkaBrokers),
		zap.String("dynamodb_endpoint", cfg.DynamoDBEndpoint))

	// DynamoDB 클라이언트 초기화
	dynamoClient, err := repository.NewDynamoDBClient(cfg)
	if err != nil {
		log.Fatal("Failed to create DynamoDB client:", err)
	}

	// kafka producer 생성 (logger 추가)
	kafkaProducer, err := events.NewKafkaProducer(cfg.KafkaBrokers, logger)
	if err != nil {
		log.Fatal("Failed to create Kafka producer:", err)
	}
	defer kafkaProducer.Close()

	// Repository, Service, Handler 초기화
	orderRepo := repository.NewOrderRepository(dynamoClient, cfg.OrderTableName)
	orderService := service.NewOrderService(orderRepo, kafkaProducer, logger)
	orderHandler := handler.NewOrderHandler(orderService, logger)

	// Gin Router 설정
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.Logger(logger))
	router.Use(middleware.RequestID())

	// Routes
	v1 := router.Group("/api/v1")
	{
		v1.POST("/orders", orderHandler.CreateOrder)
		v1.GET("/orders/:id", orderHandler.GetOrder)

		// --- 여기부터 추가 ---
		// 간단한 응답 테스트를 위한 PING API
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"service":   "order-service",
				"status":    "ok",
				"timestamp": time.Now().Format(time.RFC3339),
			})
		})
		// --- 여기까지 추가 ---

		v1.GET("/health", func(c *gin.Context) {
			status := gin.H{
				"status":  "healthy",
				"service": "order-service",
				"port":    cfg.Port,
			}

			// Kafka 상태 확인
			if err := kafkaProducer.HealthCheck(); err != nil {
				status["kafka"] = "unhealthy"
				status["kafka_error"] = err.Error()
				c.JSON(503, status)
				return
			}
			status["kafka"] = "healthy"

			c.JSON(200, status)
		})
	}

	// Server 시작
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		logger.Info("Starting server", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}
	logger.Info("Server exited")
}
