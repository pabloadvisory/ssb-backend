package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pabloadvisory/ssb-backend/internal/config"
	"github.com/pabloadvisory/ssb-backend/internal/demo"
	"github.com/pabloadvisory/ssb-backend/internal/eventing"
	"github.com/pabloadvisory/ssb-backend/internal/maintenance"
	"github.com/pabloadvisory/ssb-backend/internal/observability"
	"github.com/pabloadvisory/ssb-backend/internal/platform/database"
	"github.com/pabloadvisory/ssb-backend/internal/platform/httpx"
	"github.com/pabloadvisory/ssb-backend/internal/push"
	"github.com/pabloadvisory/ssb-backend/internal/push/apns"
	"github.com/pabloadvisory/ssb-backend/internal/push/fcm"
	"github.com/pabloadvisory/ssb-backend/internal/realtime"
	postgresrepo "github.com/pabloadvisory/ssb-backend/internal/repository/postgres"
	"github.com/pabloadvisory/ssb-backend/internal/service"
	"github.com/pabloadvisory/ssb-backend/internal/transport/httpapi"
	"github.com/pabloadvisory/ssb-backend/migrations"
	"golang.org/x/sync/errgroup"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	command := "serve"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	if command == "help" || command == "-h" || command == "--help" {
		usage()
		return nil
	}
	if command == "version" {
		fmt.Println(version)
		return nil
	}
	if command == "healthcheck" {
		url := "http://localhost:8080/health/ready"
		if len(os.Args) > 2 {
			url = os.Args[2]
		}
		return healthcheck(url)
	}

	cfg, err := config.Load(command)
	if err != nil {
		return fmt.Errorf("configuration: %w", err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch command {
	case "serve":
		return serve(ctx, cfg, logger)
	case "migrate":
		if len(os.Args) < 3 || os.Args[2] != "up" {
			return errors.New("usage: ssb migrate up")
		}
		pool, err := database.Open(ctx, cfg.Database, "ssb-migrate")
		if err != nil {
			return err
		}
		defer pool.Close()
		if err := migrations.Up(ctx, pool); err != nil {
			return err
		}
		logger.Info("database migrations complete")
		return nil
	case "seed":
		if len(os.Args) < 3 || os.Args[2] != "demo" {
			return errors.New("usage: ssb seed demo")
		}
		if cfg.Environment == "production" {
			return errors.New("demo data cannot be seeded in production")
		}
		pool, err := database.Open(ctx, cfg.Database, "ssb-demo-seed")
		if err != nil {
			return err
		}
		defer pool.Close()
		if err := demo.Seed(ctx, pool); err != nil {
			return err
		}
		logger.Info("demo data seeded")
		return nil
	case "push-worker":
		return runPushWorker(ctx, cfg, logger)
	default:
		usage()
		return fmt.Errorf("unknown command %q", command)
	}
}

func runPushWorker(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	pool, err := database.Open(ctx, cfg.Database, "ssb-push-worker")
	if err != nil {
		return err
	}
	defer pool.Close()

	var apnsClient *apns.Client
	if cfg.Push.APNs.KeyPath != "" {
		apnsClient, err = apns.New(apns.Config{
			KeyPath: cfg.Push.APNs.KeyPath, KeyID: cfg.Push.APNs.KeyID, TeamID: cfg.Push.APNs.TeamID,
		})
		if err != nil {
			return err
		}
	}
	var fcmClient *fcm.Client
	if cfg.Push.FCMProjectID != "" {
		fcmClient, err = fcm.New(ctx, cfg.Push.FCMProjectID)
		if err != nil {
			return err
		}
	}

	repository := postgresrepo.New(pool)
	metrics := observability.NewSlog(logger)
	dispatcher := push.NewDispatcher(apnsClient, fcmClient, cfg.Push.ActivityType)
	worker := push.NewWorker(
		repository, dispatcher, logger, metrics, push.WorkerConfig{
			Concurrency: cfg.Push.Concurrency,
			MaxAttempts: cfg.Push.MaxAttempts, PollInterval: cfg.Push.PollInterval,
			LockDuration: cfg.Push.LockDuration, SendTimeout: cfg.Push.SendTimeout,
		},
	)
	logger.Info("push worker started", "apns", apnsClient != nil, "fcm", fcmClient != nil)
	if err := worker.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func serve(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	pool, err := database.Open(ctx, cfg.Database, "ssb-api")
	if err != nil {
		return err
	}
	defer pool.Close()

	metrics := observability.NewSlog(logger)
	hub := realtime.NewHub(metrics)
	listener := realtime.NewListener(pool, hub, logger)
	workerContext, cancelWorkers := context.WithCancel(ctx)
	workers, workerContext := errgroup.WithContext(workerContext)
	defer func() {
		cancelWorkers()
		_ = workers.Wait()
	}()
	workers.Go(func() error {
		listener.Run(workerContext)
		return nil
	})

	repository := postgresrepo.New(pool)
	outboxWorker := eventing.NewWorker(
		repository, logger, metrics, eventing.WorkerConfig{
			Concurrency: cfg.Outbox.Concurrency,
			MaxAttempts: cfg.Outbox.MaxAttempts, PollInterval: cfg.Outbox.PollInterval,
			LockDuration: cfg.Outbox.LockDuration, ProcessTimeout: cfg.Outbox.ProcessTimeout,
		},
	)
	workers.Go(func() error {
		outboxWorker.Run(workerContext)
		return nil
	})
	maintenanceWorker := maintenance.NewWorker(repository, logger, metrics, maintenance.Config{
		Interval: cfg.Maintenance.Interval, DeliveryRetention: cfg.Maintenance.DeliveryRetention,
		OutboxRetention: cfg.Maintenance.OutboxRetention, AuditRetention: cfg.Maintenance.AuditRetention,
		BatchSize: cfg.Maintenance.BatchSize,
	})
	workers.Go(func() error {
		maintenanceWorker.Run(workerContext)
		return nil
	})
	queueMonitor := observability.NewQueueMonitor(repository, metrics, logger, cfg.Observability.QueueHealthInterval)
	workers.Go(func() error {
		queueMonitor.Run(workerContext)
		return nil
	})
	footballService := service.NewFootball(repository)
	notificationService := service.NewNotifications(repository)
	clientIPs, err := httpx.NewClientIPResolver(cfg.HTTP.TrustedProxyCIDRs)
	if err != nil {
		return fmt.Errorf("configure trusted proxies: %w", err)
	}
	abuse := httpapi.AbuseControls{
		ClientIPs: clientIPs,
		Installations: httpx.NewRateLimiter(
			cfg.HTTP.InstallationRatePerMinute, time.Minute,
			cfg.HTTP.InstallationRateBurst, cfg.HTTP.AbuseTrackedIPLimit,
		),
		RealtimeConnections: httpx.NewConnectionLimiter(
			cfg.HTTP.RealtimeConnectionsPerIP, cfg.HTTP.AbuseTrackedIPLimit,
		),
	}
	api := httpapi.New(footballService, notificationService, pool, hub, logger, metrics, cfg.IngestKey, abuse)
	server := &http.Server{
		Addr:              cfg.HTTP.Address,
		Handler:           api.Handler(),
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
		MaxHeaderBytes:    1 << 20,
		BaseContext: func(net.Listener) context.Context {
			return workerContext
		},
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("HTTP server started", "address", cfg.HTTP.Address, "environment", cfg.Environment, "version", version)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		cancelWorkers()
		_ = workers.Wait()
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP server: %w", err)
		}
		return nil
	case <-ctx.Done():
		logger.Info("shutdown requested")
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		cancelWorkers()
		_ = workers.Wait()
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	cancelWorkers()
	if err := workers.Wait(); err != nil {
		return fmt.Errorf("stop background workers: %w", err)
	}
	logger.Info("shutdown complete")
	return nil
}

func healthcheck(url string) error {
	client := &http.Client{Timeout: 3 * time.Second}
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("healthcheck returned %s", response.Status)
	}
	return nil
}

func usage() {
	fmt.Fprintln(os.Stderr, `Usage:
  ssb serve
  ssb migrate up
  ssb seed demo
  ssb push-worker
  ssb healthcheck [URL]
  ssb version`)
}
