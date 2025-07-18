package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/lmittmann/tint"
	unstruct "github.com/vivaneiona/genkit-unstruct"
	"google.golang.org/genai"
)

// Company represents a business entity with contact information
type Company struct {
	Name    string `json:"name" unstruct:"prompt/company"`
	Address string `json:"address" unstruct:"prompt/company"`
}

// Project represents a complex nested data structure with model-specific extraction
// This demonstrates the latest URL-style syntax for field-specific model selection
type Project struct {
	// Basic project information - processed with fast model for efficiency
	ProjectColor string `json:"projectColor" unstruct:"prompt/project/model/gemini-1.5-flash"`
	ProjectMode  string `json:"projectMode" unstruct:"prompt/project/model/gemini-1.5-flash"`
	ProjectName  string `json:"projectName" unstruct:"prompt/project/model/gemini-1.5-flash"`

	// Certificate information - requires precise extraction
	CertIssuer string `json:"certIssuer" unstruct:"prompt/cert/model/gemini-1.5-pro"`

	// Geographic coordinates - numerical precision important
	Latitude  float64 `json:"lat" unstruct:"prompt/coords/model/gemini-1.5-pro?temperature=0.1"`
	Longitude float64 `json:"lon" unstruct:"prompt/coords/model/gemini-1.5-pro?temperature=0.1"`

	// Nested structure with specific model for high-accuracy participant extraction
	Participant struct {
		Name    string `json:"name" unstruct:"prompt/participant/model/gemini-1.5-pro?temperature=0.2"`
		Address string `json:"address" unstruct:"prompt/participant/model/gemini-1.5-pro?temperature=0.2"`
	} `json:"participant"`

	// Company information with model selection
	Company    Company   `unstruct:"prompt/company-info/model/gemini-1.5-pro"`
	Affiliated []Company `unstruct:"prompt/company-info/model/gemini-1.5-pro"`
}

