# Assets Example - Template-Based Document Processing

This example demonstrates **template-based document extraction** from files using the genkit-unstruct library. It processes markdown documents from the `docs/` folder using sophisticated template-based prompts.

## ⚙︎ Features Demonstrated

### ▪︎ Template-Based Prompts
- **No hardcoded prompts** - All prompts loaded from `templates/*.twig` files
- **Twig template engine** for dynamic prompt generation  
- **Template variables** (`{{ Keys }}`, `{{ Document }}`, etc.)
- **Reusable and maintainable** prompt management

### ▪︎ Multi-Model Extraction Strategy
- **`basic` fields**: `gemini-1.5-flash` (fast, inherited by nested fields)
- **`financial` fields**: `gemini-1.5-pro` (precise for numerical data)
- **`contact` fields**: `gemini-1.5-pro` with `temperature=0.2, topK=40` (controlled extraction)
- **`projects` fields**: `gemini-1.5-flash` (efficient for array processing)

### ▪︎ Advanced Structure Features
- **Field inheritance** - Nested fields inherit parent model configuration
- **Model parameters** - Fine-tuned control with temperature and topK
- **Complex nested objects** - Contact information within organization
- **Array extraction** - Multiple projects with structured data
- **File processing** - Real document processing from markdown files

## ▪︎ Project Structure

```
assets/
├── main.go              # Main example code
├── utils.go             # Helper functions
├── templates/           # Template-based prompts
│   ├── basic.twig       # Company/organization info
│   ├── contact.twig     # Contact information
│   ├── financial.twig   # Revenue/budget data
│   └── projects.twig    # Project information
└── docs/                # Sample documents for processing
    ├── meeting-minutes.md
    ├── product-requirements.md
    └── tech-spec.md
```

## ⚙︎ Running the Example

1. **Set your API key**:
   ```bash
   export GEMINI_API_KEY=your_api_key_here
   ```

2. **Run the example**:
   ```bash
   go run *.go
   ```

## ▪︎ What It Does

The example automatically:

1. **Loads templates** from the `templates/` folder
2. **Finds markdown files** in the `docs/` directory
3. **Processes each document** using template-based prompts
4. **Extracts structured data** with different models per field type
5. **Displays results** with clean formatting using ▪︎ and ⚙︎ emojis

## ▪︎ Template System

### Template Variables Available:
- `{{ Keys }}` - List of fields to extract
- `{{ Document }}` - Document content (automatically injected)
- `{{ Version }}` - Template version
- `{{ Tag }}` - Template tag name

### Template Example (`basic.twig`):
```twig
Extract basic document information from this document.

Focus on identifying:
- Document title and type
- Company/organization name
- Author/creator information  

Extract the following fields: {% for key in Keys %}{{ key }}{% if not loop.last %}, {% endif %}{% endfor %}

Return JSON with this exact structure format:
{
  "organisation": {
    "name": "company_name",
    "docType": "document_type"
  }
}

Use the exact field names from the Keys list and ensure proper JSON formatting.
```

## ▪︎ Struct Tag Configuration

```go
type ExtractionRequest struct {
    Organisation struct {
        // Inherits from organisation struct tag (gemini-1.5-flash)
        Name         string  `json:"name"`
        DocumentType string  `json:"docType"`
        
        // Override with specific model for precision
        Revenue float64 `json:"revenue" unstruct:"prompt/financial/model/gemini-1.5-pro"`
        Budget  float64 `json:"budget" unstruct:"prompt/financial/model/gemini-1.5-pro"`
        
        // Model with parameters for controlled extraction
        Contact struct {
            Name  string `json:"name"`  // Inherits from contact struct
            Email string `json:"email"` // Inherits from contact struct  
            Phone string `json:"phone"` // Inherits from contact struct
        } `json:"contact" unstruct:"prompt/contact/model/gemini-1.5-pro?temperature=0.2&topK=40"`
        
        // Fast model for array processing
        Projects []Project `json:"projects" unstruct:"prompt/projects/model/gemini-1.5-flash"`
    } `json:"organisation" unstruct:"prompt/basic/model/gemini-1.5-flash"`
}
```

## ▪︎ Key Learning Points

1. **Template Separation**: Prompts are separated from code for better maintainability
2. **Model Optimization**: Different models for different data types optimize cost vs. quality
3. **Field Inheritance**: Nested fields automatically inherit parent configuration
4. **Parameter Control**: Fine-tune model behavior with temperature and topK
5. **File Processing**: Real document processing from markdown files
6. **Clean Output**: Consistent emoji usage (▪︎ and ⚙︎) for professional appearance

## ⚙︎ Customization

- **Add new templates**: Create `.twig` files in `templates/` folder
- **Modify extraction structure**: Update the struct tags and field types
- **Adjust model parameters**: Change temperature, topK, or model selection
- **Add new document types**: Extend the example with different file formats
- **Process different folders**: Change the `docs` folder path in the code

This example showcases production-ready document extraction focused on real file processing scenarios.
