package main

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
	v1 "github.com/snirkop89/ppe-ecommerce/api/v1"
	"github.com/snirkop89/ppe-ecommerce/core/httpio"
	"github.com/snirkop89/ppe-ecommerce/validator"
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

func orderCreateHandler(log *slog.Logger, producer *kafka.Producer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Products []v1.Product `json:"products"`
			Customer v1.Customer  `json:"customer"`
		}

		if err := httpio.Decode(r.Body, &input); err != nil {
			log.Error(err.Error())
			return
		}

		order := v1.Order{
			OrderID:  uuid.NewString(),
			Products: input.Products,
			Customer: input.Customer,
		}

		v := validator.New()
		if v1.ValidateOrder(v, &order); !v.Valid() {
			log.With("error", v.Errors).Error("failed validating order")
			httpio.FailedValidationResponse(w, r, v.Errors)
			return
		}

		msg, err := json.Marshal(order.ToOrderReceivedEvent())
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

		err = httpio.WriteJSON(w, http.StatusAccepted, map[string]any{
			"message": "order accepted",
		})
		if err != nil {
			log.Error(err.Error())
			w.WriteHeader(500)
		}
	}
}
