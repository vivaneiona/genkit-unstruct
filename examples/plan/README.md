# Execution Plan Example

This example demonstrates the EXPLAIN-style execution plan system for the Unstructor library, which provides PostgreSQL-inspired cost analysis for AI extraction workflows.

## Features Demonstrated

### 1. Basic Execution Plan with Abstract Costs
Shows the hierarchical structure of the extraction plan with relative cost estimates:

```
SchemaAnalysis (cost=44.0, tokens=50, fields=[Name Email Phone Experience Education Skills])
  ├─ PromptCall "NameExtractionPrompt" (model=gpt-3.5-turbo, cost=6.5, tokens=350, field=Name)
  ├─ PromptCall "EmailExtractionPrompt" (model=gpt-3.5-turbo, cost=6.4, tokens=340, field=Email)
  ├─ PromptCall "PhoneExtractionPrompt" (model=gpt-3.5-turbo, cost=6.5, tokens=350, field=Phone)
  ├─ PromptCall "ExperienceExtractionPrompt" (model=gpt-4, cost=6.5, tokens=350, field=Experience)
  ├─ PromptCall "EducationExtractionPrompt" (model=gpt-4, cost=6.5, tokens=350, field=Education)
  ├─ PromptCall "SkillsExtractionPrompt" (model=gpt-3.5-turbo, cost=6.5, tokens=350, field=Skills)
  └─ MergeFragments (cost=1.1, fields=[Name Email Phone Experience Education Skills])
```

### 2. Real Cost Estimation
Provides actual USD cost estimates based on current model pricing:

- GPT-3.5-turbo: ~$0.0007-0.0008 per field
- GPT-4: ~$0.0123 per field (17x more expensive)
- Total estimated cost: ~$0.028 for 6 fields

### 3. Multiple Output Formats

#### Text Format
Human-readable ASCII tree with PostgreSQL-style annotations

#### JSON Format
Structured data for programmatic consumption and tool integration

#### Graphviz DOT Format
Visual graph representation that can be rendered at https://dreampuf.github.io/GraphvizOnline/

#### HTML Format
Web-friendly representation for dashboard integration

### 4. Cost Analysis Features

- **Field-level breakdown**: Shows cost per extraction field
- **Model comparison**: Highlights cost differences between models
- **Token estimation**: Provides input/output token estimates
- **Total cost projection**: Aggregate cost across all operations

### 5. Schema-driven Planning

The planner automatically:
- Extracts fields from the provided schema
- Assigns appropriate models per field
- Estimates token counts based on field complexity
- Calculates hierarchical costs (children roll up to parents)

## Running the Example

```bash
# Using Go directly
go run main.go

# Using justfile
just run
```

## Key Components

### Plan Structure
- **SchemaAnalysis**: Root node that analyzes the extraction schema
- **PromptCall**: Individual LLM calls for each field extraction
- **MergeFragments**: Final step that combines all extracted data

### Cost Models
- **Abstract costs**: Relative complexity units for comparison
- **Real costs**: USD estimates based on token pricing
- **Token estimation**: Heuristic-based input/output size prediction

### Multiple Models
Different fields can use different models based on complexity:
- Simple fields (Name, Email, Phone): GPT-3.5-turbo
- Complex fields (Experience, Education): GPT-4

## Use Cases

1. **Cost Optimization**: Compare different model assignments before execution
2. **Budget Planning**: Estimate costs for large-scale extraction jobs
3. **Performance Analysis**: Identify bottlenecks in extraction pipelines
4. **Schema Validation**: Verify extraction plans before processing
5. **Tool Integration**: Export plans for external analysis tools

## Integration

This EXPLAIN system can be integrated with:
- Cost tracking dashboards
- CI/CD pipelines for cost validation
- Monitoring systems for actual vs. estimated costs
- Visualization tools using the DOT format
- Budget management systems using JSON output

The design follows PostgreSQL's EXPLAIN pattern, making it familiar to developers and easily extensible for new node types and cost models.
