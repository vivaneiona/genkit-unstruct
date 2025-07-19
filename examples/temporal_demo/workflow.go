package temporal_demo

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	unstruct "github.com/vivaneiona/genkit-unstruct"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/genai"
)

// DocumentRequest represents a document processing request with file URI-based content sources
type DocumentRequest struct {
	// Content source using file:// URI scheme for local files:
	// - "file://docs/document.md" for local markdown files
	// - "file://docs/report.txt" for local text files
	// Note: Only file:// scheme is supported in this file-only version
	ContentURI  string `json:"content_uri"`
	DisplayName string `json:"display_name,omitempty"`
}

// ExtractionRequest represents a request for extracting a specific section
type ExtractionRequest struct {
	Request DocumentRequest        `json:"request"`
	Section string                 `json:"section"`           // "basic", "financial", "project", "contact"
	Model   string                 `json:"model"`             // Which model to use
	Options map[string]interface{} `json:"options,omitempty"` // Additional options like temperature
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
// This implementation uses a single activity call for complete document extraction
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
	activityCtx := workflow.WithActivityOptions(ctx, ao)

	logger.Info("Configured activity options",
		"timeout", "3m",
		"max_attempts", 3,
	)

	// Execute single document extraction activity directly (no need for TemporalRunner for single activity)
	logger.Info("Starting document extraction activity execution")

	var finalResult ExtractionTarget
	err := workflow.ExecuteActivity(activityCtx, ExtractDocumentDataActivity, input.Request).Get(activityCtx, &finalResult)
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

	logger.Info("Document extraction completed successfully",
		"title", finalResult.Basic.Title,
		"author", finalResult.Basic.Author,
		"project_code", finalResult.Project.Code,
		"budget", finalResult.Financial.Budget,
		"contact_name", finalResult.Contact.Name,
	)

	processingTime := workflow.Now(ctx).Sub(startTime)
	logger.Info("Document extraction workflow completed successfully",
		"processing_time", processingTime,
		"model_calls", 1,
		"workflow_id", workflow.GetInfo(ctx).WorkflowExecution.ID,
		"basic_title", finalResult.Basic.Title,
		"project_code", finalResult.Project.Code,
		"budget", finalResult.Financial.Budget,
	)

	return &WorkflowOutput{
		Data:      finalResult,
		Success:   true,
		Timestamp: workflow.Now(ctx),
		Metadata: struct {
			ProcessingTime time.Duration `json:"processing_time"`
			ModelCalls     int           `json:"model_calls"`
			TokensUsed     int           `json:"tokens_used"`
		}{
			ProcessingTime: processingTime,
			ModelCalls:     1, // Single extraction activity call
			TokensUsed:     0, // Would be populated in real implementation
		},
	}, nil
}

// getRequestType determines content type from URL scheme or falls back to legacy fields
func getRequestType(req DocumentRequest) string {
	// Try modern URL-based approach first
	if req.ContentURI != "" {
		if parsedURL, err := url.Parse(req.ContentURI); err == nil && parsedURL.Scheme != "" {
			switch parsedURL.Scheme {
			case "file":
				return "file_path"
			case "http", "https":
				return "remote_url"
			default:
				// Unknown scheme, try to infer from content
				return "remote_url"
			}
		}
		// If ContentURI is set but unparseable, treat as text
		return "text_content"
	}

	return "unknown"
}

// Helper functions to reduce code duplication

// createGenAIClientFromEnv creates a GenAI client using the GEMINI_API_KEY environment variable
func createGenAIClientFromEnv(ctx context.Context, logger interface{ Error(string, ...interface{}) }) (*genai.Client, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		logger.Error("Missing required environment variable", "variable", "GEMINI_API_KEY")
		return nil, fmt.Errorf("GEMINI_API_KEY not set")
	}

	client, err := CreateDefaultGenAIClient(ctx, apiKey)
	if err != nil {
		logger.Error("Failed to create GenAI client", "error", err.Error())
		return nil, fmt.Errorf("failed to create GenAI client: %w", err)
	}

	return client, nil
}

