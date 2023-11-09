package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/dgraph-io/badger/v4"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	v1 "github.com/snirkop89/ppe-ecommerce/api/v1"
	"github.com/snirkop89/ppe-ecommerce/logger"
	"golang.org/x/sync/errgroup"
)

func main() {
	addr := flag.String("addr", ":8081", "address to listen on, i.e 127.0.0.1:8000")
	flag.Parse()

	log := logger.NewLogger("inventory-consumer")

	// Initialize kafka producer
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
	})
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
	defer p.Close()

	// Open th embedded database. Used for saving handles kafka messages,
	// to avoid duplication.
	db, err := badger.Open(badger.DefaultOptions("/tmp/inventory-consumer"))
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
	defer db.Close()

	// Prepare a context to catch cancelation signals.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()
	g, ctx := errgroup.WithContext(ctx)

	failedOrders := make(chan v1.OrderReceived)
	confirmedOrders := make(chan v1.Order)
	defer func() {
		close(failedOrders)
		close(confirmedOrders)
	}()

	g.Go(func() error {
		return consumeOrders(ctx, log, db, "OrderReceived", failedOrders, confirmedOrders)
	})

	go publishError(ctx, p, failedOrders)
	go publishOrderConfirmed(ctx, log, p, confirmedOrders)

	// Setup routes
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(logger.LoggingMiddleware(log))

	r.Route("/v1", func(r chi.Router) {
		r.Get("/healthcheck", healthcheckHandler)
	})

	srv := &http.Server{
		Addr:         *addr,
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
		return nil
	})
	// ########

	// Wait for any error in intialization for shutdown.
	err = g.Wait()
	if err != nil {
		log.Error(err.Error())
	}
}

func healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}
