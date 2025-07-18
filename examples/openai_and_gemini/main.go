package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	unstruct "github.com/vivaneiona/genkit-unstruct"
	"google.golang.org/genai"
)

// Demonstrating different model selection strategies
type ExtractionRequest struct {
	// Basic company info - using Gemini Flash for speed and cost efficiency
	Company struct {
		Name         string `json:"name"`
		Industry     string `json:"industry"`
		Founded      int    `json:"founded"`
		Headquarters string `json:"headquarters"`
	} `json:"company" unstruct:"prompt/basic/model/gemini-1.5-flash"`

	// Financial analysis - using Gemini Pro for precision with structured data
	Financials struct {
		Revenue    float64 `json:"revenue"`
		Profit     float64 `json:"profit"`
		MarketCap  float64 `json:"market_cap"`
		GrowthRate float64 `json:"growth_rate"`
		RiskScore  int     `json:"risk_score"` // 1-10 scale
		Outlook    string  `json:"outlook"`    // Positive/Negative/Neutral
	} `json:"financials" unstruct:"prompt/financial/model/gemini-1.5-pro?temperature=0.1"`

	// Technical details - using Gemini Pro for technical understanding
	Technology struct {
		PrimaryTech   []string `json:"primary_tech"`
		CloudProvider string   `json:"cloud_provider"`
		Architecture  string   `json:"architecture"`
		Security      struct {
			Compliance []string `json:"compliance"`
			Encryption string   `json:"encryption"`
		} `json:"security"`
	} `json:"technology" unstruct:"prompt/technical/model/gemini-1.5-pro"`

	// Strategic analysis - using Gemini Pro with creative parameters for strategic thinking
	Strategy struct {
		MarketPosition    string   `json:"market_position"`
		Competitors       []string `json:"competitors"`
		Strengths         []string `json:"strengths"`
		Threats           []string `json:"threats"`
		Opportunities     []string `json:"opportunities"`
		StrategicPriority string   `json:"strategic_priority"`
	} `json:"strategy" unstruct:"prompt/strategy/model/gemini-1.5-pro?temperature=0.3&topK=40"`

	// Contact information - using Gemini Pro with strict parameters for accuracy
	Contact struct {
		CEO struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"ceo"`
		IR struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			Phone string `json:"phone"`
		} `json:"investor_relations"`
		Press struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"press"`
	} `json:"contact" unstruct:"prompt/contact/model/gemini-1.5-pro?temperature=0.1&topK=5"`

	// Competitive analysis - using latest Gemini model for advanced reasoning
	Competitive struct {
		MarketShare        float64  `json:"market_share"`
		CompetitiveRank    int      `json:"competitive_rank"`
		KeyDifferentiators []string `json:"key_differentiators"`
		MarketTrends       []string `json:"market_trends"`
	} `json:"competitive" unstruct:"prompt/competitive/model/gemini-2.0-flash-exp"`
}

