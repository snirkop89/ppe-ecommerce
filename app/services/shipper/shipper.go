package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/dgraph-io/badger/v4"
	v1 "github.com/snirkop89/ppe-ecommerce/api/v1"
	"github.com/snirkop89/ppe-ecommerce/core/httpio"
	"github.com/snirkop89/ppe-ecommerce/core/publisher"
)

func (app *application) consumeOrders(ctx context.Context) error {
	topic := "OrderPickedAndPacked"
	app.log.Info("Started consuming messages", "topic", topic)

	err := app.consumer.Subscribe(topic, nil)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := app.consumer.ReadMessage(10 * time.Second)
			if err != nil {
				if !err.(kafka.Error).IsTimeout() {
					app.log.Error("consuming", "error", err)
				}
				continue
			}

			// Parse msg
			var orderPicked v1.OrderPickedAndPacked
			err = httpio.Decode(bytes.NewReader(msg.Value), &orderPicked)
			if err != nil {
				app.handleError(orderPicked, err)
				continue
			}

			app.log.Info("Order picked and packed", "order", orderPicked)
			handled, err := app.alreadyHandled(orderPicked.Header.ID)
			if err != nil {
				app.handleError(orderPicked, err)
				continue
			}
			if handled {
				app.log.Info("Event already handled", "event_id", orderPicked.Header.ID)
				continue
			}
			if err := app.saveMessage(orderPicked.Header.ID); err != nil {
				app.log.Error("failed saving message", "error", err.Error())
			}

			// Publish OrderConfirmed
			app.publishNotification(ctx, orderPicked.Order)
		}
	}
}

func (app *application) alreadyHandled(eventID string) (bool, error) {
	found := false
	err := app.db.View(func(txn *badger.Txn) error {
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

func (app *application) saveMessage(eventID string) error {
	return app.db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(eventID), []byte("1")).WithTTL(7 * 24 * time.Hour)
		return txn.SetEntry(e)
	})
}

func (app *application) handleError(order v1.OrderPickedAndPacked, err error) {
	app.log.Error(err.Error())
	app.publishError(order)
}

func (app *application) publishError(order v1.OrderPickedAndPacked) error {
	var topic string = "DeadLetterQueue"
	errorEvent := v1.OrderError{
		Header: v1.NewHeader(),
		Event:  order,
	}
	if err := publisher.PublishEvent(app.config.Kafka.server, topic, errorEvent); err != nil {
		return err
	}
	return nil
}

func (app *application) publishNotification(ctx context.Context, confirmed v1.Order) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	topic := "Notification"
	notif := v1.Notification{
		Header:    v1.NewHeader(),
		Type:      "email",
		Recipient: confirmed.Customer.Email,
		From:      "orders@ppe4all",
		Subject:   fmt.Sprintf("Hi %s %s, your order is being shipped", confirmed.Customer.FirstName, confirmed.Customer.LastName),
		Body:      "<p>We have finished packing your pack. It's on it's way!</p>",
	}
	if err := publisher.PublishEvent(app.config.Kafka.server, topic, notif); err != nil {
		return err
	}
	return nil
}
