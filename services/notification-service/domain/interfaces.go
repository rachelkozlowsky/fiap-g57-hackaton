package domain

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

type DatabaseInterface interface {

	CreateNotification(notification *Notification) error
	UpdateNotification(notification *Notification) error
	Ping() error
	Close() error
}

type SMTPInterface interface {
	SendEmail(to, subject, body string) error
}

type RabbitMQInterface interface {
	SubscribeNotification() (<-chan amqp.Delivery, error)
}

type AuthServiceClient interface {
	GetUserByID(userID string) (*User, error)
}

type VideoServiceClient interface {
	GetVideoByID(videoID string) (*Video, error)
}
