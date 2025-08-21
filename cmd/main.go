package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/cloud-wave-best-zizon/order-service/internal/events"
	"github.com/cloud-wave-best-zizon/order-service/internal/handler"
	"github.com/cloud-wave-best-zizon/order-service/internal/repository"
	"github.com/cloud-wave-best-zizon/order-service/internal/service"
	"github.com/cloud-wave-best-zizon/order-service/pkg/config"
	"github.com/cloud-wave-best-zizon/order-service/pkg/middleware"
	pkgtls "github.com/cloud-wave-best-zizon/order-service/pkg/tls"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
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

	// TLS 설정 로드 (여기로 이동!)
	tlsConfig := &pkgtls.TLSConfig{}
	if err := envconfig.Process("", tlsConfig); err != nil {
		logger.Fatal("Failed to load TLS config", zap.Error(err))
	}

	logger.Info("Service configuration",
		zap.String("port", cfg.Port),
		zap.String("kafka_brokers", cfg.KafkaBrokers),
		zap.String("dynamodb_endpoint", cfg.DynamoDBEndpoint),
		zap.Bool("tls_enabled", tlsConfig.Enabled))

	// DynamoDB 클라이언트 초기화
	dynamoClient, err := repository.NewDynamoDBClient(cfg)
	if err != nil {
		log.Fatal("Failed to create DynamoDB client:", err)
	}

	// kafka producer 생성
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

		v1.GET("/orders/test/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"service":   "order-service",
				"status":    "ok",
				"timestamp": time.Now().Format(time.RFC3339),
			})
		})

		v1.GET("/health", func(c *gin.Context) {
			status := gin.H{
				"status":    "healthy",
				"service":   "order-service",
				"port":      cfg.Port,
				"tls":       tlsConfig.Enabled,
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

	// Server 설정 (한 번만!)
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// TLS 설정 적용
	var tlsConfigMutex sync.RWMutex
	if tlsConfig.Enabled {
		tlsCfg, err := pkgtls.LoadTLSConfig(tlsConfig, logger)
		if err != nil {
			logger.Fatal("Failed to load TLS configuration", zap.Error(err))
		}
		srv.TLSConfig = tlsCfg

		// 인증서 자동 리로드
		go pkgtls.WatchCertificates(tlsConfig, func(newCfg *tls.Config) error {
			tlsConfigMutex.Lock()
			defer tlsConfigMutex.Unlock()
			srv.TLSConfig = newCfg
			return nil
		}, logger)
	}

	// Server 시작 
	go func() {
		logger.Info("Starting server",
			zap.String("port", cfg.Port),
			zap.Bool("tls_enabled", tlsConfig.Enabled))

		var err error
		if tlsConfig.Enabled {
			// TLS 활성화 시
			err = srv.ListenAndServeTLS("", "") // 인증서는 TLSConfig에서 로드
		} else {
			// 일반 HTTP
			err = srv.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
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