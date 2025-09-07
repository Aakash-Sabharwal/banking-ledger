package queue

import (
	"context"
	"fmt"
	"log"
	"time"

	"banking-ledger/internal/domain"

	"github.com/streadway/amqp"
)

// RabbitMQQueue implements the MessageQueue interface
type RabbitMQQueue struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	url     string
}

// NewRabbitMQQueue creates a new RabbitMQ queue
func NewRabbitMQQueue(url string) (domain.MessageQueue, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	return &RabbitMQQueue{
		conn:    conn,
		channel: channel,
		url:     url,
	}, nil
}

// Publish publishes a message to a queue
func (q *RabbitMQQueue) Publish(ctx context.Context, queueName string, message []byte) error {
	// Declare queue to ensure it exists
	_, err := q.channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Set message properties for persistence
	msg := amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/json",
		Body:         message,
		Timestamp:    time.Now(),
	}

	err = q.channel.Publish(
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,     // immediate
		msg,
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// Subscribe subscribes to a queue and processes messages
func (q *RabbitMQQueue) Subscribe(ctx context.Context, queueName string, handler func([]byte) error) error {
	// Declare queue to ensure it exists
	queue, err := q.channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Set QoS to process one message at a time
	err = q.channel.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// Start consuming messages
	msgs, err := q.channel.Consume(
		queue.Name, // queue
		"",         // consumer
		false,      // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	// Process messages in a goroutine
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}

				// Process message with retry logic
				err := q.processMessageWithRetry(msg, handler)
				if err != nil {
					log.Printf("Failed to process message after retries: %v", err)
					// Reject message and don't requeue (send to DLQ if configured)
					msg.Nack(false, false)
				} else {
					// Acknowledge successful processing
					msg.Ack(false)
				}
			}
		}
	}()

	return nil
}

// processMessageWithRetry processes a message with retry logic
func (q *RabbitMQQueue) processMessageWithRetry(msg amqp.Delivery, handler func([]byte) error) error {
	const maxRetries = 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := handler(msg.Body)
		if err == nil {
			return nil
		}

		lastErr = err
		log.Printf("Message processing failed (attempt %d/%d): %v", attempt, maxRetries, err)

		if attempt < maxRetries {
			// Exponential backoff
			backoff := time.Duration(attempt) * time.Second
			time.Sleep(backoff)
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

// Close closes the connection
func (q *RabbitMQQueue) Close() error {
	if q.channel != nil {
		if err := q.channel.Close(); err != nil {
			log.Printf("Error closing channel: %v", err)
		}
	}

	if q.conn != nil {
		if err := q.conn.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
	}

	return nil
}

// reconnect attempts to reconnect to RabbitMQ
func (q *RabbitMQQueue) reconnect() error {
	if q.conn != nil && !q.conn.IsClosed() {
		return nil
	}

	conn, err := amqp.Dial(q.url)
	if err != nil {
		return fmt.Errorf("failed to reconnect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel on reconnection: %w", err)
	}

	q.conn = conn
	q.channel = channel

	return nil
}
