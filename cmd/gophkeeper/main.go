package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/rawen554/goph-keeper/internal/adapters/store"
	"github.com/rawen554/goph-keeper/internal/app"
	"github.com/rawen554/goph-keeper/internal/config"
	"github.com/rawen554/goph-keeper/internal/logger"
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

	storage, err := store.NewStore(ctx, config.DatabaseDSN, config.LogLevel)
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

	a := app.NewApp(config, storage, logger.Named("app"))
	srv, err := a.NewServer()
	if err != nil {
		logger.Fatalf("error creating server: %w", err)
	}

	go func(errs chan<- error) {
		if config.EnableHTTPS {
			_, errCert := os.ReadFile(config.TLSCertPath)
			_, errKey := os.ReadFile(config.TLSKeyPath)

			if errors.Is(errCert, os.ErrNotExist) || errors.Is(errKey, os.ErrNotExist) {
				privateKey, certBytes, err := app.CreateCertificates(logger.Named("certs-builder"))
				if err != nil {
					errs <- fmt.Errorf("error creating tls certs: %w", err)
					return
				}

				if err := app.WriteCertificates(certBytes, config.TLSCertPath, privateKey, config.TLSKeyPath, logger); err != nil {
					errs <- fmt.Errorf("error writing tls certs: %w", err)
					return
				}
			}

			srv.TLSConfig = &tls.Config{
				MinVersion:         tls.VersionTLS12,
				ClientAuth:         tls.RequestClientCert,
				KeyLogWriter:       bufio.NewWriter(os.Stdout),
				InsecureSkipVerify: true,
			}

			if err := srv.ListenAndServeTLS(config.TLSCertPath, config.TLSKeyPath); err != nil {
				if errors.Is(err, http.ErrServerClosed) {
					return
				}
				errs <- fmt.Errorf("run tls server has failed: %w", err)
				return
			}
		}

		logger.Warnf("serving http server %s without TLS: Use only for development", srv.Addr)
		if err := srv.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return
			}
			errs <- fmt.Errorf("run server has failed: %w", err)
		}
	}(componentsErrs)

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
