package temporal_demo

import (
	"context"
	"fmt"
	"os"
	"time"

	unstruct "github.com/vivaneiona/genkit-unstruct"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// DocumentRequest represents a document processing request
type DocumentRequest struct {
	// Either provide text content or file path
	TextContent string `json:"text_content,omitempty"`
	FilePath    string `json:"file_path,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

// ExtractionTarget represents the structure we want to extract
type ExtractionTarget struct {
	// Basic document information - uses fast model
	Basic struct {
		Title   string `json:"title"`    // inherited from struct tag
		Author  string `json:"author"`   // inherited from struct tag
		DocType string `json:"doc_type"` // inherited from struct tag
		Date    string `json:"date"`     // inherited from struct tag
	} `json:"basic" unstruct:"basic,gemini-1.5-flash"`

	// Financial data - uses precise model
	Financial struct {
		Budget   float64 `json:"budget"`   // inherited from struct tag
		Currency string  `json:"currency"` // inherited from struct tag
	} `json:"financial" unstruct:"financial,gemini-1.5-pro"`

	// Project information - uses fast model
	Project struct {
		Code      string `json:"code"`       // inherited from struct tag
		Status    string `json:"status"`     // inherited from struct tag
		TeamSize  int    `json:"team_size"`  // inherited from struct tag
		StartDate string `json:"start_date"` // inherited from struct tag
		EndDate   string `json:"end_date"`   // inherited from struct tag
	} `json:"project" unstruct:"project,gemini-1.5-flash"`

	// Contact information - uses precise model with low temperature
	Contact struct {
		Name  string `json:"name"`  // inherited from struct tag
		Email string `json:"email"` // inherited from struct tag
		Phone string `json:"phone"` // inherited from struct tag
	} `json:"contact" unstruct:"person,gemini-1.5-pro?temperature=0.2"`
}

type WorkflowInput struct {
	Request DocumentRequest `json:"request"`
}

type WorkflowOutput struct {
	Data      ExtractionTarget `json:"data"`
	Success   bool             `json:"success"`
	Timestamp time.Time        `json:"timestamp"`
	Metadata  struct {
		ProcessingTime time.Duration `json:"processing_time"`
		ModelCalls     int           `json:"model_calls"`
		TokensUsed     int           `json:"tokens_used"`
	} `json:"metadata"`
}

// DocumentExtractionWorkflow demonstrates temporal workflow for document processing
func DocumentExtractionWorkflow(ctx workflow.Context, input WorkflowInput) (*WorkflowOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting document extraction workflow",
		"workflow_id", workflow.GetInfo(ctx).WorkflowExecution.ID,
		"run_id", workflow.GetInfo(ctx).WorkflowExecution.RunID,
		"request_type", getRequestType(input.Request),
		"display_name", input.Request.DisplayName,
	)

	startTime := workflow.Now(ctx)

	// Configure activity options with timeout and retry
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 3 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    100 * time.Second,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	logger.Info("Configured activity options",
		"timeout", "3m",
		"max_attempts", 3,
	)

	// Execute the document extraction activity
	var result ExtractionTarget
	err := workflow.ExecuteActivity(ctx, ExtractDocumentDataActivity, input.Request).Get(ctx, &result)
	if err != nil {
		logger.Error("Document extraction failed",
			"error", err.Error(),
			"workflow_id", workflow.GetInfo(ctx).WorkflowExecution.ID,
		)
		return &WorkflowOutput{
			Success:   false,
			Timestamp: workflow.Now(ctx),
		}, fmt.Errorf("extraction failed: %w", err)
	}

	processingTime := workflow.Now(ctx).Sub(startTime)
	logger.Info("Document extraction completed successfully",
		"processing_time", processingTime,
		"model_calls", 4,
		"workflow_id", workflow.GetInfo(ctx).WorkflowExecution.ID,
		"basic_title", result.Basic.Title,
		"project_code", result.Project.Code,
		"budget", result.Financial.Budget,
	)

	return &WorkflowOutput{
		Data:      result,
		Success:   true,
		Timestamp: workflow.Now(ctx),
		Metadata: struct {
			ProcessingTime time.Duration `json:"processing_time"`
			ModelCalls     int           `json:"model_calls"`
			TokensUsed     int           `json:"tokens_used"`
		}{
			ProcessingTime: processingTime,
			ModelCalls:     4, // basic, financial, project, contact
			TokensUsed:     0, // Would be populated in real implementation
		},
	}, nil
}

// Helper function to determine request type
func getRequestType(req DocumentRequest) string {
	if req.TextContent != "" {
		return "text_content"
	}
	if req.FilePath != "" {
		return "file_path"
	}
	return "unknown"
}

// ExtractDocumentDataActivity performs the actual document extraction
func ExtractDocumentDataActivity(ctx context.Context, req DocumentRequest) (ExtractionTarget, error) {
	logger := activity.GetLogger(ctx)

	// Get activity info for enhanced logging
	info := activity.GetInfo(ctx)

	logger.Info("Starting document extraction activity",
		"activity_id", info.ActivityID,
		"activity_type", info.ActivityType.Name,
		"workflow_id", info.WorkflowExecution.ID,
		"request_type", getRequestType(req),
		"display_name", req.DisplayName,
	)

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		logger.Error("Missing required environment variable", "variable", "GEMINI_API_KEY")
		return ExtractionTarget{}, fmt.Errorf("GEMINI_API_KEY not set")
	}

	logger.Info("Environment validated", "gemini_api_key_set", true)

	// Create GenAI client
	client, err := CreateDefaultGenAIClient(ctx, apiKey)
	if err != nil {
		logger.Error("Failed to create GenAI client", "error", err.Error())
		return ExtractionTarget{}, fmt.Errorf("failed to create GenAI client: %w", err)
	}

	logger.Info("GenAI client created successfully")

	// Create extractor with Twig templates
	extractor := unstruct.New[ExtractionTarget](client, Prompts)
	logger.Info("Unstruct extractor initialized")

	// Create assets based on input type
	var assets []unstruct.Asset
	if req.TextContent != "" {
		// Use text content directly
		logger.Info("Processing text content",
			"content_length", len(req.TextContent),
		)
		assets = []unstruct.Asset{
			unstruct.NewTextAsset(req.TextContent),
		}
	} else if req.FilePath != "" {
		// Use file asset (will upload to Files API)
		displayName := req.DisplayName
		if displayName == "" {
			displayName = fmt.Sprintf("Temporal Document Processing - %s", req.FilePath)
		}

		logger.Info("Processing file asset",
			"file_path", req.FilePath,
			"display_name", displayName,
		)

		assets = []unstruct.Asset{
			unstruct.NewFileAsset(
				client,
				req.FilePath,
				unstruct.WithDisplayName(displayName),
			),
		}
	} else {
		logger.Error("Invalid request: missing content", "text_content_empty", req.TextContent == "", "file_path_empty", req.FilePath == "")
		return ExtractionTarget{}, fmt.Errorf("either text_content or file_path must be provided")
	}

	logger.Info("Assets prepared for extraction", "asset_count", len(assets))

	// Extract structured data
	logger.Info("Starting unstruct extraction",
		"default_model", "gemini-1.5-flash",
		"timeout", "2m",
	)

	extractionStart := time.Now()
	result, err := extractor.Unstruct(
		ctx,
		assets,
		unstruct.WithModel("gemini-1.5-flash"), // Default model (can be overridden by struct tags)
		unstruct.WithTimeout(2*time.Minute),
	)
	extractionDuration := time.Since(extractionStart)

	if err != nil {
		logger.Error("Extraction failed",
			"error", err.Error(),
			"duration", extractionDuration,
		)
		return ExtractionTarget{}, fmt.Errorf("unstruct extraction failed: %w", err)
	}

	logger.Info("Extraction completed successfully",
		"duration", extractionDuration,
		"basic_title", result.Basic.Title,
		"basic_author", result.Basic.Author,
		"project_code", result.Project.Code,
		"budget", result.Financial.Budget,
		"contact_name", result.Contact.Name,
	)

	return *result, nil
}

// DryRunActivity performs a dry run to estimate costs without making API calls
func DryRunActivity(ctx context.Context, req DocumentRequest) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)

	// Get activity info for enhanced logging
	info := activity.GetInfo(ctx)

	logger.Info("Starting dry run activity",
		"activity_id", info.ActivityID,
		"workflow_id", info.WorkflowExecution.ID,
		"request_type", getRequestType(req),
	)

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		logger.Error("Missing required environment variable", "variable", "GEMINI_API_KEY")
		return nil, fmt.Errorf("GEMINI_API_KEY not set")
	}

	// Create GenAI client
	client, err := CreateDefaultGenAIClient(ctx, apiKey)
	if err != nil {
		logger.Error("Failed to create GenAI client", "error", err.Error())
		return nil, fmt.Errorf("failed to create GenAI client: %w", err)
	}

	logger.Info("GenAI client created for dry run")

	// Create extractor
	extractor := unstruct.New[ExtractionTarget](client, Prompts)

	// Create assets
	var assets []unstruct.Asset
	if req.TextContent != "" {
		logger.Info("Using provided text content for dry run", "content_length", len(req.TextContent))
		assets = []unstruct.Asset{unstruct.NewTextAsset(req.TextContent)}
	} else {
		// For dry run, we can use a placeholder if no text is provided
		logger.Info("Using placeholder text for dry run")
		assets = []unstruct.Asset{unstruct.NewTextAsset("Sample document for cost estimation")}
	}

	// Perform dry run
	logger.Info("Performing dry run analysis")
	dryRunStart := time.Now()
	stats, err := extractor.DryRun(ctx, assets, unstruct.WithModel("gemini-1.5-flash"))
	dryRunDuration := time.Since(dryRunStart)

	if err != nil {
		logger.Error("Dry run failed",
			"error", err.Error(),
			"duration", dryRunDuration,
		)
		return nil, fmt.Errorf("dry run failed: %w", err)
	}

	result := map[string]interface{}{
		"prompt_calls":        stats.PromptCalls,
		"total_input_tokens":  stats.TotalInputTokens,
		"total_output_tokens": stats.TotalOutputTokens,
		"model_calls":         stats.ModelCalls,
		"estimated_cost":      "See model_calls for breakdown",
	}

	logger.Info("Dry run completed successfully",
		"duration", dryRunDuration,
		"prompt_calls", stats.PromptCalls,
		"total_input_tokens", stats.TotalInputTokens,
		"total_output_tokens", stats.TotalOutputTokens,
		"model_calls", len(stats.ModelCalls),
	)

	return result, nil
}
