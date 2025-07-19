# Temporal Unstruct Integration

Enterprise-grade document extraction with durable workflows, intelligent templating, and fault-tolerant parallel processing.

## Architecture

This integration demonstrates production-ready document processing using Temporal's deterministic execution model combined with unstruct's multi-model LLM extraction capabilities.

### Core Components

**Workflow Orchestration**
- `DocumentExtractionWorkflow` - Coordinates extraction pipeline with retry logic
- `ExtractDocumentDataActivity` - Executes parallel LLM calls with fault tolerance  
- `DryRunActivity` - Cost estimation and validation without API consumption

**Deterministic Execution Engine**
- `TemporalRunner` - Adapts unstruct's concurrency to Temporal's deterministic model
- Ensures replay consistency across worker failures and restarts
- Maintains parallel processing efficiency within workflow constraints

**Template-Driven Extraction**
```go
type ExtractionTarget struct {
    Metadata struct {
        Title, Author, Type, Date string
    } `json:"metadata" unstruct:"basic,gemini-1.5-flash"`
    
    Financial struct {
        Budget float64
        Currency string  
    } `json:"financial" unstruct:"financial,gemini-1.5-pro"`
    
    Project struct {
        Code, Status string
        Team int
        Timeline string
    } `json:"project" unstruct:"project,gemini-1.5-flash"`
    
    Contact struct {
        Name, Email, Phone string
    } `json:"contact" unstruct:"person,gemini-1.5-pro?temperature=0.2"`
}
```

## Temporal Runner Implementation

The `TemporalRunner` bridges unstruct's parallel processing with Temporal's deterministic execution requirements:

### Design Principles

**Deterministic Concurrency**
```go
func (r *TemporalRunner) Go(fn func() error) {
    r.wg.Add(1)
    workflow.Go(r.ctx, func(ctx workflow.Context) {
        defer r.wg.Done()
        if err := fn(); err != nil {
            r.mu.Lock()
            r.errors = append(r.errors, err)
            r.mu.Unlock()
        }
    })
}
```

**Key Features:**
- **Replay Safety**: Uses `workflow.Go` for deterministic async execution
- **Error Aggregation**: Thread-safe collection of failures across coroutines  
- **Synchronization**: WaitGroup ensures all operations complete before proceeding
- **Fail-Fast**: Returns first encountered error for immediate feedback

**Fault Tolerance**
- Workflow state persisted across worker restarts
- Automatic retry with exponential backoff
- Partial result recovery on activity failures
- Observable execution through Temporal UI
```

## Setup

## Quick Start

### Prerequisites
- Temporal Server (localhost:7233)
- Google AI API Key
- Go 1.24+

### Setup & Execution
```bash
# Environment
export GEMINI_API_KEY=your_api_key_here
just server dev-up

# Run (separate terminals)
just worker     # Start worker process
just start-demo # Execute workflow
```

## Usage Patterns

### Text Processing
```go
input := WorkflowInput{
    Request: DocumentRequest{
        TextContent: documentText,
        DisplayName: "Document Analysis",
    },
}
```

### File Processing  
```go
input := WorkflowInput{
    Request: DocumentRequest{
        FilePath: "docs/report.md", 
        DisplayName: "File Analysis",
    },
}
```

## Production Features

### Enterprise Capabilities
- **Durable Execution** - Workflows survive worker failures and restarts
- **Intelligent Retry** - Exponential backoff with configurable policies  
- **Full Observability** - Execution history and metrics via Temporal UI
- **Horizontal Scaling** - Multi-worker deployment with load balancing

### Model Optimization Strategy
```go
// Smart model allocation for cost/performance balance
Financial struct {
    Amount float64 `unstruct:"financial,gemini-1.5-pro"`     // High precision
} 
Metadata struct {
    Title string    `unstruct:"basic,gemini-1.5-flash"`       // Fast processing
}
Contact struct {
    Email string    `unstruct:"person,gemini-1.5-pro?temperature=0.1"` // Accuracy critical
}
```

## Operations

### Development Commands
```bash
# Server lifecycle
just server dev-up      # Start with web UI (port 8233)
just server dev-down    # Clean shutdown

# Development cycle  
just build && just test # Validate changes
just worker            # Start processing
just starter           # Execute workflow

# Maintenance
just tidy && just vet  # Dependencies and analysis
just build

```

### Monitoring & Observability
**Temporal UI**: http://localhost:8233
- Real-time workflow execution tracking
- Performance metrics and bottleneck analysis  
- Error diagnostics with full stack traces
- Historical execution patterns and trends

### Project Structure
```
temporal_demo/
├── workflow.go           # Core workflow definitions
├── temporal_runner.go    # Deterministic concurrency adapter  
├── prompts.go           # Twig template engine integration
├── templates/           # Extraction prompt templates
├── worker/             # Temporal worker process
└── starter/            # Workflow execution client
```

## Temporal Concurrency Adapter

The `temporal_runner.go` implements a critical integration component that adapts unstruct's parallel processing model to Temporal's deterministic execution environment.

### The Challenge

Unstruct normally uses Go's standard concurrency primitives (`sync.WaitGroup`, `sync.Mutex`) for parallel LLM API calls. However, Temporal workflows require **deterministic execution** to ensure reliable replay and fault tolerance. Standard Go concurrency is non-deterministic and breaks Temporal's guarantees.

### The Solution: TemporalRunner

```go
// Deterministic concurrency without sync primitives
type TemporalRunner struct {
    ctx       workflow.Context  // Temporal deterministic context
    firstErr  error            // Fail-fast error collection
    completed int              // Simple completion tracking
    total     int              // Operation count for synchronization
}
```

### Key Design Principles

1. **No Sync Primitives**: Eliminates `sync.Mutex` and `sync.WaitGroup` that break deterministic replay
2. **Simple State Tracking**: Uses basic counters instead of complex channel operations
3. **Workflow.Go Integration**: Leverages Temporal's deterministic coroutines for parallel execution
4. **Fail-Fast Semantics**: Returns first error immediately while allowing other operations to complete
5. **Zero Sleep Polling**: Uses `workflow.Sleep(ctx, 0)` for deterministic yielding between checks

### Why This Matters

- **Deterministic Replay**: Workflows can be replayed exactly during failures or debugging
- **Fault Tolerance**: Worker crashes don't lose processing state - workflows resume from checkpoints
- **Parallel Efficiency**: Multiple LLM API calls execute concurrently while maintaining consistency
- **Enterprise Ready**: Meets production requirements for reliability and observability

This adapter enables unstruct's powerful parallel processing within Temporal's enterprise-grade workflow orchestration, delivering both performance and reliability.

## Why Temporal + Unstruct?

This integration delivers **enterprise-grade document processing** with:

- **Fault Tolerance**: Zero data loss through durable execution
- **Scalability**: Horizontal scaling across worker clusters  
- **Observability**: Complete execution visibility and debugging
- **Cost Optimization**: Intelligent model selection and parallel processing
- **Production Ready**: Battle-tested reliability for critical workloads

Perfect for **high-volume document pipelines**, **compliance-critical processing**, and **mission-critical data extraction** where reliability and observability are paramount.
