package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"status-service/domain"
	"status-service/infra/utils"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

func InitRedis() *RedisClient {
	redisURL := utils.GetEnv("REDIS_URL", "redis://localhost:6379")
	
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}

	client := redis.NewClient(opt)
	ctx := context.Background()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("Connected to Redis")

	return &RedisClient{
		client: client,
		ctx:    ctx,
	}
}

func (r *RedisClient) Ping() error {
	return r.client.Ping(r.ctx).Err()
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}

func (r *RedisClient) Set(key string, value interface{}, ttl time.Duration) error {
	return r.client.Set(r.ctx, key, value, ttl).Err()
}

func (r *RedisClient) Get(key string) (string, error) {
	val, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("key not found")
	}
	return val, err
}

func (r *RedisClient) Delete(key string) error {
	return r.client.Del(r.ctx, key).Err()
}

func (r *RedisClient) Exists(key string) (bool, error) {
	count, err := r.client.Exists(r.ctx, key).Result()
	return count > 0, err
}

func (r *RedisClient) SetUser(userID string, user *domain.User, ttl time.Duration) error {
	key := fmt.Sprintf("user:%s", userID)
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return r.Set(key, string(data), ttl)
}

func (r *RedisClient) GetUser(userID string) (*domain.User, error) {
	key := fmt.Sprintf("user:%s", userID)
	data, err := r.Get(key)
	if err != nil {
		return nil, err
	}
	
	var user domain.User
	if err := json.Unmarshal([]byte(data), &user); err != nil {
		return nil, err
	}
	
	return &user, nil
}

func (r *RedisClient) DeleteUser(userID string) error {
	key := fmt.Sprintf("user:%s", userID)
	return r.Delete(key)
}

func (r *RedisClient) SetSession(sessionID string, session *domain.Session, ttl time.Duration) error {
	key := fmt.Sprintf("session:%s", sessionID)
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return r.Set(key, string(data), ttl)
}

func (r *RedisClient) GetSession(sessionID string) (*domain.Session, error) {
	key := fmt.Sprintf("session:%s", sessionID)
	data, err := r.Get(key)
	if err != nil {
		return nil, err
	}
	
	var session domain.Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, err
	}
	
	return &session, nil
}

func (r *RedisClient) DeleteSession(sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return r.Delete(key)
}

func (r *RedisClient) IncrementRateLimit(key string, window time.Duration) (int64, error) {
	pipe := r.client.Pipeline()
	
	incr := pipe.Incr(r.ctx, key)
	pipe.Expire(r.ctx, key, window)
	
	_, err := pipe.Exec(r.ctx)
	if err != nil {
		return 0, err
	}
	
	return incr.Val(), nil
}

func (r *RedisClient) CheckRateLimit(key string, limit int64, window time.Duration) (bool, error) {
	count, err := r.IncrementRateLimit(key, window)
	if err != nil {
		return false, err
	}
	
	return count <= limit, nil
}

func (r *RedisClient) InvalidatePattern(pattern string) error {
	iter := r.client.Scan(r.ctx, 0, pattern, 0).Iterator()
	
	keys := []string{}
	for iter.Next(r.ctx) {
		keys = append(keys, iter.Val())
	}
	
	if err := iter.Err(); err != nil {
		return err
	}
	
	if len(keys) > 0 {
		return r.client.Del(r.ctx, keys...).Err()
	}
	
	return nil
}

func (r *RedisClient) Publish(channel string, message string) error {
	return r.client.Publish(r.ctx, channel, message).Err()
}

func (r *RedisClient) Subscribe(channel string) *redis.PubSub {
	return r.client.Subscribe(r.ctx, channel)
}