// createAssetsFromRequest creates unstruct.Asset slice from DocumentRequest
func createAssetsFromRequest(ctx context.Context, req DocumentRequest, client *genai.Client, logger interface {
	Error(string, ...interface{})
	Info(string, ...interface{})
}, purposePrefix string) ([]unstruct.Asset, error) {
	if req.ContentURI == "" {
		logger.Error("Invalid request: missing content URI")
		return nil, fmt.Errorf("ContentURI must be provided with file:// scheme")
	}

	// Handle modern URI-based content
	parsedURL, err := url.Parse(req.ContentURI)
	if err != nil {
		logger.Error("Failed to parse content URI", "uri", req.ContentURI, "error", err.Error())
		return nil, fmt.Errorf("invalid content URI: %w", err)
	}

	switch parsedURL.Scheme {
	case "file":
		// Extract file path from file:// URI
		filePath := parsedURL.Path
		// Remove leading slash for relative paths
		if len(filePath) > 0 && filePath[0] == '/' {
			filePath = filePath[1:]
		}

		logger.Info("Processing file URI",
			"file_path", filePath,
			"original_uri", req.ContentURI,
			"parsed_path", parsedURL.Path,
		)

		displayName := req.DisplayName
		if displayName == "" {
			displayName = fmt.Sprintf("%s - %s", purposePrefix, filePath)
		}

		return []unstruct.Asset{
			unstruct.NewFileAsset(
				client,
				filePath,
				unstruct.WithDisplayName(displayName),
			),
		}, nil

	case "data":
		// Handle data: URIs for inline content (not used in this file-only version)
		logger.Error("Data URIs not supported in file-only mode", "uri", req.ContentURI)
		return nil, fmt.Errorf("data URIs not supported in file-only mode")

	default:
		logger.Error("Unsupported URI scheme", "scheme", parsedURL.Scheme, "uri", req.ContentURI)
		return nil, fmt.Errorf("unsupported URI scheme: %s", parsedURL.Scheme)
	}
}

