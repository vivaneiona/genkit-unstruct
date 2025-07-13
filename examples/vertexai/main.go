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

// Fake prompt store
type filePrompts map[string]string

func (p filePrompts) GetPrompt(tag string, _ int) (string, error) {
	if s, ok := p[tag]; ok {
		slog.Debug("Found prompt", "tag", tag, "prompt_length", len(s))
		return s, nil
	}
	slog.Debug("Prompt not found", "tag", tag, "available_tags", getKeys(p))
	return "", fmt.Errorf("prompt %q not found", tag)
}

func getKeys(m filePrompts) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Example destination struct for menu analysis
type MenuAnalysis struct {
	// Basic menu info
	RestaurantName string `json:"restaurantName" unstruct:"restaurant"`
	CuisineType    string `json:"cuisineType" unstruct:"restaurant"`

	// Popular items
	PopularDish   string `json:"popularDish" unstruct:"dishes"`
	ExpensiveItem string `json:"expensiveItem" unstruct:"dishes"`
	CheapestItem  string `json:"cheapestItem" unstruct:"dishes"`

	// Price analysis
	AveragePrice float64 `json:"averagePrice" unstruct:"pricing"`
	PriceRange   string  `json:"priceRange" unstruct:"pricing"`

	// Additional insights
	HasVeganOptions bool   `json:"hasVeganOptions" unstruct:"dietary"`
	HasGlutenFree   bool   `json:"hasGlutenFree" unstruct:"dietary"`
	RecommendedFor  string `json:"recommendedFor" unstruct:"insights"`
	OverallRating   string `json:"overallRating" unstruct:"insights"`
}

