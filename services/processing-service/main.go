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
	"processing-service/database"
	"processing-service/infra/broker"
	"processing-service/infra/clients"
	"processing-service/infra/storage"
	"processing-service/service"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	godotenv.Load()

	db := database.InitDatabase()
	defer db.Close()

	minio := storage.InitMinIO()
	rabbitmq := broker.InitRabbitMQ()
	defer rabbitmq.Close()

	videoServiceURL := os.Getenv("VIDEO_SERVICE_URL")
	if videoServiceURL == "" {
		videoServiceURL = "http://video-service:8082"
	}
	videoClient := clients.NewVideoServiceClient(videoServiceURL)
	log.Printf("Video Service client initialized: %s", videoServiceURL)

	workerCount := getEnvInt("WORKER_COUNT", 5)
	log.Printf("Processing Service starting with %d workers", workerCount)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			worker := service.NewWorker(workerID, db, minio, rabbitmq, videoClient)
			worker.Start(ctx)
		}(i)
	}

	go func() {
		log.Println("Metrics server starting on :8090")
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(":8090", nil); err != nil {
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
	if value := os.Getenv(key); value != "" {
		var intValue int
		fmt.Sscanf(value, "%d", &intValue)
		return intValue
	}
	return defaultValue
}
