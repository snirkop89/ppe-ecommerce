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
	app.log.Info("Started consuming messages", "topic", "OrderReceived")

	err := app.consumer.Subscribe("OrderReceived", nil)
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
			var orderReceived v1.OrderReceived
			err = httpio.Decode(bytes.NewReader(msg.Value), &orderReceived)
			if err != nil {
				app.handleError(orderReceived, err)
				continue
			}

			app.log.Info("Order received", "order", orderReceived)
			handled, err := app.alreadyHandled(orderReceived.Header.ID)
			if err != nil {
				app.handleError(orderReceived, err)
				continue
			}
			if handled {
				app.log.Info("Event already handled", "event_id", orderReceived.Header.ID)
				continue
			}
			if err := app.saveMessage(orderReceived.Header.ID); err != nil {
				app.log.Error("failed saving message", "error", err.Error())
			}

			// Publish OrderConfirmed
			app.publishOrderConfirmed(ctx, orderReceived.Order)
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

func (app *application) handleError(order v1.OrderReceived, err error) {
	app.log.Error(err.Error())
	app.publishError(order)
}

func (app *application) publishError(order v1.OrderReceived) error {
	var topic string = "DeadLetterQueue"
	errorEvent := v1.OrderError{
		Header: v1.NewHeader(),
		Event:  order,
	}
	if err := publisher.PublishEvent("localhost", topic, errorEvent); err != nil {
		return err
	}
	return nil
}

func (app *application) publishOrderConfirmed(ctx context.Context, confirmed v1.Order) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	topic := "OrderConfirmed"
	oc := v1.OrderConfirmed{
		Header: v1.NewHeader(),
		Order:  confirmed,
	}
	if err := publisher.PublishEvent("localhost", topic, oc); err != nil {
		return err
	}
	return nil
}