func main() {
	// Set up colored logging with tint to see debug messages
	logger := slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:   slog.LevelDebug,
			NoColor: false,
		}),
	)
	slog.SetDefault(logger)

	ctx := context.Background()
	slog.Debug("Starting VertexAI example", "context", "background")

	// Check for required environment variables
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		slog.Debug("GOOGLE_CLOUD_PROJECT not found in environment")
		fmt.Println("Error: GOOGLE_CLOUD_PROJECT environment variable is required")
		fmt.Println("Please set it with your Google Cloud project ID:")
		fmt.Println("export GOOGLE_CLOUD_PROJECT=your_project_id")
		os.Exit(1)
	}
	slog.Debug("Found GOOGLE_CLOUD_PROJECT", "project_id", projectID)

	// Initialize Genkit with GoogleAI plugin (VertexAI alternative for demo)
	// In production, you would use VertexAI plugin with proper authentication
	fmt.Println("Initializing Genkit with GoogleAI plugin (VertexAI alternative)...")
	slog.Debug("Initializing Genkit with GoogleAI plugin")
	_, err := genkit.Init(ctx, genkit.WithPlugins(&googlegenai.GoogleAI{}))
	if err != nil {
		slog.Debug("Genkit initialization failed", "error", err)
		fmt.Printf("Failed to initialize Genkit: %v\n", err)
		os.Exit(1)
	}
	slog.Debug("Genkit initialization completed successfully")

	// Create client for VertexAI-style models (using GoogleAI for demo)
	// Note: In production, you would configure proper VertexAI authentication
	slog.Debug("Creating client for VertexAI-style models", "project", projectID)
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  os.Getenv("GEMINI_API_KEY"),
	})
	if err != nil {
		slog.Debug("VertexAI client creation failed", "error", err)
		fmt.Printf("Failed to create VertexAI client: %v\n", err)
		os.Exit(1)
	}
	slog.Debug("VertexAI client created successfully")

	// 1. Prompts for each extractor tag ⭐️
	slog.Debug("Setting up prompts", "prompt_count", 5)
	prompts := filePrompts{
		"restaurant": `Extract restaurant information from this menu text and return as JSON object with fields: {{.Keys}}. Return only a single JSON object, not an array. Text: {{.Document}}`,
		"dishes":     `Identify popular, expensive, and cheapest dishes from this menu and return as JSON with fields: {{.Keys}}. Return only valid JSON. Text: {{.Document}}`,
		"pricing":    `Analyze pricing information from this menu and return as JSON with fields: {{.Keys}}. Return numeric values for averagePrice as numbers, not strings. Example: {"averagePrice": 24.50, "priceRange": "$15-35"}. Text: {{.Document}}`,
		"dietary":    `Check for dietary options in this menu and return as JSON with fields: {{.Keys}}. Return boolean values for hasVeganOptions and hasGlutenFree. Example: {"hasVeganOptions": true, "hasGlutenFree": false}. Text: {{.Document}}`,
		"insights":   `Provide insights and recommendations about this restaurant based on the menu and return as JSON with fields: {{.Keys}}. Return only valid JSON. Text: {{.Document}}`,
	}
	slog.Debug("prompts configured",
		"restaurant_prompt_length", len(prompts["restaurant"]),
		"dishes_prompt_length", len(prompts["dishes"]),
		"pricing_prompt_length", len(prompts["pricing"]),
		"dietary_prompt_length", len(prompts["dietary"]),
		"insights_prompt_length", len(prompts["insights"]))

	// 2. Build unstructor
	slog.Debug("Creating Unstructor")
	uno := unstruct.New[MenuAnalysis](client, prompts)
	slog.Debug("Unstructor created successfully")

	// 3. Run extraction
	menuText := `
		Bella Vista Italian Restaurant
		
		APPETIZERS
		- Bruschetta al Pomodoro - $12.95 (V, GF available)
		- Antipasto Platter - $18.95
		- Calamari Fritti - $15.95
		
		PASTA & RISOTTO
		- Spaghetti Carbonara - $22.95 (Most Popular!)
		- Penne Arrabbiata - $19.95 (V, GF)
		- Truffle Risotto - $28.95
		- Lasagna della Casa - $24.95
		
		MAIN COURSES
		- Osso Buco - $42.95 (Chef's Special)
		- Chicken Parmigiana - $26.95
		- Grilled Salmon - $32.95 (GF)
		- Vegan Eggplant Stack - $23.95 (V, GF)
		
		DESSERTS
		- Tiramisu - $8.95
		- Gelato (3 scoops) - $7.95
		- Cannoli Siciliani - $9.95
		
		V = Vegan, GF = Gluten Free Available
		"Authentic Italian cuisine in a cozy atmosphere. Perfect for romantic dinners and family celebrations."
	`

	fmt.Printf("Analyzing menu from Bella Vista Italian Restaurant...\n")
	slog.Debug("Starting menu analysis", "menu_length", len(menuText), "model", "gemini-2.0-flash", "timeout", "45s")

	out, err := uno.Unstruct(
		context.Background(),
		unstruct.AssetsFrom(menuText),
		unstruct.WithModel("gemini-2.0-flash"), // Using VertexAI's latest Gemini model
		unstruct.WithTimeout(45*time.Second),
	)
	if err != nil {
		slog.Debug("Menu analysis failed", "error", err)
		fmt.Printf("Failed to analyze menu: %v\n", err)
		os.Exit(1)
	}
	slog.Debug("Menu analysis completed successfully",
		"restaurant_name", out.RestaurantName,
		"cuisine_type", out.CuisineType,
		"popular_dish", out.PopularDish,
		"average_price", out.AveragePrice,
		"has_vegan", out.HasVeganOptions,
		"has_gluten_free", out.HasGlutenFree)

	fmt.Printf("\n=== MENU ANALYSIS RESULTS ===\n")
	fmt.Printf("Restaurant: %s\n", out.RestaurantName)
	fmt.Printf("Cuisine Type: %s\n", out.CuisineType)
	fmt.Printf("\nDish Analysis:\n")
	fmt.Printf("  Popular Dish: %s\n", out.PopularDish)
	fmt.Printf("  Most Expensive: %s\n", out.ExpensiveItem)
	fmt.Printf("  Cheapest Item: %s\n", out.CheapestItem)
	fmt.Printf("\nPricing:\n")
	fmt.Printf("  Average Price: $%.2f\n", out.AveragePrice)
	fmt.Printf("  Price Range: %s\n", out.PriceRange)
	fmt.Printf("\nDietary Options:\n")
	fmt.Printf("  Vegan Options: %t\n", out.HasVeganOptions)
	fmt.Printf("  Gluten-Free Options: %t\n", out.HasGlutenFree)
	fmt.Printf("\nInsights:\n")
	fmt.Printf("  Recommended For: %s\n", out.RecommendedFor)
	fmt.Printf("  Overall Rating: %s\n", out.OverallRating)
	fmt.Printf("\nFull result: %+v\n", *out)
	slog.Debug("VertexAI example completed successfully")
}
