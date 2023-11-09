package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
	v1 "github.com/snirkop89/ppe-ecommerce/api/v1"
	"github.com/snirkop89/ppe-ecommerce/core/httpio"
)

func consumeOrders(ctx context.Context, log *slog.Logger, db *badger.DB, topic string, failedOrders chan<- v1.OrderReceived, confirmed chan<- v1.Order) error {
	log.Info("Started consuming messages", "topic", topic)
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"group.id":          "inventory",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		return err
	}
	defer c.Close()

	err = c.Subscribe(topic, nil)
	if err != nil {
		return err
	}

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		msg, err := c.ReadMessage(5 * time.Second)
		if err != nil {
			if !err.(kafka.Error).IsTimeout() {
				log.Error("consuming", "error", err)
			}
			continue
		}

		// Parse msg
		var orderReceived v1.OrderReceived
		err = httpio.Decode(bytes.NewReader(msg.Value), &orderReceived)
		if err != nil {
			log.Error("decoding message", "error", err)
			failedOrders <- orderReceived
			continue
		}

		log.Info("Order received", "order", orderReceived)
		handled, err := alreadyHandled(db, orderReceived.Header.ID)
		if err != nil {
			log.Error(err.Error())
			failedOrders <- orderReceived
			continue
		}
		if handled {
			log.Info("Event already handled", "event_id", orderReceived.Header.ID)
			continue
		}
		if err := saveMessage(db, orderReceived.Header.ID); err != nil {
			log.Error("failed saving message", "error", err.Error())
		}

		// Publish OrderConfirmed
		confirmed <- orderReceived.Order
	}
}

func alreadyHandled(db *badger.DB, eventID string) (bool, error) {
	found := false
	err := db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(eventID))
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return nil
			}
			return err
		}
		found = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return found, nil
}

func saveMessage(db *badger.DB, eventID string) error {
	return db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(eventID), []byte("1")).WithTTL(7 * 24 * time.Hour)
		return txn.SetEntry(e)
	})
}

func publishError(ctx context.Context, p *kafka.Producer, failedOrders <-chan v1.OrderReceived) {
	var topic string = "DeadLetterQueue"
	for {
		select {
		case order := <-failedOrders:
			errorEvent := v1.OrderError{
				Header: v1.Header{
					ID:          uuid.NewString(),
					PublishedAt: time.Now(),
				},
				Event: order,
			}
			data, err := json.Marshal(errorEvent)
			if err != nil {
				continue
			}
			p.Produce(&kafka.Message{
				TopicPartition: kafka.TopicPartition{Topic: &topic},
				Value:          data,
			}, nil)
			slog.Info("Published error event")
		case <-ctx.Done():
			return
		}
	}
}

func publishOrderConfirmed(ctx context.Context, log *slog.Logger, p *kafka.Producer, confirmed <-chan v1.Order) {
	topic := "OrderConfirmed"
	for {
		select {
		case <-ctx.Done():
			return
		case order := <-confirmed:
			oc := v1.OrderConfirmed{
				Header: v1.NewHeader(),
				Order:  order,
			}
			msg, err := json.Marshal(oc)
			if err != nil {
				log.Error(err.Error())
			}
			err = p.Produce(&kafka.Message{
				TopicPartition: kafka.TopicPartition{Topic: &topic},
				Value:          msg,
			}, nil)
			if err != nil {
				log.Error(err.Error())
			}
		}
	}
}
