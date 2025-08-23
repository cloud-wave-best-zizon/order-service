package main

import (
	"context"
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
	"crypto/tls"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	tlsConfig := &pkgtls.TLSConfig{}
	if err := envconfig.Process("", tlsConfig); err != nil {
		logger.Fatal("Failed to load TLS config", zap.Error(err))
	}

	logger.Info("Service configuration",
		zap.String("port", cfg.Port),
		zap.String("kafka_brokers", cfg.KafkaBrokers),
		zap.Bool("tls_enabled", tlsConfig.Enabled),
		zap.Bool("internal_tls", os.Getenv("INTERNAL_TLS_ENABLED") == "true"))

	// Initialize components
	dynamoClient, err := repository.NewDynamoDBClient(cfg)
	if err != nil {
		log.Fatal("Failed to create DynamoDB client:", err)
	}

	kafkaProducer, err := events.NewKafkaProducer(cfg.KafkaBrokers, logger)
	if err != nil {
		log.Fatal("Failed to create Kafka producer:", err)
	}
	defer kafkaProducer.Close()

	orderRepo := repository.NewOrderRepository(dynamoClient, cfg.OrderTableName)
	orderService := service.NewOrderService(orderRepo, kafkaProducer, logger)
	orderHandler := handler.NewOrderHandler(orderService, logger)

	// Setup Gin Router
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.Logger(logger))
	router.Use(middleware.RequestID())

	// Routes
	v1 := router.Group("/api/v1")
	{
		v1.POST("/orders", orderHandler.CreateOrder)
		v1.GET("/orders/:id", orderHandler.GetOrder)
		v1.GET("/health", func(c *gin.Context) {
			status := gin.H{
				"status":  "healthy",
				"service": "order-service",
				"port":    cfg.Port,
				"tls":     tlsConfig.Enabled,
				"internal_tls": os.Getenv("INTERNAL_TLS_ENABLED") == "true",
			}
			if err := kafkaProducer.HealthCheck(); err != nil {
				status["kafka"] = "unhealthy"
				c.JSON(503, status)
				return
			}
			status["kafka"] = "healthy"
			c.JSON(200, status)
		})
	}

	var wg sync.WaitGroup
	servers := []*http.Server{}

	// HTTP Server for ALB (port 8080)
	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}
	servers = append(servers, httpServer)

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("Starting HTTP server for ALB", zap.String("port", cfg.Port))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	// mTLS Server for service-to-service (port 8443)
	if os.Getenv("INTERNAL_TLS_ENABLED") == "true" {
		tlsCfg, err := pkgtls.LoadTLSConfig(tlsConfig, logger)
		if err != nil {
			logger.Error("Failed to load TLS config", zap.Error(err))
		} else {
			httpsServer := &http.Server{
				Addr:      ":8443",
				Handler:   router,
				TLSConfig: tlsCfg,
			}
			servers = append(servers, httpsServer)

			wg.Add(1)
			go func() {
				defer wg.Done()
				logger.Info("Starting mTLS server for internal communication", zap.String("port", "8443"))
				if err := httpsServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
					logger.Error("mTLS server failed", zap.Error(err))
				}
			}()

			// Watch for certificate updates
			go pkgtls.WatchCertificates(tlsConfig, func(newCfg *tls.Config) error {
				httpsServer.TLSConfig = newCfg
				logger.Info("TLS configuration reloaded")
				return nil
			}, logger)
		}
	}

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down servers...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, srv := range servers {
		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("Server shutdown failed", zap.Error(err))
		}
	}
	
	wg.Wait()
	logger.Info("All servers stopped")
}
