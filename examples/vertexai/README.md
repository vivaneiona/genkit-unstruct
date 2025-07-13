# VertexAI Plugin Example

This example demonstrates using genkit-unstract with Google Cloud VertexAI models through the Genkit VertexAI plugin.

## Setup

1. Install dependencies:
```bash
go mod tidy
```

2. Set required environment variables:
```bash
export GOOGLE_CLOUD_PROJECT=your_project_id
export GEMINI_API_KEY=your_gemini_api_key  # Required for this example as fallback
```

3. (Optional) For full VertexAI integration, set up Application Default Credentials:
```bash
gcloud auth application-default login
```

## Running

```bash
go run main.go
```

## What it does

This example:
1. Initializes Genkit with the VertexAI plugin
2. Analyzes a restaurant menu to extract comprehensive information
3. Groups fields by analysis type for efficient processing:
   - `restaurant`: Basic restaurant info (name, cuisine type)
   - `dishes`: Popular items analysis (best/worst dishes)
   - `pricing`: Financial analysis (average price, price range)
   - `dietary`: Dietary options (vegan, gluten-free)
   - `insights`: Recommendations and ratings

## Sample Output

```
=== MENU ANALYSIS RESULTS ===
Restaurant: Bella Vista Italian Restaurant
Cuisine Type: Italian

Dish Analysis:
  Popular Dish: Spaghetti Carbonara
  Most Expensive: Osso Buco
  Cheapest Item: Gelato (3 scoops)

Pricing:
  Average Price: $22.50
  Price Range: $7.95-42.95

Dietary Options:
  Vegan Options: true
  Gluten-Free Options: true

Insights:
  Recommended For: Romantic dinners and family celebrations
  Overall Rating: Authentic Italian cuisine in cozy atmosphere
```

## Models Used

This example demonstrates using VertexAI's latest models:
- `gemini-2.0-flash`: Latest high-performance model for complex analysis
- The Genkit VertexAI plugin provides access to Google Cloud's managed AI models

## Production Notes

For production use with VertexAI:
1. Configure proper authentication (Application Default Credentials)
2. Set appropriate project ID and region
3. Use VertexAI-specific client configuration instead of the fallback shown in this example
