package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"

	demo "example.com/unstruct-temporal-demo"
	"github.com/lmittmann/tint"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/log"
)

func main() {
	// Set up structured logging with tinted output
	logger := slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:      slog.LevelInfo,
			TimeFormat: time.Kitchen,
			AddSource:  true,
		}),
	)
	slog.SetDefault(logger)

	logger.Info("Starting Temporal workflow starter for file-based unstruct extraction demo",
		slog.String("component", "starter"),
		slog.String("mode", "file_only"),
	)

	// Check for required environment variable
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		logger.Error("Missing required environment variable",
			slog.String("variable", "GEMINI_API_KEY"),
		)
		os.Exit(1)
	}

	logger.Info("Environment configuration validated",
		slog.Bool("gemini_api_key_set", true),
	)

	// Connect to Temporal service
	c, err := client.Dial(client.Options{
		HostPort: "localhost:7233",
		Logger:   log.NewStructuredLogger(logger),
	})
	if err != nil {
		logger.Error("Failed to connect to Temporal server",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	defer c.Close()

	logger.Info("Connected to Temporal server successfully")

	// Find available documents in docs/ directory
	docsDir := "docs"
	files, err := os.ReadDir(docsDir)
	if err != nil {
		logger.Error("Failed to read docs directory",
			slog.String("directory", docsDir),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	// Filter for markdown files
	var mdFiles []string
	for _, file := range files {
		if !file.IsDir() && (file.Name()[len(file.Name())-3:] == ".md") {
			mdFiles = append(mdFiles, file.Name())
		}
	}

	if len(mdFiles) == 0 {
		logger.Warn("No markdown files found in docs directory",
			slog.String("directory", docsDir),
		)
		os.Exit(1)
	}

	logger.Info("Found markdown files for processing",
		slog.String("directory", docsDir),
		slog.Int("file_count", len(mdFiles)),
		slog.Any("files", mdFiles),
	)

	// Process each file using file:// URI scheme
	for i, fileName := range mdFiles {
		logger.Info("Starting file extraction workflow",
			slog.String("test", "file URI-based document extraction"),
			slog.Int("test_number", i+1),
			slog.String("file_name", fileName),
		)

		// Create file:// URI with absolute path
		absPath, err := filepath.Abs(filepath.Join(docsDir, fileName))
		if err != nil {
			logger.Error("Failed to get absolute path",
				slog.String("file", fileName),
				slog.String("error", err.Error()),
			)
			continue
		}

		input := demo.WorkflowInput{
			Request: demo.DocumentRequest{
				ContentURI:  fmt.Sprintf("file://%s", absPath),
				DisplayName: "File URI: " + fileName,
			},
		}

		workflowID := "file-extract-" + fileName + "-" + strconv.FormatInt(time.Now().UnixNano(), 10)

		logger.Info("Executing workflow",
			slog.String("workflow_id", workflowID),
			slog.String("workflow_type", "DocumentExtractionWorkflow"),
			slog.String("task_queue", "unstruct-q"),
			slog.String("file_path", docsDir+"/"+fileName),
		)

		resp, err := c.ExecuteWorkflow(
			context.Background(),
			client.StartWorkflowOptions{
				ID:        workflowID,
				TaskQueue: "unstruct-q",
			},
			demo.DocumentExtractionWorkflow,
			input,
		)
		if err != nil {
			logger.Error("Failed to start workflow",
				slog.String("workflow_id", workflowID),
				slog.String("file_name", fileName),
				slog.String("error", err.Error()),
			)
			continue
		}

		logger.Info("Workflow started, waiting for completion",
			slog.String("workflow_id", workflowID),
			slog.String("file_name", fileName),
		)

		var result *demo.WorkflowOutput
		err = resp.Get(context.Background(), &result)
		if err != nil {
			logger.Error("Workflow execution failed",
				slog.String("workflow_id", workflowID),
				slog.String("file_name", fileName),
				slog.String("error", err.Error()),
			)
			continue
		}

		logger.Info("File extraction workflow completed",
			slog.String("workflow_id", workflowID),
			slog.String("file_name", fileName),
			slog.Bool("success", result.Success),
			slog.Duration("processing_time", result.Metadata.ProcessingTime),
			slog.Int("model_calls", result.Metadata.ModelCalls),
			slog.String("timestamp", result.Timestamp.Format(time.RFC3339)),
		)

		if result.Success {
			logger.Info("Extracted data summary",
				slog.String("file_name", fileName),
				slog.String("title", result.Data.Basic.Title),
				slog.String("author", result.Data.Basic.Author),
				slog.String("doc_type", result.Data.Basic.DocType),
				slog.Float64("budget", result.Data.Financial.Budget),
				slog.String("currency", result.Data.Financial.Currency),
				slog.String("project_code", result.Data.Project.Code),
				slog.String("project_status", result.Data.Project.Status),
				slog.Int("team_size", result.Data.Project.TeamSize),
				slog.String("contact_name", result.Data.Contact.Name),
				slog.String("contact_email", result.Data.Contact.Email),
			)
		}
	}

	logger.Info("File-based extraction demo completed",
		slog.String("status", "all files processed"),
		slog.Int("total_files", len(mdFiles)),
	)
}
