package main

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/dgraph-io/badger/v4"
	v1 "github.com/snirkop89/ppe-ecommerce/api/v1"
	"github.com/snirkop89/ppe-ecommerce/core/httpio"
	"github.com/snirkop89/ppe-ecommerce/core/publisher"
)

func (app *application) consumeOrders(ctx context.Context) error {
	app.log.Info("Started consuming messages", "topic", "Notification")

	err := app.consumer.Subscribe("Notification", nil)
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
			var notification v1.Notification
			err = httpio.Decode(bytes.NewReader(msg.Value), &notification)
			if err != nil {
				app.handleError(notification, err)
				continue
			}

			app.log.Info("notification received", "event", notification)
			handled, err := app.alreadyHandled(notification.Header.ID)
			if err != nil {
				app.handleError(notification, err)
				continue
			}
			if handled {
				app.log.Info("Event already handled", "event_id", notification.Header.ID)
				continue
			}
			if err := app.saveMessage(notification.Header.ID); err != nil {
				app.log.Error("failed saving message", "error", err.Error())
			}

			app.sendNotification(ctx, notification)
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

func (app *application) handleError(order v1.Notification, err error) {
	app.log.Error(err.Error())
	app.publishError(order)
}

func (app *application) publishError(notification v1.Notification) error {
	var topic string = "DeadLetterQueue"
	errorEvent := v1.OrderError{
		Header: v1.NewHeader(),
		Event:  notification,
	}
	if err := publisher.PublishEvent(app.config.Kafka.server, topic, errorEvent); err != nil {
		return err
	}
	return nil
}

func (app *application) sendNotification(ctx context.Context, notification v1.Notification) error {
	app.log.Info(
		"Sending notfication",
		"type", notification.Type,
		"from", notification.From,
		"subject", notification.Subject,
	)
	return nil
}
