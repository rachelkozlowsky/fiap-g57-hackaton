package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"auth-service/database"
	"auth-service/infra/handlers"
	"auth-service/infra/metrics"
	"auth-service/infra/utils"
	"auth-service/service"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	db := database.InitDatabase()
	defer db.Close()

	redis := database.InitRedis()
	defer redis.Close()

	router := setupRouter(db, redis)

	srv := &http.Server{
		Addr:         ":" + utils.GetEnv("PORT", "8081"),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Auth Service starting on port %s", utils.GetEnv("PORT", "8081"))
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

func setupRouter(db *database.Database, redis *database.RedisClient) *gin.Engine {
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
	router.GET("/health/ready", readinessProbe(db, redis))

	router.GET("/metrics", metrics.MetricsHandler)

	api := router.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			authService := service.NewAuthService(db, redis)
			authHandler := handlers.NewAuthHandler(authService)
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/logout", handlers.AuthMiddleware(), authHandler.Logout)
			auth.GET("/me", handlers.AuthMiddleware(), authHandler.GetCurrentUser)
		}

		users := api.Group("/users")
		users.Use(handlers.AuthMiddleware())
		{
			userHandler := handlers.NewUserHandler(db)
			users.GET("", handlers.AdminMiddleware(), userHandler.ListUsers)
			users.GET("/:id", userHandler.GetUser)
			users.PUT("/:id", userHandler.UpdateUser)
			users.DELETE("/:id", handlers.AdminMiddleware(), userHandler.DeleteUser)
		}
	}

	return router
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "auth-service",
		"version": "1.0.0",
		"time":    time.Now().Unix(),
	})
}

func livenessProbe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive"})
}

func readinessProbe(db *database.Database, redis *database.RedisClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := db.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"error":  "database connection failed",
			})
			return
		}

		if err := redis.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"error":  "redis connection failed",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}
