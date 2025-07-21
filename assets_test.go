package unstruct

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genai"
)

func TestNewTextAsset(t *testing.T) {
	content := "Test content"
	asset := NewTextAsset(content)

	assert.Equal(t, content, asset.Content)
}

func TestNewImageAsset(t *testing.T) {
	data := []byte("fake image data")
	mimeType := "image/png"
	asset := NewImageAsset(data, mimeType)

	assert.Equal(t, data, asset.Data)
	assert.Equal(t, mimeType, asset.MimeType)
}

func TestNewMultiModalAsset(t *testing.T) {
	text := "Test text"
	part1 := NewTextPart("part1")
	part2 := NewImagePart([]byte("image"), "image/png")

	asset := NewMultiModalAsset(text, part1, part2)

	assert.Equal(t, text, asset.Text)
	assert.Len(t, asset.Media, 2)
	assert.Equal(t, part1, asset.Media[0])
	assert.Equal(t, part2, asset.Media[1])
}

func TestTextAsset_CreateMessages(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	t.Run("valid content", func(t *testing.T) {
		asset := &TextAsset{Content: "Test content"}
		messages, err := asset.CreateMessages(ctx, logger)

		require.NoError(t, err)
		require.Len(t, messages, 1)
		assert.Equal(t, "user", messages[0].Role)
		require.Len(t, messages[0].Parts, 1)
		assert.Equal(t, "text", messages[0].Parts[0].Type)
		assert.Equal(t, "Test content", messages[0].Parts[0].Text)
	})

	t.Run("empty content", func(t *testing.T) {
		asset := &TextAsset{Content: ""}
		messages, err := asset.CreateMessages(ctx, logger)

		assert.Error(t, err)
		assert.Equal(t, ErrEmptyDocument, err)
		assert.Nil(t, messages)
	})
}

func TestImageAsset_CreateMessages(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	t.Run("valid image data", func(t *testing.T) {
		data := []byte("fake image data")
		asset := &ImageAsset{Data: data, MimeType: "image/png"}
		messages, err := asset.CreateMessages(ctx, logger)

		require.NoError(t, err)
		require.Len(t, messages, 1)
		assert.Equal(t, "user", messages[0].Role)
		require.Len(t, messages[0].Parts, 1)
		assert.Equal(t, "image", messages[0].Parts[0].Type)
		assert.Equal(t, data, messages[0].Parts[0].Data)
		assert.Equal(t, "image/png", messages[0].Parts[0].MimeType)
	})

	t.Run("empty image data", func(t *testing.T) {
		asset := &ImageAsset{Data: []byte{}, MimeType: "image/png"}
		messages, err := asset.CreateMessages(ctx, logger)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "image data is empty")
		assert.Nil(t, messages)
	})
}

func TestMultiModalAsset_CreateMessages(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	t.Run("text and media", func(t *testing.T) {
		text := "Test text"
		part := NewImagePart([]byte("image"), "image/png")
		asset := &MultiModalAsset{Text: text, Media: []*Part{part}}

		messages, err := asset.CreateMessages(ctx, logger)

		require.NoError(t, err)
		require.Len(t, messages, 1)
		assert.Equal(t, "user", messages[0].Role)
		require.Len(t, messages[0].Parts, 2)
		// First part should be text
		assert.Equal(t, "text", messages[0].Parts[0].Type)
		assert.Equal(t, text, messages[0].Parts[0].Text)
		// Second part should be image
		assert.Equal(t, "image", messages[0].Parts[1].Type)
	})

	t.Run("only media", func(t *testing.T) {
		part := NewImagePart([]byte("image"), "image/png")
		asset := &MultiModalAsset{Text: "", Media: []*Part{part}}

		messages, err := asset.CreateMessages(ctx, logger)

		require.NoError(t, err)
		require.Len(t, messages, 1)
		require.Len(t, messages[0].Parts, 1)
		assert.Equal(t, "image", messages[0].Parts[0].Type)
	})

	t.Run("no content", func(t *testing.T) {
		asset := &MultiModalAsset{Text: "", Media: []*Part{}}

		messages, err := asset.CreateMessages(ctx, logger)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no content provided")
		assert.Nil(t, messages)
	})
}