func main() {
	ctx := context.Background()

	// Check for required environment variable
	geminiKey := os.Getenv("GEMINI_API_KEY")
	if geminiKey == "" {
		fmt.Println("Error: GEMINI_API_KEY environment variable is required")
		fmt.Println("Please set it with your Google AI API key:")
		fmt.Println("export GEMINI_API_KEY=your_gemini_key_here")
		os.Exit(1)
	}

	fmt.Println("Extraction Example")
	fmt.Println("==============================")
	fmt.Println("Demonstrating different Gemini models with varied parameters")
	fmt.Println("Note: For OpenAI integration, see the openai example")

	// Setup Gemini client
	fmt.Println("\nCreating Google GenAI client...")
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  geminiKey,
	})
	if err != nil {
		log.Fatal("Failed to create Gemini client:", err)
	}

	// Specialized prompts for different model capabilities and use cases
	prompts := unstruct.SimplePromptProvider{
		"basic": `Extract basic company information. Focus on factual, easily identifiable data.
This prompt uses Gemini Flash for speed and cost efficiency.
For founded field: Extract only the 4-digit year as a number (e.g., 2018, not "Founded in 2018").
Fields to extract: {{.Keys}}
Return JSON with exact field structure.`,

		"financial": `Perform detailed financial analysis with high precision. 
This prompt uses Gemini Pro with low temperature for accurate numeric extraction.
For risk_score: Evaluate on scale 1-10 (1=very low risk, 10=very high risk).
For outlook: Classify as "Positive", "Negative", or "Neutral".
Fields to extract: {{.Keys}}
Return JSON with exact field structure and numeric values.`,

		"technical": `Analyze technical infrastructure and architecture details. 
This prompt uses Gemini Pro for technical understanding.
Focus on technology stack, security measures, and technical capabilities.
For compliance: List applicable standards (SOC2, ISO27001, GDPR, etc.).
Fields to extract: {{.Keys}}
Return JSON with exact field structure.`,

		"strategy": `Conduct strategic business analysis with creative insights.
This prompt uses Gemini Pro with higher temperature for strategic thinking.
Evaluate market position, competitive landscape, and strategic opportunities.
Focus on: market dynamics, competitive advantages, strategic positioning.
Fields to extract: {{.Keys}}
Return JSON with exact field structure.`,

		"contact": `Extract precise contact information for key personnel.
This prompt uses Gemini Pro with very low temperature for maximum accuracy.
Focus on executive leadership and investor relations.
Ensure email formats are valid and phone numbers include country codes.
Fields to extract: {{.Keys}}
Return JSON with exact field structure.`,

		"competitive": `Advanced competitive analysis using latest AI capabilities.
This prompt uses Gemini 2.0 Flash Experimental for cutting-edge reasoning.
Analyze market position, competitive dynamics, and strategic implications.
Fields to extract: {{.Keys}}
Return JSON with exact field structure.`,
	}

	// Create extractor
	extractor := unstruct.New[ExtractionRequest](client, prompts)

	// Example 1: Comprehensive company analysis
	fmt.Println("\n=== Company Analysis ===")
	runCompanyAnalysis(ctx, extractor)

	// Example 2: Model strategy explanation
	fmt.Println("\n=== Model Selection Strategy ===")
	runModelStrategyAnalysis(ctx, extractor)

	// Example 3: Cost and performance comparison
	fmt.Println("\n=== Performance and Cost Analysis ===")
	runPerformanceAnalysis(ctx, extractor)
}

