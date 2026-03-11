package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"status-service/database"
	"status-service/domain"
	"status-service/infra/cache"
	"status-service/infra/clients"
	"status-service/infra/handlers"
	"status-service/infra/storage"
	"status-service/infra/utils"
	"status-service/service"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	db := database.InitDatabase()
	defer db.Close()

	redis := cache.InitRedis()
	defer redis.Close()

	minio := storage.InitMinIO()

	videoClient := clients.NewVideoServiceClient(utils.GetEnv("VIDEO_SERVICE_URL", "http://video-service:8082"))

	statusService := service.NewStatusService(db, redis, minio, videoClient)
	statusHandler := handlers.NewStatusHandler(statusService)

	router := setupRouter(db, redis, statusHandler)

	srv := &http.Server{
		Addr:         ":" + utils.GetEnv("PORT", "8083"),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Status Service starting on port %s", utils.GetEnv("PORT", "8083"))
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

func setupRouter(db domain.DatabaseInterface, redis domain.RedisInterface, statusHandler *handlers.StatusHandler) *gin.Engine {
	if utils.GetEnv("GIN_MODE", "debug") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(handlers.CorsMiddleware())

	router.GET("/health", healthCheck)
	router.GET("/health/live", livenessProbe)
	router.GET("/health/ready", readinessProbe(db, redis))
	router.GET("/metrics", metricsHandler)

	api := router.Group("/api/v1")
	{
		videos := api.Group("/videos")
		videos.Use(handlers.AuthMiddleware())
		{
			videos.GET("/status", statusHandler.ListVideos)
			videos.GET("/:id", statusHandler.GetVideo)
			videos.GET("/:id/download", statusHandler.DownloadZip)
		}

		stats := api.Group("/stats")
		stats.Use(handlers.AuthMiddleware())
		{
			stats.GET("/user", statusHandler.GetUserStats)
			stats.GET("/system", statusHandler.GetSystemStats)
		}
	}

	return router
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "status-service",
		"version": "1.0.0",
		"time":    time.Now().Unix(),
	})
}

func livenessProbe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive"})
}

func readinessProbe(db domain.DatabaseInterface, redis domain.RedisInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := db.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": "database"})
			return
		}
		if err := redis.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": "redis"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}

func metricsHandler(c *gin.Context) {
	c.String(http.StatusOK, "# Status Service Metrics\n")
}
