package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/go-chi/chi/v5"
)

type config struct {
	addr string
}

func main() {
	var cfg config
	flag.StringVar(&cfg.addr, "addr", ":8080", "address to listen on, i.e 127.0.0.1:8000")
	flag.Parse()

	if !strings.HasPrefix(cfg.addr, ":") {
		cfg.addr = ":" + cfg.addr
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))

	// Initialize kafka producer
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost",
	})
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer p.Close()

	// Setup routes
	r := chi.NewRouter()
	r.Route("/v1", func(r chi.Router) {
		r.Get("/healthcheck", healthcheckHandler(logger))
		r.Post("/orders", orderCreateHandler(logger, p))
	})

	srv := &http.Server{
		Addr:         cfg.addr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	logger.Info("Starting HTTP server", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		logger.Error(err.Error())
	}
}
