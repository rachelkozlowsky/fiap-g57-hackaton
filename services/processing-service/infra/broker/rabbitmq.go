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
}

func InitRabbitMQ() *RabbitMQClient {
	url := utils.GetEnv("RABBITMQ_URL", "amqp://g57:g57123456@localhost:5672/")

	conn, err := amqp.Dial(url)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open channel: %v", err)
	}

	err = channel.ExchangeDeclare(
		"video.exchange",
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare exchange: %v", err)
	}

	err = channel.ExchangeDeclare(
		"notification.exchange",
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare exchange: %v", err)
	}

	_, err = channel.QueueDeclare(
		"video.upload.queue",
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-max-priority":            10,
			"x-message-ttl":             86400000,
			"x-dead-letter-exchange":    "video.dlx",
			"x-dead-letter-routing-key": "video.upload.dlq",
		},
	)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	_, err = channel.QueueDeclare(
		"notification.queue",
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-message-ttl":             3600000,
			"x-dead-letter-exchange":    "notification.dlx",
			"x-dead-letter-routing-key": "notification.dlq",
		},
	)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	err = channel.QueueBind(
		"video.upload.queue",
		"video.upload",
		"video.exchange",
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to bind queue: %v", err)
	}

	err = channel.QueueBind(
		"notification.queue",
		"notification.#",
		"notification.exchange",
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to bind queue: %v", err)
	}

	err = channel.Qos(
		1,
		0,
		false,
	)
	if err != nil {
		log.Fatalf("Failed to set QoS: %v", err)
	}

	log.Println("Connected to RabbitMQ")

	return &RabbitMQClient{
		conn:    conn,
		channel: channel,
	}
}

func (r *RabbitMQClient) Ping() error {
	if r.conn.IsClosed() {
		return fmt.Errorf("connection is closed")
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
