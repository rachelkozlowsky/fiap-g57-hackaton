package broker

import (
	"encoding/json"
	"fmt"
	"log"
	"processing-service/domain"
	"processing-service/infra/utils"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQClient struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	url     string
}

func InitRabbitMQ() *RabbitMQClient {
	url := utils.GetEnv("RABBITMQ_URL", "amqp://g57:g57123456@localhost:5672/")
	client := &RabbitMQClient{url: url}
	client.connect()
	return client
}

func (r *RabbitMQClient) connect() {
	var err error
	r.conn, err = amqp.Dial(r.url)
	if err != nil {
		log.Printf("Failed to connect to RabbitMQ: %v", err)
		return
	}

	r.channel, err = r.conn.Channel()
	if err != nil {
		log.Printf("Failed to open channel: %v", err)
		return
	}

	err = r.declareInfrastructure()
	if err != nil {
		log.Printf("Failed to declare infrastructure: %v", err)
	}

	log.Println("Connected to RabbitMQ")
}

func (r *RabbitMQClient) declareInfrastructure() error {
	err := r.channel.ExchangeDeclare("video.exchange", "topic", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to declare video exchange: %v", err)
	}

	err = r.channel.ExchangeDeclare("notification.exchange", "topic", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to declare notification exchange: %v", err)
	}

	_, err = r.channel.QueueDeclare("video.upload.queue", true, false, false, false,
		amqp.Table{
			"x-max-priority":            10,
			"x-message-ttl":             86400000,
			"x-dead-letter-exchange":    "video.dlx",
			"x-dead-letter-routing-key": "video.upload.dlq",
		})
	if err != nil {
		return fmt.Errorf("failed to declare video queue: %v", err)
	}

	_, err = r.channel.QueueDeclare("notification.queue", true, false, false, false,
		amqp.Table{
			"x-message-ttl":             3600000,
			"x-dead-letter-exchange":    "notification.dlx",
			"x-dead-letter-routing-key": "notification.dlq",
		})
	if err != nil {
		return fmt.Errorf("failed to declare notification queue: %v", err)
	}

	err = r.channel.QueueBind("video.upload.queue", "video.upload", "video.exchange", false, nil)
	if err != nil {
		return fmt.Errorf("failed to bind video queue: %v", err)
	}

	err = r.channel.QueueBind("notification.queue", "notification.#", "notification.exchange", false, nil)
	if err != nil {
		return fmt.Errorf("failed to bind notification queue: %v", err)
	}

	err = r.channel.Qos(1, 0, false)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %v", err)
	}

	return nil
}

func (r *RabbitMQClient) ensureConnection() error {
	if r.conn == nil || r.conn.IsClosed() {
		log.Println("RabbitMQ connection closed, reconnecting...")
		r.connect()
	}

	if r.conn == nil || r.conn.IsClosed() {
		return fmt.Errorf("failed to reconnect to RabbitMQ")
	}

	if r.channel == nil || r.channel.IsClosed() {
		var err error
		r.channel, err = r.conn.Channel()
		if err != nil {
			return fmt.Errorf("failed to recreate channel: %v", err)
		}
	}

	return nil
}

func (r *RabbitMQClient) Ping() error {
	if err := r.ensureConnection(); err != nil {
		return err
	}
	return nil
}

func (r *RabbitMQClient) Close() error {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

func (r *RabbitMQClient) PublishVideoUpload(message domain.VideoProcessingMessage, priority int) error {
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return r.channel.Publish(
		"video.exchange",
		"video.upload",
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
			Priority:     uint8(priority),
		},
	)
}

func (r *RabbitMQClient) SubscribeVideoUpload() (<-chan amqp.Delivery, error) {
	if err := r.ensureConnection(); err != nil {
		return nil, err
	}
	return r.channel.Consume(
		"video.upload.queue",
		"",
		false,
		false,
		false,
		false,
		nil,
	)
}

func (r *RabbitMQClient) PublishNotification(message domain.NotificationMessage) error {
	if err := r.ensureConnection(); err != nil {
		return err
	}

	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return r.channel.Publish(
		"notification.exchange",
		"notification.email",
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
		},
	)
}