func runCompanyAnalysis(ctx context.Context, extractor *unstruct.Unstructor[ExtractionRequest]) {
	// Comprehensive company document
	companyDoc := `
TechFlow Solutions Inc. - Comprehensive Company Report 2024

COMPANY OVERVIEW
================
Company Name: TechFlow Solutions Inc.
Industry: Enterprise Software & Cloud Services
Founded: 2018
Headquarters: San Francisco, California, USA

FINANCIAL PERFORMANCE
====================
Revenue (2024): $125,800,000 USD
Net Profit: $18,750,000 USD
Market Capitalization: $2,100,000,000 USD
Year-over-Year Growth: 34.2%
Debt-to-Equity Ratio: 0.23
Operating Margin: 22.1%

TECHNOLOGY INFRASTRUCTURE
========================
Primary Technologies: Go, Python, React, Kubernetes, Docker
Cloud Provider: Amazon Web Services (AWS)
Architecture: Microservices with event-driven design
Database: PostgreSQL, Redis, MongoDB
Security Compliance: SOC2 Type II, ISO27001, GDPR
Encryption: AES-256 encryption at rest and in transit
API Gateway: Kong Gateway with OAuth 2.0

STRATEGIC POSITION
==================
Market Position: Leading provider of workflow automation software
Primary Competitors: Monday.com, Asana, ServiceNow, Atlassian
Key Strengths: 
- Advanced AI-powered automation
- Strong enterprise customer base (Fortune 500)
- Rapid international expansion
- High customer retention rate (98.5%)

Major Threats:
- Increasing competition from Microsoft and Google
- Economic downturn affecting enterprise spending
- Cybersecurity risks in cloud infrastructure

Growth Opportunities:
- Expansion into healthcare and finance verticals
- AI/ML workflow optimization
- International market penetration (Europe, Asia)
- Strategic acquisitions of complementary technologies

Strategic Priority: Focus on AI-powered automation features and international expansion

LEADERSHIP TEAM
===============
Chief Executive Officer:
  Name: Sarah Chen
  Email: sarah.chen@techflow.com

Investor Relations:
  Name: Michael Rodriguez
  Email: investor.relations@techflow.com
  Phone: +1-415-555-0199

Press Contact:
  Name: Jennifer Park
  Email: press@techflow.com

COMPETITIVE LANDSCAPE
===================
Market Share: 12.3% of enterprise workflow automation market
Competitive Ranking: #3 in the enterprise automation space
Key Differentiators:
- AI-first approach to workflow automation
- Superior integration capabilities
- Enterprise-grade security and compliance

Market Trends:
- Increasing demand for AI-powered automation
- Shift toward no-code/low-code solutions
- Growing emphasis on cybersecurity
- Remote work driving workflow digitization
`

	assets := []unstruct.Asset{
		unstruct.NewTextAsset(companyDoc),
	}

	start := time.Now()
	result, err := extractor.Unstruct(ctx, assets,
		unstruct.WithModel("gemini-1.5-flash"), // Default model
		unstruct.WithTimeout(60*time.Second),
		unstruct.WithRetry(3, 2*time.Second),
	)

	if err != nil {
		log.Printf("Error extracting company data: %v", err)
		return
	}

	duration := time.Since(start)

	// Display results organized by model and use case
	fmt.Printf("Extraction completed in %v\n\n", duration)

	fmt.Println("� BASIC INFO (Gemini Flash - Speed & Cost Optimized)")
	fmt.Printf("  Company: %s\n", result.Company.Name)
	fmt.Printf("  Industry: %s\n", result.Company.Industry)
	fmt.Printf("  Founded: %d\n", result.Company.Founded)
	fmt.Printf("  HQ: %s\n", result.Company.Headquarters)

	fmt.Println("\n💰 FINANCIAL ANALYSIS (Gemini Pro - Low Temperature for Precision)")
	fmt.Printf("  Revenue: $%.0f\n", result.Financials.Revenue)
	fmt.Printf("  Profit: $%.0f\n", result.Financials.Profit)
	fmt.Printf("  Market Cap: $%.0f\n", result.Financials.MarketCap)
	fmt.Printf("  Growth Rate: %.1f%%\n", result.Financials.GrowthRate)
	fmt.Printf("  Risk Score: %d/10\n", result.Financials.RiskScore)
	fmt.Printf("  Outlook: %s\n", result.Financials.Outlook)

	fmt.Println("\n⚙️ TECHNOLOGY (Gemini Pro - Technical Understanding)")
	fmt.Printf("  Primary Tech: %v\n", result.Technology.PrimaryTech)
	fmt.Printf("  Cloud: %s\n", result.Technology.CloudProvider)
	fmt.Printf("  Architecture: %s\n", result.Technology.Architecture)
	fmt.Printf("  Compliance: %v\n", result.Technology.Security.Compliance)
	fmt.Printf("  Encryption: %s\n", result.Technology.Security.Encryption)

	fmt.Println("\n🎯 STRATEGY (Gemini Pro - Higher Temperature for Creative Analysis)")
	fmt.Printf("  Market Position: %s\n", result.Strategy.MarketPosition)
	fmt.Printf("  Competitors: %v\n", result.Strategy.Competitors)
	fmt.Printf("  Strengths: %d identified\n", len(result.Strategy.Strengths))
	fmt.Printf("  Threats: %d identified\n", len(result.Strategy.Threats))
	fmt.Printf("  Opportunities: %d identified\n", len(result.Strategy.Opportunities))
	fmt.Printf("  Priority: %s\n", result.Strategy.StrategicPriority)

	fmt.Println("\n👥 CONTACTS (Gemini Pro - Strict Parameters for Accuracy)")
	fmt.Printf("  CEO: %s (%s)\n", result.Contact.CEO.Name, result.Contact.CEO.Email)
	fmt.Printf("  IR: %s (%s, %s)\n", result.Contact.IR.Name, result.Contact.IR.Email, result.Contact.IR.Phone)
	fmt.Printf("  Press: %s (%s)\n", result.Contact.Press.Name, result.Contact.Press.Email)

	fmt.Println("\n🏆 COMPETITIVE (Gemini 2.0 Flash Exp - Advanced Reasoning)")
	fmt.Printf("  Market Share: %.1f%%\n", result.Competitive.MarketShare)
	fmt.Printf("  Competitive Rank: #%d\n", result.Competitive.CompetitiveRank)
	fmt.Printf("  Differentiators: %d identified\n", len(result.Competitive.KeyDifferentiators))
	fmt.Printf("  Market Trends: %d identified\n", len(result.Competitive.MarketTrends))
}