// extractSectionData performs extraction for a specific section type
func extractSectionData(ctx context.Context, section string, model string, assets []unstruct.Asset, client *genai.Client, logger interface {
	Error(string, ...interface{})
}) (map[string]interface{}, error) {
	switch section {
	case "basic":
		type BasicInfo struct {
			Title   string `json:"title"`
			Author  string `json:"author"`
			DocType string `json:"doc_type"`
			Date    string `json:"date"`
		}

		extractor := unstruct.New[BasicInfo](client, Prompts)
		result, err := extractor.Unstruct(ctx, assets, unstruct.WithModel(model), unstruct.WithTimeout(2*time.Minute))
		if err != nil {
			logger.Error("Basic section extraction failed", "error", err.Error())
			return nil, fmt.Errorf("basic section extraction failed: %w", err)
		}

		return map[string]interface{}{
			"title":    result.Title,
			"author":   result.Author,
			"doc_type": result.DocType,
			"date":     result.Date,
		}, nil

	case "financial":
		type FinancialInfo struct {
			Budget   float64 `json:"budget"`
			Currency string  `json:"currency"`
		}

		extractor := unstruct.New[FinancialInfo](client, Prompts)
		result, err := extractor.Unstruct(ctx, assets, unstruct.WithModel(model), unstruct.WithTimeout(2*time.Minute))
		if err != nil {
			logger.Error("Financial section extraction failed", "error", err.Error())
			return nil, fmt.Errorf("financial section extraction failed: %w", err)
		}

		return map[string]interface{}{
			"budget":   result.Budget,
			"currency": result.Currency,
		}, nil

	case "project":
		type ProjectInfo struct {
			Code      string `json:"code"`
			Status    string `json:"status"`
			TeamSize  int    `json:"team_size"`
			StartDate string `json:"start_date"`
			EndDate   string `json:"end_date"`
		}

		extractor := unstruct.New[ProjectInfo](client, Prompts)
		result, err := extractor.Unstruct(ctx, assets, unstruct.WithModel(model), unstruct.WithTimeout(2*time.Minute))
		if err != nil {
			logger.Error("Project section extraction failed", "error", err.Error())
			return nil, fmt.Errorf("project section extraction failed: %w", err)
		}

		return map[string]interface{}{
			"code":       result.Code,
			"status":     result.Status,
			"team_size":  result.TeamSize,
			"start_date": result.StartDate,
			"end_date":   result.EndDate,
		}, nil

	case "contact":
		type ContactInfo struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			Phone string `json:"phone"`
		}

		extractor := unstruct.New[ContactInfo](client, Prompts)
		result, err := extractor.Unstruct(ctx, assets, unstruct.WithModel(model), unstruct.WithTimeout(2*time.Minute))
		if err != nil {
			logger.Error("Contact section extraction failed", "error", err.Error())
			return nil, fmt.Errorf("contact section extraction failed: %w", err)
		}

		return map[string]interface{}{
			"name":  result.Name,
			"email": result.Email,
			"phone": result.Phone,
		}, nil

	default:
		logger.Error("Unsupported section for extraction", "section", section)
		return nil, fmt.Errorf("unsupported section: %s", section)
	}
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
		"content_uri", req.ContentURI,
	)

	// Create GenAI client using helper function
	client, err := createGenAIClientFromEnv(ctx, logger)
	if err != nil {
		return ExtractionTarget{}, err
	}

	logger.Info("GenAI client created successfully")

	// Create extractor with Twig templates
	extractor := unstruct.New[ExtractionTarget](client, Prompts)
	logger.Info("Unstruct extractor initialized")

	// Create assets using helper function
	assets, err := createAssetsFromRequest(ctx, req, client, logger, "Temporal Document Processing")
	if err != nil {
		return ExtractionTarget{}, err
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

	// Create GenAI client using helper function
	client, err := createGenAIClientFromEnv(ctx, logger)
	if err != nil {
		return nil, err
	}

	logger.Info("GenAI client created for dry run")

	// Create extractor
	extractor := unstruct.New[ExtractionTarget](client, Prompts)

	// Create assets using helper function
	assets, err := createAssetsFromRequest(ctx, req, client, logger, "Dry Run")
	if err != nil {
		return nil, err
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

// ExtractSectionActivity extracts a specific section of the document
func ExtractSectionActivity(ctx context.Context, req ExtractionRequest) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)

	// Get activity info for enhanced logging
	info := activity.GetInfo(ctx)

	logger.Info("Starting section extraction activity",
		"activity_id", info.ActivityID,
		"activity_type", info.ActivityType.Name,
		"workflow_id", info.WorkflowExecution.ID,
		"section", req.Section,
		"model", req.Model,
		"request_type", getRequestType(req.Request),
		"display_name", req.Request.DisplayName,
	)

	// Create GenAI client using helper function
	client, err := createGenAIClientFromEnv(ctx, logger)
	if err != nil {
		return nil, err
	}

	logger.Info("GenAI client created successfully")

	// Create assets using helper function
	purposePrefix := fmt.Sprintf("Temporal Section Processing - %s", req.Section)
	assets, err := createAssetsFromRequest(ctx, req.Request, client, logger, purposePrefix)
	if err != nil {
		return nil, err
	}

	logger.Info("Assets prepared for section extraction", "asset_count", len(assets))

	// Create targeted extractor for section-specific extraction
	logger.Info("Starting section extraction",
		"section", req.Section,
		"model", req.Model,
	)

	extractionStart := time.Now()

	// Define section-specific extraction types
	model := req.Model
	if model == "" {
		model = "gemini-1.5-flash" // Default model
	}

	// Extract section data using helper function
	result, err := extractSectionData(ctx, req.Section, model, assets, client, logger)
	if err != nil {
		return nil, err
	}

	extractionDuration := time.Since(extractionStart)

	logger.Info("Section extraction completed successfully",
		"section", req.Section,
		"duration", extractionDuration,
		"fields_extracted", len(result),
	)

	return result, nil
}
