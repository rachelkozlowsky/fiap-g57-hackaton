package service

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
	"processing-service/domain"
	"processing-service/infra/utils"
	"github.com/google/uuid"
)

var execCommand = exec.CommandContext

type Worker struct {
	ID          int
	db          domain.DatabaseInterface
	minio       domain.MinIOInterface
	rabbitmq    domain.RabbitMQInterface
	videoClient domain.VideoServiceClient
}

func NewWorker(id int, db domain.DatabaseInterface, minio domain.MinIOInterface, rabbitmq domain.RabbitMQInterface, videoClient domain.VideoServiceClient) *Worker {
	return &Worker{
		ID:          id,
		db:          db,
		minio:       minio,
		rabbitmq:    rabbitmq,
		videoClient: videoClient,
	}
}

func (w *Worker) Start(ctx context.Context) {
	log.Printf("Worker %d started", w.ID)

	msgs, err := w.rabbitmq.SubscribeVideoUpload()
	if err != nil {
		log.Fatalf("Worker %d: Failed to subscribe to queue: %v", w.ID, err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d stopping...", w.ID)
			return
		case msg, ok := <-msgs:
			if !ok {
				log.Printf("Worker %d: Channel closed", w.ID)
				return
			}

			var message domain.VideoProcessingMessage
			if err := json.Unmarshal(msg.Body, &message); err != nil {
				log.Printf("Worker %d: Error unmarshaling message: %v", w.ID, err)
				msg.Nack(false, false)
				continue
			}

			log.Printf("Worker %d: Processing video %s", w.ID, message.VideoID)
			err := w.processVideo(ctx, &message)

			if err != nil {
				log.Printf("Worker %d: Error processing video %s: %v", w.ID, message.VideoID, err)
				msg.Nack(false, true)
			} else {
				log.Printf("Worker %d: Successfully processed video %s", w.ID, message.VideoID)
				msg.Ack(false)
			}
		}
	}
}

func (w *Worker) processVideo(ctx context.Context, message *domain.VideoProcessingMessage) error {
	video, err := w.videoClient.GetVideoByID(message.VideoID)
	if err != nil {
		return fmt.Errorf("failed to get video from Video Service: %w", err)
	}

	if err := w.videoClient.UpdateVideoStatus(message.VideoID, "processing", ""); err != nil {
		log.Printf("Warning: Failed to update video status to processing: %v", err)
	}

	job := &domain.ProcessingJob{
		ID:        generateID(),
		VideoID:   message.VideoID,
		UserID:    message.UserID,
		WorkerID:  fmt.Sprintf("worker-%d", w.ID),
		Status:    "running",
		StartedAt: timePtr(time.Now()),
		CreatedAt: time.Now(),
	}
	w.db.CreateProcessingJob(job)

	tempDir := filepath.Join("temp", message.VideoID)
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	videoPath := filepath.Join(tempDir, message.Filename)
	if err := w.minio.DownloadFile(message.StoragePath, videoPath); err != nil {
		w.updateJobFailed(job, err)
		w.updateVideoFailed(video, err)
		return fmt.Errorf("failed to download video: %w", err)
	}

	framesDir := filepath.Join(tempDir, "frames")
	os.MkdirAll(framesDir, 0755)

	framePattern := filepath.Join(framesDir, "frame_%04d.png")
	fps := utils.GetEnv("FFMPEG_FPS", "1")

	cmd := execCommand(ctx, "ffmpeg",
		"-i", videoPath,
		"-vf", fmt.Sprintf("fps=%s", fps),
		"-y",
		framePattern,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		w.updateJobFailed(job, fmt.Errorf("ffmpeg error: %w, output: %s", err, string(output)))
		w.updateVideoFailed(video, fmt.Errorf("failed to extract frames"))
		return fmt.Errorf("ffmpeg failed: %w", err)
	}

	frames, err := filepath.Glob(filepath.Join(framesDir, "*.png"))
	if err != nil || len(frames) == 0 {
		w.updateJobFailed(job, fmt.Errorf("no frames extracted"))
		w.updateVideoFailed(video, fmt.Errorf("no frames extracted"))
		return fmt.Errorf("no frames extracted")
	}

	log.Printf("Worker %d: Extracted %d frames from video %s", w.ID, len(frames), message.VideoID)

	zipFilename := fmt.Sprintf("frames_%s_%s.zip", message.VideoID, time.Now().Format("20060102_150405"))
	zipPath := filepath.Join(tempDir, zipFilename)

	if err := w.createZipFile(frames, zipPath); err != nil {
		w.updateJobFailed(job, err)
		w.updateVideoFailed(video, fmt.Errorf("failed to create zip"))
		return fmt.Errorf("failed to create zip: %w", err)
	}

	zipFile, err := os.Open(zipPath)
	if err != nil {
		w.updateJobFailed(job, err)
		w.updateVideoFailed(video, fmt.Errorf("failed to open zip"))
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer zipFile.Close()

	zipInfo, _ := zipFile.Stat()
	zipStoragePath, err := w.minio.UploadProcessedFile(zipFile, zipFilename, zipInfo.Size())
	if err != nil {
		w.updateJobFailed(job, err)
		w.updateVideoFailed(video, fmt.Errorf("failed to upload zip"))
		return fmt.Errorf("failed to upload zip: %w", err)
	}

	frameCount := len(frames)
	if err := w.videoClient.CompleteVideo(message.VideoID, zipStoragePath, zipInfo.Size(), frameCount); err != nil {
		log.Printf("Warning: Failed to mark video as completed via HTTP: %v", err)
	}

	job.Status = "completed"
	job.CompletedAt = timePtr(time.Now())
	duration := int(time.Since(*job.StartedAt).Seconds())
	job.DurationSeconds = &duration
	w.db.UpdateProcessingJob(job)

	w.rabbitmq.PublishNotification(domain.NotificationMessage{
		UserID:  message.UserID,
		VideoID: message.VideoID,
		Type:    "video_completed",
		Subject: "Video Processing Completed",
		Message: fmt.Sprintf("Your video has been processed successfully. %d frames extracted.", frameCount),
	})

	log.Printf("Worker %d: Video %s processed successfully (%d frames, %.2fMB zip)",
		w.ID, message.VideoID, frameCount, float64(zipInfo.Size())/1024/1024)

	return nil
}

func (w *Worker) createZipFile(files []string, zipPath string) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, file := range files {
		if err := w.addFileToZip(zipWriter, file); err != nil {
			return err
		}
	}

	return nil
}

func (w *Worker) addFileToZip(zipWriter *zip.Writer, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	header.Name = filepath.Base(filename)
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}

func (w *Worker) updateJobFailed(job *domain.ProcessingJob, err error) {
	job.Status = "failed"
	job.CompletedAt = timePtr(time.Now())
	job.ErrorMessage = stringPtr(err.Error())
	w.db.UpdateProcessingJob(job)
}

func (w *Worker) updateVideoFailed(video *domain.Video, err error) {
	if httpErr := w.videoClient.FailVideo(video.ID, err.Error()); httpErr != nil {
		log.Printf("Warning: Failed to mark video as failed via HTTP: %v", httpErr)
	}

	w.rabbitmq.PublishNotification(domain.NotificationMessage{
		UserID:  video.UserID,
		VideoID: video.ID,
		Type:    "video_failed",
		Subject: "Video Processing Failed",
		Message: fmt.Sprintf("Failed to process your video: %s", err.Error()),
	})
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func stringPtr(s string) *string {
	return &s
}

func generateID() string {
	return uuid.New().String()
}
