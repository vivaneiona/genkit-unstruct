package main

import (
	"log/slog"
	"os"
	"time"

	demo "example.com/unstruct-temporal-demo"
	"github.com/lmittmann/tint"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/worker"
)

func main() {
	// Setup tinted logger with slog
	logger := slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:      slog.LevelInfo,
			TimeFormat: time.Kitchen,
			AddSource:  true,
		}),
	)
	slog.SetDefault(logger)

	logger.Info("Starting Temporal worker for unstruct extraction demo",
		slog.String("component", "worker"),
		slog.String("task_queue", "unstruct-q"),
	)

	c, err := client.Dial(client.Options{
		HostPort: "localhost:7233",
		Logger:   log.NewStructuredLogger(logger),
	})
	if err != nil {
		logger.Error("Failed to connect to Temporal server",
			slog.String("error", err.Error()),
			slog.String("server", "localhost:7233"),
		)
		os.Exit(1)
	}
	defer c.Close()

	logger.Info("Connected to Temporal server successfully",
		slog.String("server", "localhost:7233"),
	)

	w := worker.New(c, "unstruct-q", worker.Options{})

	// Register workflows
	w.RegisterWorkflow(demo.DocumentExtractionWorkflow)
	logger.Info("Registered workflow",
		slog.String("workflow", "DocumentExtractionWorkflow"),
	)

	// Register activities
	w.RegisterActivity(demo.ExtractDocumentDataActivity)
	w.RegisterActivity(demo.ExtractSectionActivity)
	w.RegisterActivity(demo.DryRunActivity)
	logger.Info("Registered activities",
		slog.String("activities", "ExtractDocumentDataActivity, ExtractSectionActivity, DryRunActivity"),
	)

	logger.Info("Worker configuration complete",
		slog.String("features", "Twig templates, media assets support"),
		slog.String("status", "starting"),
	)

	if err := w.Run(worker.InterruptCh()); err != nil {
		logger.Error("Worker execution failed",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	logger.Info("Worker shutting down gracefully")
}
