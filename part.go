package unstruct

// Part represents a part of a message (text, image, etc.)
type Part struct {
	Type string
	Text string
	Data []byte
}

// NewTextPart creates a new text part
func NewTextPart(text string) *Part {
	return &Part{Type: "text", Text: text}
}

// NewImagePart creates a new image part with data and mime type
func NewImagePart(data []byte, mimeType string) *Part {
	return &Part{Type: "image", Data: data}
}
