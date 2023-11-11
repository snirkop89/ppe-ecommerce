package publisher

import (
	"encoding/json"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type Producer struct {
	Client *kafka.Producer
}

func New(config *kafka.ConfigMap) (*Producer, error) {
	p, err := kafka.NewProducer(config)
	if err != nil {
		return nil, err
	}
	return &Producer{
		Client: p,
	}, nil
}

func (p *Producer) PublishEvent(topic string, data any) error {
	msg, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("publish event: %w", err)
	}

	err = p.Client.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic},
		Value:          msg,
	}, nil)
	if err != nil {
		return fmt.Errorf("publish event: %w", err)
	}
	return nil

}

func (p *Producer) Close() {
	p.Client.Close()
}

func PublishEvent(hostAddress, topic string, data any) error {
	p, err := New(&kafka.ConfigMap{
		"bootstrap.servers": hostAddress,
	})
	if err != nil {
		return fmt.Errorf("publis event: %w", err)
	}
	defer p.Close()

	return p.PublishEvent(topic, data)
}
