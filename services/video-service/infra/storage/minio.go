package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"
	"video-service/infra/utils"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOClient struct {
	client          *minio.Client
	bucketRaw       string
	bucketProcessed string
}

func InitMinIO() *MinIOClient {
	endpoint := utils.GetEnv("MINIO_ENDPOINT", "localhost:9000")
	accessKey := utils.GetEnv("MINIO_ACCESS_KEY", "g57")
	secretKey := utils.GetEnv("MINIO_SECRET_KEY", "g57123456")
	useSSL := utils.GetEnv("MINIO_USE_SSL", "false") == "true"
	bucketRaw := utils.GetEnv("MINIO_BUCKET_RAW", "videos-raw")
	bucketProcessed := utils.GetEnv("MINIO_BUCKET_PROCESSED", "videos-processed")

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalf("Failed to create MinIO client: %v", err)
	}

	ctx := context.Background()

	buckets := []string{bucketRaw, bucketProcessed}
	for _, bucket := range buckets {
		exists, err := client.BucketExists(ctx, bucket)
		if err != nil {
			log.Fatalf("Failed to check bucket %s: %v", bucket, err)
		}
		if !exists {
			err = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
			if err != nil {
				log.Fatalf("Failed to create bucket %s: %v", bucket, err)
			}
			log.Printf("Created bucket: %s", bucket)
		}
	}

	log.Println("✅ Connected to MinIO")

	return &MinIOClient{
		client:          client,
		bucketRaw:       bucketRaw,
		bucketProcessed: bucketProcessed,
	}
}

func (m *MinIOClient) Ping() error {
	ctx := context.Background()
	_, err := m.client.BucketExists(ctx, m.bucketRaw)
	return err
}

func (m *MinIOClient) UploadFile(reader io.Reader, filename string, size int64) (string, error) {
	ctx := context.Background()

	objectName := fmt.Sprintf("%s/%s", time.Now().Format("2006/01/02"), filename)

	_, err := m.client.PutObject(ctx, m.bucketRaw, objectName, reader, size, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return "", err
	}

	return objectName, nil
}

func (m *MinIOClient) UploadProcessedFile(reader io.Reader, filename string, size int64) (string, error) {
	ctx := context.Background()

	objectName := fmt.Sprintf("%s/%s", time.Now().Format("2006/01/02"), filename)

	_, err := m.client.PutObject(ctx, m.bucketProcessed, objectName, reader, size, minio.PutObjectOptions{
		ContentType: "application/zip",
	})
	if err != nil {
		return "", err
	}

	return objectName, nil
}

func (m *MinIOClient) DownloadFile(objectName, destPath string) error {
	ctx := context.Background()

	err := m.client.FGetObject(ctx, m.bucketRaw, objectName, destPath, minio.GetObjectOptions{})
	return err
}

func (m *MinIOClient) GetFileStream(objectName string) (*minio.Object, error) {
	ctx := context.Background()

	obj, err := m.client.GetObject(ctx, m.bucketProcessed, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	_, err = obj.Stat()
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (m *MinIOClient) GetPresignedURL(objectName string, expiry time.Duration) (string, error) {
	ctx := context.Background()

	url, err := m.client.PresignedGetObject(ctx, m.bucketProcessed, objectName, expiry, nil)
	if err != nil {
		return "", err
	}

	urlString := url.String()
	return strings.Replace(urlString, "minio:9000", "localhost:9000", 1), nil
}

func (m *MinIOClient) DeleteFile(objectName string) error {
	ctx := context.Background()

	err := m.client.RemoveObject(ctx, m.bucketRaw, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		err = m.client.RemoveObject(ctx, m.bucketProcessed, objectName, minio.RemoveObjectOptions{})
	}

	return err
}

func (m *MinIOClient) ListFiles(prefix string) ([]string, error) {
	ctx := context.Background()

	objectCh := m.client.ListObjects(ctx, m.bucketRaw, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	files := []string{}
	for object := range objectCh {
		if object.Err != nil {
			return nil, object.Err
		}
		files = append(files, object.Key)
	}

	return files, nil
}

func (m *MinIOClient) GetFileInfo(objectName string) (*minio.ObjectInfo, error) {
	ctx := context.Background()

	info, err := m.client.StatObject(ctx, m.bucketRaw, objectName, minio.StatObjectOptions{})
	if err != nil {
		info, err = m.client.StatObject(ctx, m.bucketProcessed, objectName, minio.StatObjectOptions{})
	}

	return &info, err
}
