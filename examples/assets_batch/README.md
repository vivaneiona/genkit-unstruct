# Batch File Processing Example

This example demonstrates how **genkit‑unstruct** processes multiple files efficiently using `BatchFileAsset` to upload and analyze documents in batches through the Google GenAI Files API.

---

## What the example covers

* **BatchFileAsset integration** – process multiple files in a single operation with shared progress tracking
* **Advanced progress callbacks** – monitor file-by-file processing with detailed statistics
* **Metadata collection** – automatically gather file information including checksums, sizes, and upload timestamps
* **Auto cleanup** – optionally remove uploaded files from the Files API after processing
* **Mixed document types** – handle different file formats (.md, .txt, etc.) in a single batch
* **Performance metrics** – track processing time, throughput, and success rates
* **Cost estimation** – dry run capabilities for batch operations

---

## Key differences from single FileAsset

| Feature | FileAsset | BatchFileAsset |
|---------|-----------|----------------|
| **File Processing** | One file per asset | Multiple files per asset |
| **Progress Tracking** | Per-file callbacks | Batch-wide progress with per-file details |
| **Metadata** | Individual file metadata | Aggregated metadata + batch summary |
| **Performance** | Individual uploads | Optimized batch uploads |
| **Cost Estimation** | Single file estimates | Batch cost projections |
| **Cleanup** | Individual file cleanup | Batch cleanup operations |

---

## Data structures extracted

```go
type DocumentMetadata struct {
    Title       string `json:"title"       unstruct:"basic"`
    Description string `json:"description" unstruct:"basic"`
    Category    string `json:"category"    unstruct:"basic"`
    Author      string `json:"author"      unstruct:"person"`
    Date        string `json:"date"       unstruct:"basic"`
    Version     string `json:"version"    unstruct:"basic"`
}

type ProjectInfo struct {
    ProjectCode string  `json:"projectCode" unstruct:"project"`
    ProjectName string  `json:"projectName" unstruct:"project"`
    Budget      float64 `json:"budget"      unstruct:"project"`
    Currency    string  `json:"currency"    unstruct:"project"`
    StartDate   string  `json:"startDate"   unstruct:"project"`
    EndDate     string  `json:"endDate"     unstruct:"project"`
    Status      string  `json:"status"      unstruct:"project"`
    Priority    string  `json:"priority"    unstruct:"project"`
    ProjectLead string  `json:"projectLead" unstruct:"person"`
    TeamSize    int     `json:"teamSize"    unstruct:"project"`
}
```

---

## Example scenarios

### 1. Basic Batch Processing

```go
// Simple batch processing with progress tracking
batchAsset := unstruct.NewBatchFileAsset(
    client,
    markdownFiles,
    unstruct.WithBatchProgressCallback(progressCallback),
)
```

**Features:**
- Progress tracking for each file
- Basic batch upload and processing
- Simple error handling

### 2. Advanced Batch Processing

```go
// Advanced batch with metadata and cleanup
batchAsset := unstruct.NewBatchFileAsset(
    client,
    markdownFiles,
    unstruct.WithBatchProgressCallback(progressCallback),
    unstruct.WithBatchIncludeMetadata(true),
    unstruct.WithBatchAutoCleanup(true),
    unstruct.WithBatchRetentionDays(7),
)
```

**Features:**
- Detailed file metadata collection
- Automatic cleanup after processing
- Performance statistics and analytics
- File retention management

### 3. Mixed Document Types

```go
// Process different file types together
allFiles := append(markdownFiles, textFiles...)
batchAsset := unstruct.NewBatchFileAsset(
    client,
    allFiles,
    unstruct.WithBatchIncludeMetadata(true),
)
```

**Features:**
- Handle multiple file formats
- Unified processing pipeline
- Format-aware progress reporting

### 4. Cost Estimation

```go
// Estimate costs before processing
stats, err := u.DryRun(ctx, assets, unstruct.WithModel("gemini-1.5-flash"))
```

**Features:**
- Batch cost estimation with detailed breakdown
- Token usage predictions per file
- Actual cost calculations using current Gemini pricing
- Batch processing efficiency metrics
- Cost comparison vs individual file processing

---

## Progress callback features

The `ProgressCallback` function provides detailed tracking:

```go
progressCallback := func(processed, total int, currentFile string) {
    if currentFile != "" {
        // File currently being processed
        fmt.Printf("Processing %d/%d: %s\n", processed+1, total, currentFile)
    } else {
        // Batch processing complete
        fmt.Printf("Complete: %d/%d files processed\n", processed, total)
    }
}
```

**Callback data includes:**
- Current file being processed
- Overall progress (processed/total)
- File-specific information (size, type, etc.)
- Processing timestamps and performance metrics

