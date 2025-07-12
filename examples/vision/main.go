package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/lmittmann/tint"
	unstruct "github.com/vivaneiona/genkit-unstruct"
	"google.golang.org/genai"
)

// DocumentData represents structured data that can be extracted from documents/images
type DocumentData struct {
	// Basic document information - use fast model for simple extraction
	DocumentType string `json:"documentType" unstruct:"document-type,gemini-1.5-flash"`
	DocumentDate string `json:"documentDate" unstruct:"document-type,gemini-2.5-pro"`

	// Financial information - use more powerful model for accuracy
	Invoice struct {
		InvoiceNumber string  `json:"invoiceNumber"`
		TotalAmount   float64 `json:"totalAmount"`
		Currency      string  `json:"currency"`
		DueDate       string  `json:"dueDate"`
	} `json:"invoice" unstruct:"financial,gemini-1.5-pro"`

	// Company information - use pro model for complex entity extraction
	Vendor struct {
		Name    string `json:"name"`
		Address string `json:"address"`
		Phone   string `json:"phone"`
		Email   string `json:"email"`
	} `json:"vendor" unstruct:"vendor-info,gemini-1.5-pro"`

	Customer struct {
		Name    string `json:"name"`
		Address string `json:"address"`
		Phone   string `json:"phone"`
		Email   string `json:"email"`
	} `json:"customer" unstruct:"customer-info,gemini-1.5-pro"`

	LineItems []LineItem `json:"lineItems" unstruct:"line-items,gemini-1.5-pro"`
}

type LineItem struct {
	Description string  `json:"description"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unitPrice"`
	Total       float64 `json:"total"`
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
	slog.Debug("Starting vision example", "context", "background")

	// Check for required environment variable
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		slog.Debug("GEMINI_API_KEY not found in environment")
		log.Fatal("GEMINI_API_KEY environment variable is required")
	}
	slog.Debug("Found GEMINI_API_KEY", "key_length", len(apiKey))

	// Create Google GenAI client directly
	fmt.Println("Creating Google GenAI client...")
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
		APIKey:  apiKey,
	})
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}

	// Find an image file to process
	imagePath := findImageFile()
	if imagePath == "" {
		log.Fatal("No image file found. Please place an image file (jpg, png, pdf) in this directory.")
	}

	fmt.Printf("Processing image: %s\n", imagePath)

	// Upload image to Files API
	fmt.Println("Uploading image to Files API...")
	file, err := client.Files.UploadFromPath(ctx, imagePath, &genai.UploadFileConfig{
		MIMEType:    getMIMEType(imagePath),
		DisplayName: fmt.Sprintf("Document Analysis - %s", filepath.Base(imagePath)),
	})
	if err != nil {
		log.Fatal("Failed to upload:", err)
	}
	fmt.Printf("Uploaded! File URI: %s\n", file.URI)

	// Create Stick-based prompt provider from template files
	fmt.Println("Setting up Stick template engine...")
	promptProvider, err := unstruct.NewStickPromptProvider(
		unstruct.WithFS(os.DirFS("."), "templates"),
		unstruct.WithVar("fileURI", file.URI),
	)
	if err != nil {
		log.Fatal("Failed to create Stick prompt provider:", err)
	}

	// Create unstructor
	fmt.Println("Creating unstructor...")
	u := unstruct.New[DocumentData](client, promptProvider)

	// Create input text that mentions the uploaded image
	visionInput := fmt.Sprintf(`Document uploaded to Gemini Files API: %s

Please analyze the uploaded document image and extract structured information.
The image has been uploaded to the Files API and should be processed for data extraction.

Instructions: Extract all relevant information accurately including:
- Document type and dates
- Financial information (amounts, invoice numbers, due dates)
- Vendor/company information (names, addresses, contact details)
- Customer information 
- Line items and product details

Pay attention to document structure and layout for accurate extraction.`, file.URI)

	fmt.Println("Extracting structured data from image using unstruct...")

	// Extract structured data with timeout and appropriate model
	result, err := u.Unstruct(ctx, visionInput,
		unstruct.WithModel("gemini-1.5-pro"), // Use Pro model for vision tasks
		unstruct.WithTimeout(60*time.Second), // Longer timeout for vision processing
	)
	if err != nil {
		log.Printf("DETAILED ERROR: %v", err)
		log.Fatal("Failed to extract data:", err)
	}

	// Display results
	displayResults(result)

	// Clean up uploaded file
	fmt.Println("\nCleaning up uploaded file...")
	_, err = client.Files.Delete(ctx, file.Name, nil)
	if err != nil {
		log.Printf("Warning: Failed to delete uploaded file: %v", err)
	} else {
		fmt.Println("Successfully cleaned up uploaded file")
	}
}

// findImageFile looks for common image file types in the current directory
func findImageFile() string {
	extensions := []string{".jpg", ".jpeg", ".png", ".pdf", ".tiff", ".bmp"}

	files, err := os.ReadDir(".")
	if err != nil {
		return ""
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		ext := filepath.Ext(file.Name())
		for _, validExt := range extensions {
			if ext == validExt {
				return file.Name()
			}
		}
	}

	return ""
}

// getMIMEType returns the MIME type for common image formats
func getMIMEType(filename string) string {
	ext := filepath.Ext(filename)
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".pdf":
		return "application/pdf"
	case ".tiff", ".tif":
		return "image/tiff"
	case ".bmp":
		return "image/bmp"
	default:
		return "image/jpeg" // fallback
	}
}

func displayResults(result *DocumentData) {
	fmt.Println("\n=== Extracted Document Data ===")
	fmt.Printf("Document Type: %s\n", result.DocumentType)
	fmt.Printf("Document Date: %s\n", result.DocumentDate)

	fmt.Printf("\nInvoice Information:\n")
	fmt.Printf("  Number: %s\n", result.Invoice.InvoiceNumber)
	fmt.Printf("  Amount: %.2f %s\n", result.Invoice.TotalAmount, result.Invoice.Currency)
	fmt.Printf("  Due Date: %s\n", result.Invoice.DueDate)

	fmt.Printf("\nVendor Information:\n")
	fmt.Printf("  Name: %s\n", result.Vendor.Name)
	fmt.Printf("  Address: %s\n", result.Vendor.Address)
	fmt.Printf("  Phone: %s\n", result.Vendor.Phone)
	fmt.Printf("  Email: %s\n", result.Vendor.Email)

	fmt.Printf("\nCustomer Information:\n")
	fmt.Printf("  Name: %s\n", result.Customer.Name)
	fmt.Printf("  Address: %s\n", result.Customer.Address)
	fmt.Printf("  Phone: %s\n", result.Customer.Phone)
	fmt.Printf("  Email: %s\n", result.Customer.Email)

	if len(result.LineItems) > 0 {
		fmt.Printf("\nLine Items (%d items):\n", len(result.LineItems))
		for i, item := range result.LineItems {
			fmt.Printf("  %d. %s - Qty: %d, Price: %.2f, Total: %.2f\n",
				i+1, item.Description, item.Quantity, item.UnitPrice, item.Total)
		}
	}
}
