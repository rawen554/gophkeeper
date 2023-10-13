package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/rawen554/goph-keeper/internal/adapters/accrual"
	"github.com/rawen554/goph-keeper/internal/adapters/store"
	"github.com/rawen554/goph-keeper/internal/app"
	"github.com/rawen554/goph-keeper/internal/config"
	"github.com/rawen554/goph-keeper/internal/logger"
	"github.com/rawen554/goph-keeper/internal/processing"
)

const (
	timeoutServerShutdown = time.Second * 5
	timeoutShutdown       = time.Second * 10
	component             = "component"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancelCtx := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancelCtx()

	logger, err := logger.NewLogger()
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	config, err := config.ParseFlags()
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	storage, err := store.NewStore(ctx, config.DatabaseURI, config.LogLevel)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	wg := &sync.WaitGroup{}
	defer func() {
		wg.Wait()
	}()

	wg.Add(1)
	go func() {
		defer logger.Info("closed DB")
		defer wg.Done()
		<-ctx.Done()

		storage.Close()
	}()

	componentsErrs := make(chan error, 1)

	app := app.NewApp(config, storage, logger.With(component, "app"))
	srv, err := app.NewServer()
	if err != nil {
		logger.Fatalf("error creating server: %w", err)
	}

	go func(errs chan<- error) {
		if err := srv.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return
			}
			errs <- fmt.Errorf("run server has failed: %w", err)
		}
	}(componentsErrs)

	accrual, err := accrual.NewAccrualClient(config.AccrualAddr, logger.With(component, "accrual-client"))
	if err != nil {
		return fmt.Errorf("failed to create accrual client: %w", err)
	}

	processingInstance := processing.NewProcessingController(
		storage,
		accrual,
		logger.With(component, "processing-controller"),
	)

	go func(ctx context.Context) {
		processingInstance.Process(ctx)
	}(ctx)

	wg.Add(1)
	go func() {
		defer logger.Info("server has been shutdown")
		defer wg.Done()
		<-ctx.Done()

		shutdownTimeoutCtx, cancelShutdownTimeoutCtx := context.WithTimeout(context.Background(), timeoutServerShutdown)
		defer cancelShutdownTimeoutCtx()
		if err := srv.Shutdown(shutdownTimeoutCtx); err != nil {
			logger.Errorf("an error occurred during server shutdown: %v", err)
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-componentsErrs:
		logger.Error(err)
		cancelCtx()
	}

	go func() {
		ctx, cancelCtx := context.WithTimeout(context.Background(), timeoutShutdown)
		defer cancelCtx()

		<-ctx.Done()
		logger.Fatal("failed to gracefully shutdown the service")
	}()

	return nil
}
