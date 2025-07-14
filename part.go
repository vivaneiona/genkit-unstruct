package unstruct

// Part represents a part of a message (text, image, file, etc.)
type Part struct {
	Type     string
	Text     string
	Data     []byte
	FileURI  string // For file uploads
	MimeType string // For images and files
}

// NewTextPart creates a new text part
func NewTextPart(text string) *Part {
	return &Part{Type: "text", Text: text}
}

// NewImagePart creates a new image part with data and mime type
func NewImagePart(data []byte, mimeType string) *Part {
	return &Part{Type: "image", Data: data, MimeType: mimeType}
}

// NewFilePart creates a new file part that references an uploaded file URI
func NewFilePart(fileURI, mimeType string) *Part {
	return &Part{Type: "file", FileURI: fileURI, MimeType: mimeType}
}
