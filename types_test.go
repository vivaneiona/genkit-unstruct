package unstruct

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithOutputSchema(t *testing.T) {
	schema := `{"type": "object", "properties": {"name": {"type": "string"}}}`
	option := WithOutputSchema(schema)

	var opts Options
	option(&opts)

	assert.Equal(t, schema, opts.OutputSchemaJSON)
}

func TestWithStreaming(t *testing.T) {
	option := WithStreaming()

	var opts Options
	option(&opts)

	assert.True(t, opts.Streaming)
}

func TestWithRetry(t *testing.T) {
	maxRetries := 3
	backoff := 100 * time.Millisecond
	option := WithRetry(maxRetries, backoff)

	var opts Options
	option(&opts)

	assert.Equal(t, maxRetries, opts.MaxRetries)
	assert.Equal(t, backoff, opts.Backoff)
}

func TestWithParser(t *testing.T) {
	called := false
	parser := func(data []byte) (any, error) {
		called = true
		return string(data), nil
	}
	option := WithParser(parser)

	var opts Options
	option(&opts)

	assert.NotNil(t, opts.CustomParser)

	// Test the parser
	result, err := opts.CustomParser([]byte("test"))
	require.NoError(t, err)
	assert.Equal(t, "test", result)
	assert.True(t, called)
}

func TestWithGroupOption(t *testing.T) {
	groupName := "testGroup"
	prompt := "test-prompt"
	model := "test-model"
	option := WithGroup(groupName, prompt, model)

	var opts Options
	option(&opts)

	require.NotNil(t, opts.Groups)
	group := opts.Groups[groupName]
	assert.Equal(t, groupName, group.Name)
	assert.Equal(t, prompt, group.Prompt)
	assert.Equal(t, model, group.Model)
}

func TestWithFlattenGroups(t *testing.T) {
	option := WithFlattenGroups()

	var opts Options
	option(&opts)

	assert.True(t, opts.FlattenGroups)
}

func TestOptionsMultipleOptions(t *testing.T) {
	var opts Options

	// Apply multiple options
	WithModel("test-model")(&opts)
	WithTimeout(30 * time.Second)(&opts)
	WithFallbackPrompt("fallback")(&opts)
	WithFlattenGroups()(&opts)

	assert.Equal(t, "test-model", opts.Model)
	assert.Equal(t, 30*time.Second, opts.Timeout)
	assert.Equal(t, "fallback", opts.FallbackPrompt)
	assert.True(t, opts.FlattenGroups)
}

type TypesTestStruct struct {
	Name string `json:"name"`
}

func TestWithModelFor(t *testing.T) {
	model := "test-model"
	option := WithModelFor(model, TypesTestStruct{}, "Name")

	var opts Options
	option(&opts)

	require.NotNil(t, opts.FieldModels)
	assert.Equal(t, model, opts.FieldModels["TypesTestStruct.Name"])
}

func TestFieldModelMap(t *testing.T) {
	fieldModels := make(FieldModelMap)
	fieldModels["User.Name"] = "fast-model"
	fieldModels["User.Email"] = "precise-model"

	assert.Equal(t, "fast-model", fieldModels["User.Name"])
	assert.Equal(t, "precise-model", fieldModels["User.Email"])
	assert.Equal(t, "", fieldModels["NonExistent.Field"])
}

func TestGroupDefinition(t *testing.T) {
	group := GroupDefinition{
		Name:   "Financial",
		Prompt: "financial-prompt",
		Model:  "precise-model",
	}

	assert.Equal(t, "Financial", group.Name)
	assert.Equal(t, "financial-prompt", group.Prompt)
	assert.Equal(t, "precise-model", group.Model)
}
