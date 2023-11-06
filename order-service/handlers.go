package main

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/snirkop89/ppe-ecommerce/core/httpio"
)

var (
	orderReceivedTopic = "OrderReceived"
)

func healthcheckHandler(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		msg := map[string]string{
			"status": "ok",
		}
		err := httpio.WriteJSON(w, http.StatusOK, msg)
		if err != nil {
			log.Error("Writing response", "error", err)
		}
	}
}

func publishEventHandler(log *slog.Logger, producer *kafka.Producer) http.HandlerFunc {
	type OrderReceived struct {
		ID      string `json:"id"`
		Product string `json:"product"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var orderReceived OrderReceived
		if err := httpio.Decode(r.Body, &orderReceived); err != nil {
			log.Error(err.Error())
			return
		}

		msg, err := json.Marshal(orderReceived)
		if err != nil {
			log.Error(err.Error())
			httpio.BadRequestResponse(w, err.Error())
			return
		}

		err = producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &orderReceivedTopic, Partition: kafka.PartitionAny},
			Value:          msg,
		}, nil)
		if err != nil {
			log.Error(err.Error())
			httpio.InternalServerErrorResponse(w, err.Error())
			return
		}
	}
}
