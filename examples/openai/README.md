# OpenAI Plugin Example

This example demonstrates using genkit-unstract with OpenAI models through the Genkit OpenAI compatibility plugin.

## Setup

1. Install dependencies:
```bash
go mod tidy
```

2. Set required environment variables:
```bash
export OPENAI_API_KEY=your_openai_api_key_here
```

## Running

```bash
go run main.go
```

## What it does

This example:
1. Initializes Genkit with the OpenAI compatibility plugin
2. Extracts customer information from unstructured text using multiple prompts
3. Groups fields by prompt type for efficient processing:
   - `customer`: Personal information (name, email, phone)
   - `business`: Company information (company name, industry)
   - `financial`: Financial data (revenue, budget)

## Sample Output

```
Extraction Results:
Name: John Smith
Email: john.smith@techcorp.com
Phone: +1-555-0123
Company: TechCorp
Industry: technology
Revenue: $1500000.00
Budget: $250000.00
```

## Note

This example uses the OpenAI compatibility plugin for Genkit setup but falls back to Gemini for the actual unstruct operations. In a production setup, you would configure unstruct to work directly with OpenAI models through the appropriate client configuration.
