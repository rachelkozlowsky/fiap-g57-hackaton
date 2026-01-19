package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"notification-service/database"
	"notification-service/infra/clients"
	"notification-service/infra/email"
	"notification-service/infra/rabbitmq"
	"notification-service/infra/utils"
	"notification-service/service"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	godotenv.Load()

	db := database.InitDatabase()
	defer db.Close()

	rmq := rabbitmq.InitRabbitMQ()
	defer rmq.Close()

	smtpClient := email.InitSMTP()

	authServiceURL := utils.GetEnv("AUTH_SERVICE_URL", "http://auth-service:8081")
	videoServiceURL := utils.GetEnv("VIDEO_SERVICE_URL", "http://video-service:8082")
	
	authClient := clients.NewAuthServiceClient(authServiceURL)
	videoClient := clients.NewVideoServiceClient(videoServiceURL)
	
	log.Printf("Auth Service client initialized: %s", authServiceURL)
	log.Printf("Video Service client initialized: %s", videoServiceURL)

	workerCount := getEnvInt("WORKER_COUNT", 3)
	log.Printf("Notification Service starting with %d workers", workerCount)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			worker := service.NewNotificationWorker(workerID, db, rmq, smtpClient, authClient, videoClient)
			worker.Start(ctx)
		}(i)
	}

	go func() {
		log.Println("Metrics server starting on :8091")
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(":8091", nil); err != nil {
			log.Printf("Failed to start metrics server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down workers...")
	cancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All workers stopped gracefully")
	case <-time.After(30 * time.Second):
		log.Println("Timeout waiting for workers to stop")
	}
}

func getEnvInt(key string, defaultValue int) int {
	if value := utils.GetEnv(key, ""); value != "" {
		var intValue int
		fmt.Sscanf(value, "%d", &intValue)
		return intValue
	}
	return defaultValue
}