func TestNewFileAsset(t *testing.T) {
	// Create a mock client
	client := &genai.Client{}
	path := "/path/to/file.txt"

	t.Run("default options", func(t *testing.T) {
		asset := NewFileAsset(client, path)

		assert.Equal(t, client, asset.client)
		assert.Equal(t, path, asset.Path)
		assert.Empty(t, asset.MimeType)
		assert.Empty(t, asset.DisplayName)
	})

	t.Run("with options", func(t *testing.T) {
		mimeType := "text/plain"
		displayName := "test file"
		metadata := &FileMetadata{
			FilePath: path,
			FileSize: 1024,
		}

		asset := NewFileAsset(client, path,
			WithMimeType(mimeType),
			WithDisplayName(displayName),
			WithMetadata(metadata),
			WithAutoCleanup(true),
			WithIncludeMetadata(true),
			WithRetentionDays(30),
		)

		assert.Equal(t, mimeType, asset.MimeType)
		assert.Equal(t, displayName, asset.DisplayName)
		assert.Equal(t, metadata, asset.Metadata)
		assert.True(t, asset.AutoCleanup)
		assert.True(t, asset.IncludeMetadata)
		assert.Equal(t, 30, asset.RetentionDays)
	})
}

func TestWithMimeType(t *testing.T) {
	mimeType := "text/plain"
	option := WithMimeType(mimeType)

	asset := &FileAsset{}
	option(asset)

	assert.Equal(t, mimeType, asset.MimeType)
}

func TestWithDisplayName(t *testing.T) {
	displayName := "test file"
	option := WithDisplayName(displayName)

	asset := &FileAsset{}
	option(asset)

	assert.Equal(t, displayName, asset.DisplayName)
}

func TestWithMetadata(t *testing.T) {
	metadata := &FileMetadata{
		FilePath: "/test/path",
		FileSize: 1024,
	}
	option := WithMetadata(metadata)

	asset := &FileAsset{}
	option(asset)

	assert.Equal(t, metadata, asset.Metadata)
}

func TestWithProgressCallback(t *testing.T) {
	called := false
	callback := func(processed, total int, currentFile string) {
		called = true
	}
	option := WithProgressCallback(callback)

	asset := &FileAsset{}
	option(asset)

	// Test that callback is set (we can't easily test the function directly)
	assert.NotNil(t, asset.ProgressCallback)

	// Call the callback to verify it works
	asset.ProgressCallback(10, 100, "test.txt")
	assert.True(t, called)
}

func TestWithAutoCleanup(t *testing.T) {
	option := WithAutoCleanup(true)

	asset := &FileAsset{}
	option(asset)

	assert.True(t, asset.AutoCleanup)
}

func TestWithIncludeMetadata(t *testing.T) {
	option := WithIncludeMetadata(true)

	asset := &FileAsset{}
	option(asset)

	assert.True(t, asset.IncludeMetadata)
}

func TestWithRetentionDays(t *testing.T) {
	days := 30
	option := WithRetentionDays(days)

	asset := &FileAsset{}
	option(asset)

	assert.Equal(t, days, asset.RetentionDays)
}

func TestNewBatchFileAsset(t *testing.T) {
	client := &genai.Client{}
	paths := []string{"/path/1.txt", "/path/2.txt"}

	asset := NewBatchFileAsset(client, paths)

	assert.Equal(t, client, asset.client)
	assert.Equal(t, paths, asset.FilePaths)
}

func TestWithBatchProgressCallback(t *testing.T) {
	called := false
	callback := func(current, total int, path string) {
		called = true
	}
	option := WithBatchProgressCallback(callback)

	asset := &BatchFileAsset{}
	option(asset)

	assert.NotNil(t, asset.ProgressCallback)
	asset.ProgressCallback(1, 10, "/test")
	assert.True(t, called)
}

func TestWithBatchAutoCleanup(t *testing.T) {
	option := WithBatchAutoCleanup(true)

	asset := &BatchFileAsset{}
	option(asset)

	assert.True(t, asset.AutoCleanup)
}

func TestWithBatchIncludeMetadata(t *testing.T) {
	option := WithBatchIncludeMetadata(true)

	asset := &BatchFileAsset{}
	option(asset)

	assert.True(t, asset.IncludeMetadata)
}

func TestWithBatchRetentionDays(t *testing.T) {
	days := 30
	option := WithBatchRetentionDays(days)

	asset := &BatchFileAsset{}
	option(asset)

	assert.Equal(t, days, asset.RetentionDays)
}

func TestGetMIMETypeFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/path/file.txt", "text/plain"},
		{"/path/file.pdf", "application/pdf"},
		{"/path/file.jpg", "image/jpeg"},
		{"/path/file.png", "image/png"},
		{"/path/file.doc", "application/msword"},
		{"/path/file.unknown", "application/octet-stream"},
		{"/path/file", "application/octet-stream"},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			result := getMIMETypeFromPath(test.path)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestAssetsFrom(t *testing.T) {
	t.Run("string input", func(t *testing.T) {
		content := "test content"
		assets := AssetsFrom(content)

		require.Len(t, assets, 1)

		// Should be TextAsset from string
		textAsset, ok := assets[0].(*TextAsset)
		require.True(t, ok)
		assert.Equal(t, content, textAsset.Content)
	})
}
