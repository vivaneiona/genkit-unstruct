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

	if req.ContentURI != "" {
		// Handle modern URI-based content
		parsedURL, err := url.Parse(req.ContentURI)
		if err != nil {
			logger.Error("Failed to parse content URI", "uri", req.ContentURI, "error", err.Error())
			return ExtractionTarget{}, fmt.Errorf("invalid content URI: %w", err)
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
				displayName = fmt.Sprintf("Temporal Document Processing - %s", filePath)
			}

			assets = []unstruct.Asset{
				unstruct.NewFileAsset(
					client,
					filePath,
					unstruct.WithDisplayName(displayName),
				),
			}

		case "data":
			// Handle data: URIs for inline content (not used in this file-only version)
			logger.Error("Data URIs not supported in file-only mode", "uri", req.ContentURI)
			return ExtractionTarget{}, fmt.Errorf("data URIs not supported in file-only mode")

		default:
			logger.Error("Unsupported URI scheme", "scheme", parsedURL.Scheme, "uri", req.ContentURI)
			return ExtractionTarget{}, fmt.Errorf("unsupported URI scheme: %s", parsedURL.Scheme)
		}

	} else {
		logger.Error("Invalid request: missing content URI")
		return ExtractionTarget{}, fmt.Errorf("ContentURI must be provided with file:// scheme")
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
	// Create assets based on input type
	assets := []unstruct.Asset{}
	// Handle modern URI-based content
	parsedURL, err := url.Parse(req.ContentURI)
	if err != nil {
		logger.Error("Failed to parse content URI for dry run", "uri", req.ContentURI, "error", err.Error())
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

		logger.Info("Processing file URI for dry run",
			"file_path", filePath,
			"original_uri", req.ContentURI,
		)

		displayName := req.DisplayName
		if displayName == "" {
			displayName = fmt.Sprintf("Dry Run - %s", filePath)
		}

		assets = append(assets,
			unstruct.NewFileAsset(
				client,
				filePath,
				unstruct.WithDisplayName(displayName),
			),
		)

	default:
		logger.Error("Unsupported URI scheme for dry run", "scheme", parsedURL.Scheme, "uri", req.ContentURI)
		return nil, fmt.Errorf("unsupported URI scheme: %s", parsedURL.Scheme)
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

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		logger.Error("Missing required environment variable", "variable", "GEMINI_API_KEY")
		return nil, fmt.Errorf("GEMINI_API_KEY not set")
	}

	logger.Info("Environment validated", "gemini_api_key_set", true)

	// Create GenAI client
	client, err := CreateDefaultGenAIClient(ctx, apiKey)
	if err != nil {
		logger.Error("Failed to create GenAI client", "error", err.Error())
		return nil, fmt.Errorf("failed to create GenAI client: %w", err)
	}

	logger.Info("GenAI client created successfully")

	// Create assets based on input type
	var assets []unstruct.Asset

	if req.Request.ContentURI != "" {
		// Handle modern URI-based content
		parsedURL, err := url.Parse(req.Request.ContentURI)
		if err != nil {
			logger.Error("Failed to parse content URI", "uri", req.Request.ContentURI, "error", err.Error())
			return nil, fmt.Errorf("invalid content URI: %w", err)
		}

		switch parsedURL.Scheme {
		case "file":
			// Extract file path from file:// URI
			filePath := parsedURL.Path
			logger.Info("Processing file URI for section extraction",
				"file_path", filePath,
				"original_uri", req.Request.ContentURI,
				"section", req.Section,
			)

			displayName := req.Request.DisplayName
			if displayName == "" {
				displayName = fmt.Sprintf("Temporal Section Processing - %s - %s", req.Section, filePath)
			}

			assets = []unstruct.Asset{
				unstruct.NewFileAsset(
					client,
					filePath,
					unstruct.WithDisplayName(displayName),
				),
			}

		default:
			logger.Error("Unsupported URI scheme for section extraction", "scheme", parsedURL.Scheme, "uri", req.Request.ContentURI)
			return nil, fmt.Errorf("unsupported URI scheme: %s", parsedURL.Scheme)
		}

	} else {
		logger.Error("Invalid request: missing content URI for section extraction")
		return nil, fmt.Errorf("ContentURI must be provided with file:// scheme")
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

	// Create a typed result map for the specific section
	var result map[string]interface{}

	switch req.Section {
	case "basic":
		// Create a basic info extractor
		type BasicInfo struct {
			Title   string `json:"title"`
			Author  string `json:"author"`
			DocType string `json:"doc_type"`
			Date    string `json:"date"`
		}

		basicExtractor := unstruct.New[BasicInfo](client, Prompts)
		basicResult, err := basicExtractor.Unstruct(
			ctx,
			assets,
			unstruct.WithModel(model),
			unstruct.WithTimeout(2*time.Minute),
		)
		if err != nil {
			logger.Error("Basic section extraction failed", "error", err.Error())
			return nil, fmt.Errorf("basic section extraction failed: %w", err)
		}

		result = map[string]interface{}{
			"title":    basicResult.Title,
			"author":   basicResult.Author,
			"doc_type": basicResult.DocType,
			"date":     basicResult.Date,
		}

	case "financial":
		// Create a financial info extractor
		type FinancialInfo struct {
			Budget   float64 `json:"budget"`
			Currency string  `json:"currency"`
		}

		financialExtractor := unstruct.New[FinancialInfo](client, Prompts)
		financialResult, err := financialExtractor.Unstruct(
			ctx,
			assets,
			unstruct.WithModel(model),
			unstruct.WithTimeout(2*time.Minute),
		)
		if err != nil {
			logger.Error("Financial section extraction failed", "error", err.Error())
			return nil, fmt.Errorf("financial section extraction failed: %w", err)
		}

		result = map[string]interface{}{
			"budget":   financialResult.Budget,
			"currency": financialResult.Currency,
		}

	case "project":
		// Create a project info extractor
		type ProjectInfo struct {
			Code      string `json:"code"`
			Status    string `json:"status"`
			TeamSize  int    `json:"team_size"`
			StartDate string `json:"start_date"`
			EndDate   string `json:"end_date"`
		}

		projectExtractor := unstruct.New[ProjectInfo](client, Prompts)
		projectResult, err := projectExtractor.Unstruct(
			ctx,
			assets,
			unstruct.WithModel(model),
			unstruct.WithTimeout(2*time.Minute),
		)
		if err != nil {
			logger.Error("Project section extraction failed", "error", err.Error())
			return nil, fmt.Errorf("project section extraction failed: %w", err)
		}

		result = map[string]interface{}{
			"code":       projectResult.Code,
			"status":     projectResult.Status,
			"team_size":  projectResult.TeamSize,
			"start_date": projectResult.StartDate,
			"end_date":   projectResult.EndDate,
		}

	case "contact":
		// Create a contact info extractor
		type ContactInfo struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			Phone string `json:"phone"`
		}

		contactExtractor := unstruct.New[ContactInfo](client, Prompts)
		contactResult, err := contactExtractor.Unstruct(
			ctx,
			assets,
			unstruct.WithModel(model),
			unstruct.WithTimeout(2*time.Minute),
		)
		if err != nil {
			logger.Error("Contact section extraction failed", "error", err.Error())
			return nil, fmt.Errorf("contact section extraction failed: %w", err)
		}

		result = map[string]interface{}{
			"name":  contactResult.Name,
			"email": contactResult.Email,
			"phone": contactResult.Phone,
		}

	default:
		logger.Error("Unsupported section for extraction", "section", req.Section)
		return nil, fmt.Errorf("unsupported section: %s", req.Section)
	}

	extractionDuration := time.Since(extractionStart)

	logger.Info("Section extraction completed successfully",
		"section", req.Section,
		"duration", extractionDuration,
		"fields_extracted", len(result),
	)

	return result, nil
}
