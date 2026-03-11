package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"video-service/database"
	"video-service/domain"
	"video-service/infra/broker"
	"video-service/infra/clients"
	"video-service/infra/handlers"
	"video-service/infra/metrics"
	"video-service/infra/storage"
	"video-service/infra/utils"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	db := database.InitDatabase()
	defer db.Close()

	minio := storage.InitMinIO()
	rabbitmq := broker.InitRabbitMQ()
	defer rabbitmq.Close()

	authServiceURL := utils.GetEnv("AUTH_SERVICE_URL", "http://auth-service:8081")
	authClient := clients.NewAuthServiceClient(authServiceURL)

	router := setupRouter(db, minio, rabbitmq, authClient)

	port := utils.GetEnv("PORT", "8082")
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("✅ Video Service starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}

func setupRouter(db *database.Database, minio *storage.MinIOClient, rabbitmq *broker.RabbitMQClient, authClient domain.AuthServiceClient) *gin.Engine {
	if utils.GetEnv("GIN_MODE", "debug") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(metrics.PrometheusMiddleware())
	router.Use(handlers.CorsMiddleware())

	router.GET("/health", healthCheck)
	router.GET("/health/live", livenessProbe)
	router.GET("/health/ready", readinessProbe(db, minio, rabbitmq))
	router.GET("/metrics", metrics.MetricsHandler)

	api := router.Group("/api/v1")
	{
		videos := api.Group("/videos")
		videos.Use(handlers.AuthMiddleware())
		{
			videoHandler := handlers.NewVideoHandler(db, minio, rabbitmq, authClient)
			videos.POST("/upload", videoHandler.Upload)
			videos.GET("/", videoHandler.List)
			videos.GET("/:id", videoHandler.GetVideo)
			videos.DELETE("/:id", videoHandler.DeleteVideo)
		}

		videosPublic := api.Group("/videos")
		{
			videoHandler := handlers.NewVideoHandler(db, minio, rabbitmq, authClient)
			videosPublic.GET("/:id/download", videoHandler.DownloadZip)
		}
	}

	internal := router.Group("/api/internal")
	{
		internalHandler := handlers.NewInternalHandler(db)
		internal.GET("/videos/:id", internalHandler.GetVideoByID)
		internal.GET("/videos", internalHandler.ListUserVideos)
		internal.PATCH("/videos/:id/status", internalHandler.UpdateVideoStatus)
		internal.POST("/videos/:id/complete", internalHandler.CompleteVideo)
		internal.POST("/videos/:id/fail", internalHandler.FailVideo)
		internal.GET("/stats/user/:user_id", internalHandler.GetUserStats)
		internal.GET("/stats/system", internalHandler.GetSystemStats)
	}

	return router
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "video-service",
		"version": "1.0.0",
		"time":    time.Now().Unix(),
	})
}

func livenessProbe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive"})
}

func readinessProbe(db *database.Database, minio *storage.MinIOClient, rabbitmq *broker.RabbitMQClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := db.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": "database connection failed"})
			return
		}
		if err := minio.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": "minio connection failed"})
			return
		}
		if err := rabbitmq.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": "rabbitmq connection failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}
