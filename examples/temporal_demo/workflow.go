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
// This implementation uses TemporalRunner for deterministic parallel processing
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

	// Create TemporalRunner for deterministic parallel processing within workflow
	runner := NewTemporalRunner(ctx)
	logger.Info("Created TemporalRunner for parallel extraction coordination")

	// Define result collection struct
	var results struct {
		Basic     map[string]interface{}
		Financial map[string]interface{}
		Project   map[string]interface{}
		Contact   map[string]interface{}
	}

	// Schedule parallel extraction activities using TemporalRunner
	// Each extraction targets a specific section with appropriate model

	// Basic information extraction (fast model)
	runner.Go(func() error {
		var basicResult map[string]interface{}
		req := ExtractionRequest{
			Request: input.Request,
			Section: "basic",
			Model:   "gemini-1.5-flash",
		}
		err := workflow.ExecuteActivity(activityCtx, ExtractSectionActivity, req).Get(activityCtx, &basicResult)
		if err == nil {
			results.Basic = basicResult
			logger.Info("Basic extraction completed", "fields", len(basicResult))
		}
		return err
	})

	// Financial information extraction (precise model)
	runner.Go(func() error {
		var financialResult map[string]interface{}
		req := ExtractionRequest{
			Request: input.Request,
			Section: "financial",
			Model:   "gemini-1.5-pro",
		}
		err := workflow.ExecuteActivity(activityCtx, ExtractSectionActivity, req).Get(activityCtx, &financialResult)
		if err == nil {
			results.Financial = financialResult
			logger.Info("Financial extraction completed", "fields", len(financialResult))
		}
		return err
	})

	// Project information extraction (fast model)
	runner.Go(func() error {
		var projectResult map[string]interface{}
		req := ExtractionRequest{
			Request: input.Request,
			Section: "project",
			Model:   "gemini-1.5-flash",
		}
		err := workflow.ExecuteActivity(activityCtx, ExtractSectionActivity, req).Get(activityCtx, &projectResult)
		if err == nil {
			results.Project = projectResult
			logger.Info("Project extraction completed", "fields", len(projectResult))
		}
		return err
	})

	// Contact information extraction (precise model, low temperature)
	runner.Go(func() error {
		var contactResult map[string]interface{}
		req := ExtractionRequest{
			Request: input.Request,
			Section: "contact",
			Model:   "gemini-1.5-pro",
			Options: map[string]interface{}{"temperature": 0.2},
		}
		err := workflow.ExecuteActivity(activityCtx, ExtractSectionActivity, req).Get(activityCtx, &contactResult)
		if err == nil {
			results.Contact = contactResult
			logger.Info("Contact extraction completed", "fields", len(contactResult))
		}
		return err
	})

	// Wait for all parallel extractions to complete
	logger.Info("Waiting for parallel extractions to complete")
	err := runner.Wait()
	if err != nil {
		logger.Error("Parallel extraction failed",
			"error", err.Error(),
			"workflow_id", workflow.GetInfo(ctx).WorkflowExecution.ID,
		)
		return &WorkflowOutput{
			Success:   false,
			Timestamp: workflow.Now(ctx),
		}, fmt.Errorf("extraction failed: %w", err)
	}

	// Combine results into final structure
	var finalResult ExtractionTarget

	// Map basic fields
	if results.Basic != nil {
		if title, ok := results.Basic["title"].(string); ok {
			finalResult.Basic.Title = title
		}
		if author, ok := results.Basic["author"].(string); ok {
			finalResult.Basic.Author = author
		}
		if docType, ok := results.Basic["doc_type"].(string); ok {
			finalResult.Basic.DocType = docType
		}
		if date, ok := results.Basic["date"].(string); ok {
			finalResult.Basic.Date = date
		}
	}

	// Map financial fields
	if results.Financial != nil {
		if budget, ok := results.Financial["budget"].(float64); ok {
			finalResult.Financial.Budget = budget
		}
		if currency, ok := results.Financial["currency"].(string); ok {
			finalResult.Financial.Currency = currency
		}
	}

	// Map project fields
	if results.Project != nil {
		if code, ok := results.Project["code"].(string); ok {
			finalResult.Project.Code = code
		}
		if status, ok := results.Project["status"].(string); ok {
			finalResult.Project.Status = status
		}
		if teamSize, ok := results.Project["team_size"].(int); ok {
			finalResult.Project.TeamSize = teamSize
		}
		if startDate, ok := results.Project["start_date"].(string); ok {
			finalResult.Project.StartDate = startDate
		}
		if endDate, ok := results.Project["end_date"].(string); ok {
			finalResult.Project.EndDate = endDate
		}
	}

	// Map contact fields
	if results.Contact != nil {
		if name, ok := results.Contact["name"].(string); ok {
			finalResult.Contact.Name = name
		}
		if email, ok := results.Contact["email"].(string); ok {
			finalResult.Contact.Email = email
		}
		if phone, ok := results.Contact["phone"].(string); ok {
			finalResult.Contact.Phone = phone
		}
	}

	processingTime := workflow.Now(ctx).Sub(startTime)
	logger.Info("Document extraction completed successfully",
		"processing_time", processingTime,
		"model_calls", 4,
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
	if req.Request.TextContent != "" {
		logger.Info("Processing text content",
			"content_length", len(req.Request.TextContent),
		)
		assets = []unstruct.Asset{
			unstruct.NewTextAsset(req.Request.TextContent),
		}
	} else if req.Request.FilePath != "" {
		displayName := req.Request.DisplayName
		if displayName == "" {
			displayName = fmt.Sprintf("Temporal Section Processing - %s - %s", req.Section, req.Request.FilePath)
		}

		logger.Info("Processing file asset",
			"file_path", req.Request.FilePath,
			"display_name", displayName,
		)

		assets = []unstruct.Asset{
			unstruct.NewFileAsset(
				client,
				req.Request.FilePath,
				unstruct.WithDisplayName(displayName),
			),
		}
	} else {
		logger.Error("Invalid request: missing content")
		return nil, fmt.Errorf("either text_content or file_path must be provided")
	}

	logger.Info("Assets prepared for section extraction", "asset_count", len(assets))

	// Create a simple extraction for the specific section
	logger.Info("Starting section extraction",
		"section", req.Section,
		"model", req.Model,
	)

	extractionStart := time.Now()

	// Mock extraction result based on section
	var result map[string]interface{}
	switch req.Section {
	case "basic":
		result = map[string]interface{}{
			"title":    "Sample Document Title",
			"author":   "Sample Author",
			"doc_type": "Sample Type",
			"date":     "2024-01-01",
		}
	case "financial":
		result = map[string]interface{}{
			"budget":   50000.0,
			"currency": "USD",
		}
	case "project":
		result = map[string]interface{}{
			"code":       "PROJ-001",
			"status":     "Active",
			"team_size":  5,
			"start_date": "2024-01-01",
			"end_date":   "2024-12-31",
		}
	case "contact":
		result = map[string]interface{}{
			"name":  "John Doe",
			"email": "john.doe@example.com",
			"phone": "+1-555-0123",
		}
	default:
		result = map[string]interface{}{}
	}

	extractionDuration := time.Since(extractionStart)

	logger.Info("Section extraction completed successfully",
		"section", req.Section,
		"duration", extractionDuration,
		"fields_extracted", len(result),
	)

	return result, nil
}
