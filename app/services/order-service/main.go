package main

import (
	"flag"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/snirkop89/ppe-ecommerce/core/logger"
	"github.com/snirkop89/ppe-ecommerce/core/publisher"
)

type config struct {
	addr  string
	kafka struct {
		server string
	}
}

func main() {
	var cfg config
	flag.StringVar(&cfg.addr, "addr", ":8080", "address to listen on, i.e 127.0.0.1:8000")
	flag.StringVar(&cfg.kafka.server, "kafka-server", "localhost", "kafka server address")
	flag.Parse()

	if !strings.HasPrefix(cfg.addr, ":") {
		cfg.addr = ":" + cfg.addr
	}

	log := logger.NewLogger("order-service")

	// Initialize kafka producer
	p, err := publisher.New(&kafka.ConfigMap{
		"boostrap.server": "localhost",
	})
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
	defer p.Close()

	// Setup routes
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(logger.LoggingMiddleware(log))

	r.Route("/v1", func(r chi.Router) {
		r.Get("/healthcheck", healthcheckHandler(log))
		r.Post("/orders", orderCreateHandler(log, p))
	})

	srv := &http.Server{
		Addr:         cfg.addr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// TODO: Graceful shutdown
	log.Info("Starting HTTP server", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Error(err.Error())
	}
}