---

## Processing statistics

The example tracks comprehensive performance metrics:

```go
type ProcessingStats struct {
    StartTime     time.Time     `json:"startTime"`
    EndTime       time.Time     `json:"endTime"`
    Duration      time.Duration `json:"duration"`
    TotalSize     int64         `json:"totalSize"`
    AverageSize   int64         `json:"averageSize"`
    SuccessRate   float64       `json:"successRate"`
}
```

**Metrics provided:**
- Total processing time
- Average time per file
- Data throughput (KB/sec)
- Success/failure rates
- File size statistics

---

## Setup and usage

1. **Demo mode (no API key required):**
   ```bash
   cd examples/assets_batch
   go run main.go
   ```
   This runs a demonstration with mock data showing all features.

2. **Live mode with API key:**
   ```bash
   export GEMINI_API_KEY="your-api-key"
   cd examples/assets_batch
   go run main.go
   ```

3. **Sample documents:**
   The example automatically creates sample markdown files if none exist in the `docs/` directory.

**Demo Mode Features:**
- Shows all batch processing scenarios with realistic sample data
- Demonstrates progress tracking and performance metrics
- Provides cost estimation examples
- No API calls or charges incurred
- Perfect for understanding functionality before using real API

---

## File structure

```
assets_batch/
├── main.go              # Main batch processing examples
├── go.mod               # Go module definition
├── README.md            # This documentation
├── templates/           # Stick templates for prompts
│   ├── basic.twig       # Basic document metadata
│   ├── person.twig      # Person/contact extraction
│   └── project.twig     # Project information extraction
└── docs/                # Sample documents (auto-created)
    ├── project-alpha.md           # Software project plan
    ├── meeting-minutes.md         # Team meeting notes  
    ├── product-requirements.md    # Mobile app requirements
    ├── tech-spec.md              # Technical architecture spec
    ├── research-report.md        # AI/ML research document
    └── user-guide.md             # API integration guide
```

---

## Output examples

### Basic Batch Processing
```
Found 3 markdown files to process
Processing file 1/3: project-alpha.md
Processing file 2/3: meeting-notes.md
Processing file 3/3: research-doc.md
Batch processing complete: 3/3 files processed

Batch processing completed in 2.34s

Extracted Document Metadata:
Title: Project Alpha - Development Plan
Description: A cutting-edge web application designed to streamline business processes
Category: Software Development
Author: John Smith
Date: January 15, 2024
Version: 2.1
```

### Advanced Processing Statistics
```
=== Processing Statistics ===
Total files processed: 3
Total processing time: 3.21s
Average time per file: 1.07s
Total data processed: 4.56 KB
Processing throughput: 1.42 KB/sec
✅ Batch cleanup completed
```

### Cost Estimation
```
=== Batch Cost Estimation ===
Total files: 6
Estimated prompt calls: 2
Estimated input tokens: 15,840
Estimated output tokens: 850
Average tokens per file: 2640.0 input, 141.7 output

=== Estimated Costs (Gemini 1.5 Flash) ===
Input tokens cost: $0.001188 (0.016 M tokens @ $0.075/M)
Output tokens cost: $0.000255 (0.001 M tokens @ $0.300/M)
Total estimated cost: $0.001443
Average cost per file: $0.000241

=== Batch Processing Benefits ===
Files processed in batch: 6
Prompt calls saved: 4 (vs 6 individual calls)
Processing efficiency: 66.7% reduction in API calls
```

---

## Best practices

1. **File organization:** Group related files for batch processing
2. **Progress monitoring:** Use callbacks for long-running batch operations
3. **Error handling:** Process individual file failures gracefully without stopping the batch
4. **Resource management:** Enable auto-cleanup for temporary uploads
5. **Cost control:** Use dry runs to estimate expenses before processing
6. **Performance tuning:** Monitor throughput and adjust batch sizes accordingly
7. **API key management:** Use demo mode for testing and development
8. **Graceful degradation:** Handle API failures with informative error messages

## Error handling

The example includes comprehensive error handling:

- **Invalid API keys:** Clear error messages with setup instructions
- **Network issues:** Graceful failure with retry suggestions  
- **File upload failures:** Individual file error logging without batch termination
- **Demo mode:** Fully functional demonstration without API requirements
- **Cost estimation:** Works even when file uploads fail

---

## Advanced features

- **Retention management:** Set custom retention periods for uploaded files
- **Metadata enrichment:** Collect detailed file information and checksums
- **Mixed formats:** Process different document types in unified batches
- **Performance analytics:** Track processing metrics and optimization opportunities
- **Failure recovery:** Handle individual file errors without stopping the batch
