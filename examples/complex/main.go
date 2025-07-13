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

type Company struct {
	Name    string `json:"name" unstruct:"company"`
	Address string `json:"address" unstruct:"company"`
}

// Project represents a complex nested data structure with model-specific extraction
type Project struct {
	ProjectColor string  `json:"projectColor" unstruct:"project"`
	ProjectMode  string  `json:"projectMode" unstruct:"project"`
	ProjectName  string  `json:"projectName" unstruct:"project"`
	CertIssuer   string  `json:"certIssuer"  unstruct:"cert"`
	Latitude     float64 `json:"lat" unstruct:"coords"`
	Longitude    float64 `json:"lon" unstruct:"coords"`

	// Nested structure with specific model for high-accuracy participant extraction
	Participant struct {
		Name    string `json:"name"    unstruct:"participant,gemini-1.5-pro"`
		Address string `json:"address" unstruct:"participant,gemini-1.5-pro"`
	} `json:"participant"`

	Company    Company   `unstruct:"company-info,gemini-1.5-pro"`
	Affiliated []Company `unstruct:"company-info,gemini-1.5-pro"`
}

func main() {
	// Set up colored logging with tint
	logger := slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:   slog.LevelDebug,
			NoColor: false,
		}),
	)
	slog.SetDefault(logger)

	ctx := context.Background()
	slog.Debug("Starting Complex Nested Structure example")

	// Check for required environment variable
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		slog.Debug("GEMINI_API_KEY not found in environment")
		fmt.Println("Error: GEMINI_API_KEY environment variable is required")
		fmt.Println("Please set it with your Google AI API key:")
		fmt.Println("export GEMINI_API_KEY=your_api_key_here")
		os.Exit(1)
	}
	slog.Debug("Found GEMINI_API_KEY", "key_length", len(apiKey))

	// Initialize Genkit with GoogleAI plugin
	fmt.Println("Initializing Genkit with GoogleAI plugin...")
	slog.Debug("Initializing Genkit with GoogleAI plugin")
	_, err := genkit.Init(ctx, genkit.WithPlugins(&googlegenai.GoogleAI{}))
	if err != nil {
		slog.Debug("Genkit initialization failed", "error", err)
		fmt.Printf("Failed to initialize Genkit: %v\n", err)
		os.Exit(1)
	}
	slog.Debug("Genkit initialization completed successfully")

	// Create Google AI client for unstract
	slog.Debug("Creating Google AI client", "backend", "GeminiAPI")
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  apiKey,
	})
	if err != nil {
		slog.Debug("Google AI client creation failed", "error", err)
		fmt.Printf("Failed to create Google AI client: %v\n", err)
		os.Exit(1)
	}
	slog.Debug("Google AI client created successfully")

	// Create Stick-based prompt provider from template files
	slog.Debug("Creating Stick prompt provider", "template_path", "./templates")
	promptProvider, err := unstruct.NewStickPromptProvider(
		unstruct.WithFS(os.DirFS("."), "templates"),
	)
	if err != nil {
		slog.Debug("Stick prompt provider creation failed", "error", err)
		fmt.Printf("Failed to create Stick prompt provider: %v\n", err)
		os.Exit(1)
	}
	slog.Debug("Stick prompt provider created successfully")

	// Build extractor with Stick templates
	slog.Debug("Creating Unstruct with Stick templates")
	uno := unstruct.New[Project](client, promptProvider)

	// Test documents with complex nested information
	testDocuments := []struct {
		name string
		text string
	}{
		{
			name: "Complex Project Document",
			text: `PROJECT NEXUS-2024 Status Report
			
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
			name: "Multi-Entity Project",
			text: `INFRASTRUCTURE PROJECT ALPHA-BETA-7
			
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
			name: "International Collaboration",
			text: `GLOBAL RESEARCH INITIATIVE CODE-NAME: HORIZON
			
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

	fmt.Println("🎯 Complex Nested Structure Extraction Demo")
	fmt.Println("============================================")
	fmt.Println("📋 Features demonstrated:")
	fmt.Println("   • Nested structure extraction")
	fmt.Println("   • Model-specific field processing")
	fmt.Println("   • Multi-level data hierarchy")
	fmt.Println("   • Mixed model optimization")
	fmt.Println()

	for i, doc := range testDocuments {
		fmt.Printf("\n📄 Document %d: %s\n", i+1, doc.name)
		fmt.Printf("Text: %s\n", doc.text[:200]+"...")
		fmt.Println("---")

		slog.Debug("Starting complex extraction", "document", doc.name, "default_model", "gemini-1.5-flash")

		assets := []unstruct.Asset{unstruct.NewTextAsset(doc.text)}
		out, err := uno.Unstruct(
			context.Background(),
			assets,
			unstruct.WithModel("gemini-1.5-flash"), // Default model, overridden by field-specific models
			unstruct.WithTimeout(45*time.Second),
			unstruct.WithRetry(2, 2*time.Second),
		)
		if err != nil {
			slog.Debug("Extraction failed", "document", doc.name, "error", err)
			fmt.Printf("❌ Failed to extract information: %v\n", err)
			continue
		}

		slog.Debug("Complex extraction completed successfully",
			"document", doc.name,
			"project_color", out.ProjectColor,
			"project_mode", out.ProjectMode,
			"project_name", out.ProjectName,
			"cert_issuer", out.CertIssuer,
			"participant_name", out.Participant.Name,
			"participant_address", out.Participant.Address,
		)

		fmt.Printf("✅ Extraction Results:\n")
		fmt.Printf("   🎨 Project Color: %s\n", out.ProjectColor)
		fmt.Printf("   📊 Project Mode: %s\n", out.ProjectMode)
		fmt.Printf("   📝 Project Name: %s\n", out.ProjectName)
		fmt.Printf("   📜 Certificate Issuer: %s\n", out.CertIssuer)
		fmt.Printf("   📍 Coordinates: %.6f, %.6f\n", out.Latitude, out.Longitude)
		fmt.Printf("   👤 Participant: %s at %s\n", out.Participant.Name, out.Participant.Address)
		fmt.Printf("   🏢 Company: %s at %s\n", out.Company.Name, out.Company.Address)
		fmt.Printf("   🔗 Affiliated Companies: %d\n", len(out.Affiliated))
		fmt.Printf("   🔍 Full result: %+v\n", *out)
	}

	fmt.Println("\n🎉 Complex nested structure extraction demo completed!")
	fmt.Println("💡 This example demonstrates:")
	fmt.Println("   • How different models can be used for different fields")
	fmt.Println("   • Extraction of deeply nested structures")
	fmt.Println("   • Optimization of model selection based on field complexity")
	slog.Debug("Complex example completed successfully")
}