func main() {
	// Set up structured logging with professional styling
	logger := slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:      slog.LevelInfo,
			TimeFormat: time.Kitchen,
			NoColor:    false,
		}),
	)
	slog.SetDefault(logger)

	ctx := context.Background()
	slog.Info("Starting Complex Nested Structure Extraction Example")

	// Validate environment configuration
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		slog.Error("GEMINI_API_KEY environment variable is required")
		fmt.Println("Error: GEMINI_API_KEY environment variable is required")
		fmt.Println("Please set it with your Google AI API key:")
		fmt.Println("export GEMINI_API_KEY=your_api_key_here")
		os.Exit(1)
	}
	slog.Debug("Environment validation completed", "api_key_length", len(apiKey))

	// Initialize Genkit with GoogleAI plugin
	slog.Info("Initializing Genkit framework")
	_, err := genkit.Init(ctx, genkit.WithPlugins(&googlegenai.GoogleAI{}))
	if err != nil {
		slog.Error("Failed to initialize Genkit", "error", err)
		fmt.Printf("Failed to initialize Genkit: %v\n", err)
		os.Exit(1)
	}

	// Create Google AI client for unstract operations
	slog.Info("Creating Google AI client")
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  apiKey,
	})
	if err != nil {
		slog.Error("Failed to create Google AI client", "error", err)
		fmt.Printf("Failed to create Google AI client: %v\n", err)
		os.Exit(1)
	}

	// Create Stick-based prompt provider from template files
	slog.Info("Initializing template-based prompt provider", "template_directory", "./templates")
	promptProvider, err := unstruct.NewStickPromptProvider(
		unstruct.WithFS(os.DirFS("."), "templates"),
	)
	if err != nil {
		slog.Error("Failed to create Stick prompt provider", "error", err)
		fmt.Printf("Failed to create Stick prompt provider: %v\n", err)
		os.Exit(1)
	}

	// Build extractor with template-based prompts
	slog.Info("Creating extraction engine with advanced model selection")
	extractor := unstruct.New[Project](client, promptProvider)

	// Professional test documents showcasing complex nested information extraction
	testDocuments := []struct {
		name        string
		description string
		content     string
	}{
		{
			name:        "Advanced AI Research Project",
			description: "Complex project document with multiple nested entities and precise coordinates",
			content: `PROJECT NEXUS-2024 Status Report
			
			Primary Participant: Dr. Sarah Johnson, residing at 123 Tech Boulevard, Silicon Valley, CA 94025
			Project Owner: MegaCorp Industries, headquarters located at 789 Corporate Plaza, New York, NY 10001
			
			Station NEXUS-7. Certificate issued by "Quantum Certification Authority". 
			Coordinates: 37.4419°N, 122.1430°W (Palo Alto area).
			
			Project Details:
			- Description: Advanced AI research facility for next-generation quantum computing applications with machine learning integration
			- Budget: $15,750,000 USD allocated for fiscal year 2024
			- Timeline: Project commenced January 15, 2024, scheduled completion December 31, 2025
			
			All systems operational as of current date.`,
		},
		{
			name:        "Infrastructure Modernization Initiative",
			description: "Multi-entity project with international scope and complex organizational structure",
			content: `INFRASTRUCTURE PROJECT ALPHA-BETA-7
			
			Lead Researcher: Prof. Michael Chen, address: 456 University Drive, Boston, MA 02115
			Facility Owner: Boston Tech Consortium, main office at 321 Innovation Street, Cambridge, MA 02139
			
			Location Code: BTX-9901. Security Certificate by "Advanced Systems Verification Ltd"
			GPS Coordinates: 42.3601°N, 71.0589°W
			
			Project Information:
			- Description: Sustainable energy grid modernization with smart IoT sensor deployment across metropolitan area
			- Funding: €8,250,000 European Union grant
			- Schedule: Initiated March 1, 2024, target completion June 30, 2026
			
			Current status: Phase 2 implementation in progress.`,
		},
		{
			name:        "Global Climate Research Collaboration",
			description: "International research initiative with complex multi-national coordination",
			content: `GLOBAL RESEARCH INITIATIVE CODE-NAME: HORIZON
			
			Principal Investigator: Dr. Yuki Tanaka, residential address: 789 Sakura Avenue, Tokyo, Japan 150-0001
			Sponsoring Organization: International Science Foundation, registered office: 555 Research Park, Geneva, Switzerland 1201
			
			Facility ID: ISF-HORIZON-42. Certification Authority: "Global Standards Institute"
			Location: 35.6762°N, 139.6503°E (Tokyo Bay Research Complex)
			
			Project Specifications:
			- Description: Climate change mitigation through advanced atmospheric modeling and carbon capture technology development
			- Financial Allocation: ¥2,100,000,000 Japanese Yen research budget
			- Development Period: Launch date April 10, 2024, projected finish September 15, 2027
			
			Multi-national collaboration status: Active across 12 countries.`,
		},
	}

	fmt.Println("Complex Nested Structure Extraction Demo")
	fmt.Println("==========================================")
	fmt.Println("Features demonstrated:")
	fmt.Println("  • Nested structure extraction with URL-style syntax")
	fmt.Println("  • Model-specific field processing optimization")
	fmt.Println("  • Multi-level data hierarchy management")
	fmt.Println("  • Template-based prompt engineering")
	fmt.Println()

	for i, doc := range testDocuments {
		fmt.Printf("\nDocument %d: %s\n", i+1, doc.name)
		fmt.Printf("Description: %s\n", doc.description)
		fmt.Printf("Content preview: %s...\n", doc.content[:200])
		fmt.Println("---")

		slog.Info("Starting complex extraction",
			"document", doc.name,
			"default_model", "gemini-1.5-flash",
			"content_length", len(doc.content))

		assets := []unstruct.Asset{unstruct.NewTextAsset(doc.content)}
		result, err := extractor.Unstruct(
			context.Background(),
			assets,
			unstruct.WithModel("gemini-1.5-flash"), // Default model, overridden by field-specific models
			unstruct.WithTimeout(45*time.Second),
			unstruct.WithRetry(2, 2*time.Second),
		)
		if err != nil {
			slog.Error("Extraction failed", "document", doc.name, "error", err)
			fmt.Printf("Failed to extract information: %v\n", err)
			continue
		}

		slog.Info("Complex extraction completed successfully",
			"document", doc.name,
			"project_name", result.ProjectName,
			"participant_name", result.Participant.Name,
			"company_name", result.Company.Name,
			"coordinates", fmt.Sprintf("%.6f,%.6f", result.Latitude, result.Longitude))

		fmt.Printf("Extraction Results:\n")
		fmt.Printf("  Project Color: %s\n", result.ProjectColor)
		fmt.Printf("  Project Mode: %s\n", result.ProjectMode)
		fmt.Printf("  Project Name: %s\n", result.ProjectName)
		fmt.Printf("  Certificate Issuer: %s\n", result.CertIssuer)
		fmt.Printf("  Coordinates: %.6f, %.6f\n", result.Latitude, result.Longitude)
		fmt.Printf("  Participant: %s\n", result.Participant.Name)
		fmt.Printf("  Participant Address: %s\n", result.Participant.Address)
		fmt.Printf("  Company: %s\n", result.Company.Name)
		fmt.Printf("  Company Address: %s\n", result.Company.Address)
		fmt.Printf("  Affiliated Companies: %d\n", len(result.Affiliated))

		if len(result.Affiliated) > 0 {
			fmt.Printf("  Affiliated Company Details:\n")
			for j, affiliate := range result.Affiliated {
				fmt.Printf("    %d. %s at %s\n", j+1, affiliate.Name, affiliate.Address)
			}
		}
	}

	fmt.Println("\nComplex nested structure extraction demo completed successfully.")
	fmt.Println("Key achievements:")
	fmt.Println("  • Demonstrated URL-style syntax for model selection")
	fmt.Println("  • Extracted deeply nested structures with precision")
	fmt.Println("  • Optimized model selection based on field complexity")
	fmt.Println("  • Professional logging and error handling")
	slog.Info("Complex example completed successfully")
}
