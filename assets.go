package unstruct

import (
	"context"
	"errors"
	"log/slog"
)

// Asset represents any kind of input that can be converted to messages for processing
type Asset interface {
	CreateMessages(ctx context.Context, log *slog.Logger) ([]*Message, error)
}

// TextAsset represents a text document
type TextAsset struct {
	Content string
}

// CreateMessages implements Asset for text content
func (t *TextAsset) CreateMessages(ctx context.Context, log *slog.Logger) ([]*Message, error) {
	if t.Content == "" {
		return nil, ErrEmptyDocument
	}
	return []*Message{NewUserMessage(NewTextPart(t.Content))}, nil
}

// ImageAsset represents an image document
type ImageAsset struct {
	Data     []byte
	MimeType string
}

// CreateMessages implements Asset for image content
func (i *ImageAsset) CreateMessages(ctx context.Context, log *slog.Logger) ([]*Message, error) {
	if len(i.Data) == 0 {
		return nil, errors.New("image data is empty")
	}
	part := &Part{
		Type: "image",
		Data: i.Data,
	}
	return []*Message{NewUserMessage(part)}, nil
}

// MultiModalAsset represents a combination of text and media
type MultiModalAsset struct {
	Text  string
	Media []*Part
}

// CreateMessages implements Asset for multi-modal content
func (m *MultiModalAsset) CreateMessages(ctx context.Context, log *slog.Logger) ([]*Message, error) {
	parts := []*Part{}
	if m.Text != "" {
		parts = append(parts, NewTextPart(m.Text))
	}
	parts = append(parts, m.Media...)

	if len(parts) == 0 {
		return nil, errors.New("no content provided")
	}

	return []*Message{NewUserMessage(parts...)}, nil
}

// FileAsset represents a file that can be read
type FileAsset struct {
	Path     string
	MimeType string
}

// CreateMessages implements Asset for file content
func (f *FileAsset) CreateMessages(ctx context.Context, log *slog.Logger) ([]*Message, error) {
	// This would typically read the file and determine how to process it
	// For now, this is a placeholder implementation
	return nil, errors.New("FileAsset not implemented - would read file and create appropriate messages")
}

// URLAsset represents content from a URL
type URLAsset struct {
	URL string
}

// CreateMessages implements Asset for URL content
func (u *URLAsset) CreateMessages(ctx context.Context, log *slog.Logger) ([]*Message, error) {
	// This would typically fetch content from the URL
	// For now, this is a placeholder implementation
	return nil, errors.New("URLAsset not implemented - would fetch URL content")
}

// NewTextAsset creates a new text asset
func NewTextAsset(content string) *TextAsset {
	return &TextAsset{Content: content}
}

// NewImageAsset creates a new image asset
func NewImageAsset(data []byte, mimeType string) *ImageAsset {
	return &ImageAsset{Data: data, MimeType: mimeType}
}

// NewMultiModalAsset creates a new multi-modal asset
func NewMultiModalAsset(text string, media ...*Part) *MultiModalAsset {
	return &MultiModalAsset{Text: text, Media: media}
}

func AssetsFrom(content string) []Asset {
	return []Asset{NewTextAsset(content)}
}
