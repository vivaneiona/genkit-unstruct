package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"time"

	demo "example.com/unstruct-temporal-demo"
	"github.com/lmittmann/tint"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/log"
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

	logger.Info("Starting Temporal workflow starter for unstruct extraction demo",
		slog.String("component", "starter"),
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

	// Test 1: Text-based extraction
	logger.Info("Starting test case",
		slog.String("test", "text-based document extraction"),
		slog.Int("test_number", 1),
	)

	textInput := demo.WorkflowInput{
		Request: demo.DocumentRequest{
			TextContent: `
# Research Report - Market Analysis

**Author:** Dr. Emily Rodriguez  
**Date:** February 20, 2024  
**Department:** Research & Development  
**Classification:** Internal Use

## Executive Summary

This report presents a comprehensive market analysis for our upcoming product initiatives.

**Research Project Details:**
- Project Code: RES-MARKET-2024
- Project Name: Market Analysis Study
- Budget: $75,000.00 EUR
- Start Date: January 15, 2024
- End Date: June 30, 2024
- Status: Active
- Priority: High
- Project Lead: Dr. Emily Rodriguez
- Team Size: 5

## Contact Information
- Name: Dr. Emily Rodriguez
- Email: emily.rodriguez@company.com
- Phone: +1-555-0123
`,
			DisplayName: "Text-based Research Report",
		},
	}

	workflowID := "text-extract-" + strconv.FormatInt(time.Now().UnixNano(), 10)

	logger.Info("Executing workflow",
		slog.String("workflow_id", workflowID),
		slog.String("workflow_type", "DocumentExtractionWorkflow"),
		slog.String("task_queue", "unstruct-q"),
		slog.String("input_type", "text_content"),
	)

	resp, err := c.ExecuteWorkflow(
		context.Background(),
		client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: "unstruct-q",
		},
		demo.DocumentExtractionWorkflow,
		textInput,
	)
	if err != nil {
		logger.Error("Failed to start workflow",
			slog.String("workflow_id", workflowID),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	logger.Info("Workflow started, waiting for completion",
		slog.String("workflow_id", workflowID),
	)

	var textResult *demo.WorkflowOutput
	err = resp.Get(context.Background(), &textResult)
	if err != nil {
		logger.Error("Workflow execution failed",
			slog.String("workflow_id", workflowID),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	logger.Info("Text extraction workflow completed",
		slog.String("workflow_id", workflowID),
		slog.Bool("success", textResult.Success),
		slog.Duration("processing_time", textResult.Metadata.ProcessingTime),
		slog.Int("model_calls", textResult.Metadata.ModelCalls),
		slog.String("timestamp", textResult.Timestamp.Format(time.RFC3339)),
	)

	logger.Info("Extracted data summary",
		slog.String("title", textResult.Data.Basic.Title),
		slog.String("author", textResult.Data.Basic.Author),
		slog.String("doc_type", textResult.Data.Basic.DocType),
		slog.Float64("budget", textResult.Data.Financial.Budget),
		slog.String("currency", textResult.Data.Financial.Currency),
		slog.String("project_code", textResult.Data.Project.Code),
		slog.String("project_status", textResult.Data.Project.Status),
		slog.Int("team_size", textResult.Data.Project.TeamSize),
		slog.String("contact_name", textResult.Data.Contact.Name),
		slog.String("contact_email", textResult.Data.Contact.Email),
	)

	// Test 2: File-based extraction (if docs/research-report.md exists)
	logger.Info("Starting test case",
		slog.String("test", "file-based document extraction"),
		slog.Int("test_number", 2),
	)

	filePath := "docs/research-report.md"
	if _, err := os.Stat(filePath); err == nil {
		logger.Info("File found, proceeding with file-based extraction",
			slog.String("file_path", filePath),
		)

		fileInput := demo.WorkflowInput{
			Request: demo.DocumentRequest{
				FilePath:    filePath,
				DisplayName: "File-based Research Report Analysis",
			},
		}

		workflowID2 := "file-extract-" + strconv.FormatInt(time.Now().UnixNano(), 10)

		logger.Info("Executing file-based workflow",
			slog.String("workflow_id", workflowID2),
			slog.String("file_path", filePath),
		)

		resp2, err := c.ExecuteWorkflow(
			context.Background(),
			client.StartWorkflowOptions{
				ID:        workflowID2,
				TaskQueue: "unstruct-q",
			},
			demo.DocumentExtractionWorkflow,
			fileInput,
		)
		if err != nil {
			logger.Error("Failed to start file-based workflow",
				slog.String("workflow_id", workflowID2),
				slog.String("error", err.Error()),
			)
			os.Exit(1)
		}

		var fileResult *demo.WorkflowOutput
		err = resp2.Get(context.Background(), &fileResult)
		if err != nil {
			logger.Error("File-based workflow execution failed",
				slog.String("workflow_id", workflowID2),
				slog.String("error", err.Error()),
			)
			os.Exit(1)
		}

		logger.Info("File extraction workflow completed",
			slog.String("workflow_id", workflowID2),
			slog.Bool("success", fileResult.Success),
			slog.Duration("processing_time", fileResult.Metadata.ProcessingTime),
			slog.Int("model_calls", fileResult.Metadata.ModelCalls),
		)

		logger.Info("File extraction data summary",
			slog.String("title", fileResult.Data.Basic.Title),
			slog.String("author", fileResult.Data.Basic.Author),
			slog.String("project_code", fileResult.Data.Project.Code),
			slog.Float64("budget", fileResult.Data.Financial.Budget),
		)
	} else {
		logger.Warn("File not found, skipping file-based test",
			slog.String("file_path", filePath),
			slog.String("reason", "file does not exist"),
		)
	}

	logger.Info("Demo completed successfully",
		slog.String("status", "all tests finished"),
	)
}