func runModelStrategyAnalysis(ctx context.Context, extractor *unstruct.Unstructor[ExtractionRequest]) {
	sampleDoc := "TechFlow Solutions Inc. - Brief company overview for model analysis"
	assets := []unstruct.Asset{
		unstruct.NewTextAsset(sampleDoc),
	}

	plan, err := extractor.Explain(ctx, assets, unstruct.WithModel("gemini-1.5-flash"))
	if err != nil {
		log.Printf("Error generating execution plan: %v", err)
		return
	}

	fmt.Println("Model Distribution and Execution Plan:")
	fmt.Println(plan)
	fmt.Println("\nModel Selection Strategy:")
	fmt.Println("Gemini Flash      → Basic facts (optimize for speed & cost)")
	fmt.Println("Gemini Pro (T=0.1) → Financial data (optimize for precision)")
	fmt.Println("Gemini Pro        → Technical analysis (domain knowledge)")
	fmt.Println("Gemini Pro (T=0.3) → Strategic insights (creative reasoning)")
	fmt.Println("Gemini Pro (T=0.1) → Contact extraction (maximize accuracy)")
	fmt.Println("Gemini 2.0 Flash  → Competitive analysis (advanced reasoning)")

	fmt.Println("\nParameter Tuning:")
	fmt.Println("• Temperature 0.1: High precision for financial/contact data")
	fmt.Println("• Temperature 0.3: Balanced creativity for strategic analysis")
	fmt.Println("• TopK=5: Focused selection for contact accuracy")
	fmt.Println("• TopK=40: Broader selection for strategic creativity")
}

func runPerformanceAnalysis(ctx context.Context, extractor *unstruct.Unstructor[ExtractionRequest]) {
	sampleDoc := "Sample company document for performance estimation"
	assets := []unstruct.Asset{
		unstruct.NewTextAsset(sampleDoc),
	}

	stats, err := extractor.DryRun(ctx, assets, unstruct.WithModel("gemini-1.5-flash"))
	if err != nil {
		log.Printf("Error in performance analysis: %v", err)
		return
	}

	fmt.Printf("Performance Analysis:\n")
	fmt.Printf("• Total API calls: %d\n", stats.PromptCalls)
	fmt.Printf("• Input tokens: %d\n", stats.TotalInputTokens)
	fmt.Printf("• Output tokens: %d\n", stats.TotalOutputTokens)
	fmt.Printf("• Models used: %v\n", stats.ModelCalls)

	fmt.Println("\nOptimization Strategy:")
	fmt.Println("• Use Flash models for simple extraction tasks")
	fmt.Println("• Reserve Pro models for complex analysis")
	fmt.Println("• Tune temperature based on task requirements")
	fmt.Println("• Group related fields to minimize API calls")
	fmt.Println("• Consider using experimental models for cutting-edge features")

	fmt.Println("\nFuture Integration:")
	fmt.Println("• OpenAI models could be integrated for specific reasoning tasks")
	fmt.Println("• Claude models for creative writing and analysis")
	fmt.Println("• Anthropic models for ethical reasoning and safety")
	fmt.Println("• Mixed provider strategy for cost and capability optimization")
}
