package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oziev02/ImageProcessor/internal/config"
	"github.com/oziev02/ImageProcessor/internal/migrations"
	"github.com/oziev02/ImageProcessor/internal/observability"
	"github.com/oziev02/ImageProcessor/internal/repo"
	"github.com/oziev02/ImageProcessor/internal/service"
	httptransport "github.com/oziev02/ImageProcessor/internal/transport/http"
	kafkatransport "github.com/oziev02/ImageProcessor/internal/transport/kafka"
)

type App struct {
	cfg           *config.Config
	logger        *slog.Logger
	db            *pgxpool.Pool
	httpServer    *httptransport.Server
	kafkaConsumer kafkatransport.Consumer
	processorSvc  service.ProcessorService
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	logger := observability.NewLogger()

	// Initialize database
	db, err := initDB(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize repositories
	imageRepo := repo.NewImageRepository(db)
	storageRepo := repo.NewStorageRepository(cfg.Storage.BasePath)

	// Initialize Kafka producer
	producer := kafkatransport.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic)

	// Initialize services
	imageSvc := service.NewImageService(imageRepo, storageRepo, producer, cfg)
	processorSvc := service.NewProcessorService(imageRepo, storageRepo, cfg)

	// Initialize Kafka consumer
	kafkaConsumer := kafkatransport.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Topic, cfg.Kafka.ConsumerGroup)

	// Initialize HTTP handler
	handler := httptransport.NewHandler(imageSvc, storageRepo)

	// Initialize HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	httpServer := httptransport.NewServer(addr, handler)

	return &App{
		cfg:           cfg,
		logger:        logger,
		db:            db,
		httpServer:    httpServer,
		kafkaConsumer: kafkaConsumer,
		processorSvc:  processorSvc,
	}, nil
}

func (a *App) Start() error {
	a.logger.Info("starting application", "addr", a.httpServer.Addr())

	// Start Kafka consumer in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := a.kafkaConsumer.Start(ctx, a.processorSvc); err != nil {
			a.logger.Error("kafka consumer error", "error", err)
		}
	}()

	// Start HTTP server
	go func() {
		if err := a.httpServer.Start(); err != nil {
			a.logger.Error("http server error", "error", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	a.logger.Info("shutting down application")

	// Shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	cancel() // Stop Kafka consumer

	if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown http server: %w", err)
	}

	if err := a.kafkaConsumer.Close(); err != nil {
		return fmt.Errorf("failed to close kafka consumer: %w", err)
	}

	a.db.Close()

	return nil
}

func initDB(cfg *config.Config, logger *slog.Logger) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
		cfg.Database.Password, cfg.Database.DBName, cfg.Database.SSLMode,
	)

	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Run migrations
	if err := runMigrations(cfg, logger); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Info("database initialized")
	return db, nil
}

func runMigrations(cfg *config.Config, logger *slog.Logger) error {
	// Create source driver from embedded filesystem
	sourceDriver, err := iofs.New(migrations.Files, ".")
	if err != nil {
		return fmt.Errorf("failed to create source driver: %w", err)
	}

	// Build DSN for migrate
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)

	// Create migrate instance
	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, dsn)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	// Run migrations
	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			logger.Info("database schema is up to date")
			return nil
		}
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Info("database migrations completed successfully")
	return nil
}
