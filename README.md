# genkit-unstruct
[![Go Report Card](https://goreportcard.com/badge/github.com/vivaneiona/genkit-unstruct)](https://goreportcard.com/report/github.com/vivaneiona/genkit-unstruct)


Small, typed, concurrent: extract structured data from unstructured text (or images) with a single call, built on Google Genkit.

## Installation
```bash
go get github.com/vivaneiona/genkit-unstruct@latest
```

## Quick start

```go
package main

import (
	"context"
	"fmt"

	"github.com/vivaneiona/genkit-unstruct"
	"google.golang.org/genai"
)

type Person struct {
	Name string `json:"name"   unstruct:"basic"`
	Age  int    `json:"age"    unstruct:"basic"`
	City string `json:"city"   unstruct:"basic"`
}

func main() {
	ctx := context.Background()

	client, _ := genai.NewClient(ctx) // handle error in real code
	defer client.Close()

	u := unstruct.New[Person](client, prompts)       // prompts is any PromptProvider
	p, _ := u.UnstructFromText(ctx, "John, 25, NYC") // handle error

	fmt.Printf("%+v\n", p) // → {Name:John Age:25 City:NYC}
}
```

### Core ideas
- Tags drive extraction – every struct field declares its prompt/model in unstruct:"…".
- Batch by prompt – fields that share a prompt are requested once.
- Run in parallel – different prompt groups execute concurrently.
- Typed result – output is the same struct you defined; no manual parsing.

### API surface (unstable)

```go
// High-level
func (u *Unstructor[T]) Unstruct(ctx context.Context, assets []Asset, opts ...Option) (*T, error)
func (u *Unstructor[T]) UnstructFromText(ctx context.Context, doc string, opts ...Option) (*T, error)

// Planning
func (u *Unstructor[T]) DryRun(ctx context.Context, assets []Asset, opts ...Option) (*Stats, error)
```

#### Asset helpers

```go
unstruct.NewTextAsset(text)
unstruct.NewImageAsset(data, mime)
unstruct.NewMultiModalAsset(text, media...)
```

#### Tag grammar

```go
unstruct:"prompt"                 // use prompt, default model
unstruct:"prompt,gemini-flash"    // custom prompt + model
unstruct:"gemini-pro"             // inherit parent prompt, override model
```

Minimal feature list
- Stick-template and custom prompt providers
- Nested structs and slices
- Configurable concurrency, retry, back-off
- Optional cost and token accounting

Testing

```go
go test ./...
```
