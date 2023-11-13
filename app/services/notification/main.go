package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/dgraph-io/badger/v4"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/snirkop89/ppe-ecommerce/core/httpio"
	"github.com/snirkop89/ppe-ecommerce/core/logger"
	"golang.org/x/sync/errgroup"
)

type config struct {
	Addr   string
	DBPath string
	Kafka  struct {
		server string
	}
}

type application struct {
	config   config
	log      *slog.Logger
	consumer *kafka.Consumer
	db       *badger.DB
}

func main() {

	var cfg config
	flag.StringVar(&cfg.Addr, "addr", ":8081", "address to listen on, i.e 127.0.0.1:8000")
	flag.StringVar(&cfg.DBPath, "db-path", "/tmp/notification-consumer", "directory to create database")
	flag.StringVar(&cfg.Kafka.server, "kafka-server", "localhost", "kafka server address")
	flag.Parse()

	log := logger.NewLogger("notification-consumer")

	// Open th embedded database. Used for saving handles kafka messages,
	// to avoid duplication.
	db, err := badger.Open(badger.DefaultOptions("/tmp/notification-consumer"))
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
	defer db.Close()

	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": cfg.Kafka.server,
		"group.id":          "notification-consumers",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
	defer c.Close()
	app := &application{
		config:   cfg,
		log:      log,
		consumer: c,
		db:       db,
	}

	// Prepare a context to catch cancelation signals.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return app.consumeOrders(ctx)
	})

	// Setup routes
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(logger.LoggingMiddleware(log))

	r.Route("/v1", func(r chi.Router) {
		r.Get("/healthcheck", httpio.HealthCheckHandler)
	})

	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// ######  HTTP server
	g.Go(func() error {
		log.Info("Starting HTTP server", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	g.Go(func() error {
		<-ctx.Done()
		log.Info("Received termination signal. Shutting down server")

		tCtx, tcancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer tcancel()

		err = srv.Shutdown(tCtx)
		if err != nil {
			log.Error(err.Error())
			return err
		}
		log.Info("Server shutdown completed")
		return nil
	})
	// ########

	// Wait for any error in intialization for shutdown.
	err = g.Wait()
	if err != nil {
		log.Error(err.Error())
	}
}
