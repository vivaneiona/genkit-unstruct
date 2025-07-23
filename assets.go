package unstruct

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"google.golang.org/genai"
)

// FileMetadata contains detailed information about a file
type FileMetadata struct {
	FilePath    string    `json:"filePath"`
	FileSize    int64     `json:"fileSize"`
	MIMEType    string    `json:"mimeType"`
	Checksum    string    `json:"checksum"`
	UploadedAt  time.Time `json:"uploadedAt"`
	ProcessedAt time.Time `json:"processedAt"`
	FileURI     string    `json:"fileURI"`     // Files API URI
	FileName    string    `json:"fileName"`    // Files API file name
	DisplayName string    `json:"displayName"` // Human-readable name
}

// ProgressCallback is called during batch file processing
type ProgressCallback func(processed, total int, currentFile string)

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
		Type:     "image",
		Data:     i.Data,
		MimeType: i.MimeType,
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

// FileAsset represents a file that can be uploaded to the Files API
type FileAsset struct {
	Path        string
	MimeType    string
	DisplayName string
	client      *genai.Client // injected for file upload

	// Advanced features
	AutoCleanup      bool
	IncludeMetadata  bool
	RetentionDays    int
	Metadata         *FileMetadata
	ProgressCallback ProgressCallback
	uploadedFile     *genai.File // cached uploaded file reference
}

// CreateMessages implements Asset for file content
func (f *FileAsset) CreateMessages(ctx context.Context, log *slog.Logger) ([]*Message, error) {
	if f.client == nil {
		return nil, fmt.Errorf("FileAsset requires a genai.Client for file uploads")
	}

	if f.Path == "" {
		return nil, fmt.Errorf("file path is empty")
	}

	// Check if file exists
	fileInfo, err := os.Stat(f.Path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", f.Path)
	}

	// Generate file metadata if requested
	var metadata *FileMetadata
	if f.IncludeMetadata || f.Metadata != nil {
		metadata, err = f.generateFileMetadata(fileInfo)
		if err != nil {
			log.Warn("Failed to generate file metadata", "error", err)
		}
	}

	// Use cached uploaded file if available
	var file *genai.File
	if f.uploadedFile != nil {
		file = f.uploadedFile
		log.Debug("Using cached uploaded file", "uri", file.URI)
	} else {
		// Determine MIME type if not provided
		mimeType := f.MimeType
		if mimeType == "" {
			mimeType = getMIMETypeFromPath(f.Path)
		}

		// Set display name if not provided
		displayName := f.DisplayName
		if displayName == "" {
			displayName = fmt.Sprintf("File Upload - %s", filepath.Base(f.Path))
		}

		// Call progress callback if provided
		if f.ProgressCallback != nil {
			f.ProgressCallback(0, 1, f.Path)
		}

		// Upload file to Files API
		log.Debug("Uploading file to Files API", "path", f.Path, "mime_type", mimeType)
		file, err = f.client.Files.UploadFromPath(ctx, f.Path, &genai.UploadFileConfig{
			MIMEType:    mimeType,
			DisplayName: displayName,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to upload file %s: %w", f.Path, err)
		}

		log.Debug("File uploaded", "uri", file.URI, "name", file.Name, "state", file.State, "mime_type", file.MIMEType)

		// Check if file is ready to use
		if file.State != "ACTIVE" {
			log.Warn("File is not in ACTIVE state", "state", file.State, "uri", file.URI)
			// For now, we'll continue anyway, but this might be the issue
		}

		// Cache the uploaded file reference
		f.uploadedFile = file

		// Update metadata with upload information
		if metadata != nil {
			metadata.FileURI = file.URI
			metadata.FileName = file.Name
			metadata.UploadedAt = time.Now()
			f.Metadata = metadata
		}

		// Call progress callback completion
		if f.ProgressCallback != nil {
			f.ProgressCallback(1, 1, f.Path)
		}

		log.Debug("File upload completed", "uri", file.URI, "name", file.Name, "state", file.State)
	}

	// Create message with file part that references the uploaded file
	var filePart *Part

	// Determine the MIME type for the file part
	mimeType := f.MimeType
	if mimeType == "" {
		mimeType = getMIMETypeFromPath(f.Path)
	}

	// Create file part that references the uploaded file URI
	filePart = NewFilePart(file.URI, mimeType)

	log.Debug("Created file part", "uri", file.URI, "mime_type", mimeType, "part_type", filePart.Type)

	// If metadata is included, also add a text part with file information
	if metadata != nil && f.IncludeMetadata {
		metadataText := fmt.Sprintf(`File Information:
- Original Path: %s
- Size: %d bytes
- MIME Type: %s
- Checksum (SHA256): %s
- Uploaded At: %s`,
			metadata.FilePath, metadata.FileSize, metadata.MIMEType,
			metadata.Checksum, metadata.UploadedAt.Format(time.RFC3339))

		// Return both file part and metadata text part
		return []*Message{NewUserMessage(filePart, NewTextPart(metadataText))}, nil
	}

	// Return just the file part
	return []*Message{NewUserMessage(filePart)}, nil
}

// generateFileMetadata creates detailed file metadata
func (f *FileAsset) generateFileMetadata(fileInfo os.FileInfo) (*FileMetadata, error) {
	// Calculate checksum
	file, err := os.Open(f.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for checksum: %w", err)
	}
	defer func() {
		_ = file.Close() // Best effort close, ignore error in defer
	}()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	checksum := fmt.Sprintf("%x", hasher.Sum(nil))

	// Determine MIME type
	mimeType := f.MimeType
	if mimeType == "" {
		mimeType = getMIMETypeFromPath(f.Path)
	}

	return &FileMetadata{
		FilePath:    f.Path,
		FileSize:    fileInfo.Size(),
		MIMEType:    mimeType,
		Checksum:    checksum,
		ProcessedAt: time.Now(),
		DisplayName: f.DisplayName,
	}, nil
}

// Cleanup removes the uploaded file from the Files API if AutoCleanup is enabled
func (f *FileAsset) Cleanup(ctx context.Context) error {
	if !f.AutoCleanup || f.uploadedFile == nil {
		return nil
	}

	// Note: The Files API doesn't provide a direct delete method in the current genai library
	// This would need to be implemented when the API becomes available
	return nil
}

// URLAsset represents content from a URL
type URLAsset struct {
	URL string
}

// CreateMessages implements Asset for URL content
func (u *URLAsset) CreateMessages(ctx context.Context, log *slog.Logger) ([]*Message, error) {
	resp, err := http.Get(u.URL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	html := string(body)
	if html == "" || resp.StatusCode == 404 {
		return nil, ErrEmptyDocument
	}
	return []*Message{NewUserMessage(NewTextPart(string(body)))}, nil
}

func NewURLAsset(url string) *URLAsset {
	return  &URLAsset{URL: string(url)}
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

// NewFileAsset creates a new file asset that will upload the file to the Files API
func NewFileAsset(client *genai.Client, path string, options ...func(*FileAsset)) *FileAsset {
	asset := &FileAsset{
		client: client,
		Path:   path,
	}
	for _, opt := range options {
		opt(asset)
	}
	return asset
}

// WithMimeType sets the MIME type for the file asset
func WithMimeType(mimeType string) func(*FileAsset) {
	return func(f *FileAsset) {
		f.MimeType = mimeType
	}
}

// WithDisplayName sets the display name for the file asset
func WithDisplayName(displayName string) func(*FileAsset) {
	return func(f *FileAsset) {
		f.DisplayName = displayName
	}
}

// WithMetadata sets the metadata for the file asset
func WithMetadata(metadata *FileMetadata) func(*FileAsset) {
	return func(f *FileAsset) {
		f.Metadata = metadata
	}
}

// WithProgressCallback sets the progress callback for batch processing
func WithProgressCallback(callback ProgressCallback) func(*FileAsset) {
	return func(f *FileAsset) {
		f.ProgressCallback = callback
	}
}

// WithAutoCleanup enables automatic cleanup of uploaded files after processing
func WithAutoCleanup(cleanup bool) func(*FileAsset) {
	return func(f *FileAsset) {
		f.AutoCleanup = cleanup
	}
}

// WithIncludeMetadata includes file metadata in the extraction
func WithIncludeMetadata(include bool) func(*FileAsset) {
	return func(f *FileAsset) {
		f.IncludeMetadata = include
	}
}

// WithRetentionDays sets the number of days to retain the file in Files API
func WithRetentionDays(days int) func(*FileAsset) {
	return func(f *FileAsset) {
		f.RetentionDays = days
	}
}

func AssetsFrom(content string) []Asset {
	return []Asset{NewTextAsset(content)}
}

// getMIMETypeFromPath returns the MIME type for a file by detecting it from content and extension
func getMIMETypeFromPath(path string) string {
	// Try to detect MIME type from file content if file exists
	mtype, err := mimetype.DetectFile(path)
	if err == nil {
		return mtype.String()
	}

	// Fallback to extension-based detection if file doesn't exist
	ext := filepath.Ext(path)
	switch ext {
	case ".txt":
		return "text/plain"
	case ".pdf":
		return "application/pdf"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".ppt":
		return "application/vnd.ms-powerpoint"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".csv":
		return "text/csv"
	case ".xml":
		return "application/xml"
	case ".json":
		return "application/json"
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".zip":
		return "application/zip"
	case ".tar":
		return "application/x-tar"
	case ".gz":
		return "application/gzip"
	default:
		return "application/octet-stream"
	}
}

// BatchFileAsset represents multiple files for batch processing
type BatchFileAsset struct {
	FilePaths        []string
	client           *genai.Client
	ProgressCallback ProgressCallback
	AutoCleanup      bool
	IncludeMetadata  bool
	RetentionDays    int
	uploadedFiles    []*genai.File // cached uploaded files
}

// CreateMessages implements Asset for batch file content
func (b *BatchFileAsset) CreateMessages(ctx context.Context, log *slog.Logger) ([]*Message, error) {
	if b.client == nil {
		return nil, fmt.Errorf("BatchFileAsset requires a genai.Client for file uploads")
	}

	if len(b.FilePaths) == 0 {
		return nil, fmt.Errorf("no file paths provided")
	}

	var messages []*Message
	var allMetadata []*FileMetadata

	for i, filePath := range b.FilePaths {
		// Call progress callback
		if b.ProgressCallback != nil {
			b.ProgressCallback(i, len(b.FilePaths), filePath)
		}

		// Create individual FileAsset
		fileAsset := &FileAsset{
			Path:            filePath,
			client:          b.client,
			AutoCleanup:     b.AutoCleanup,
			IncludeMetadata: b.IncludeMetadata,
			RetentionDays:   b.RetentionDays,
		}

		// Process the file
		fileMessages, err := fileAsset.CreateMessages(ctx, log)
		if err != nil {
			log.Warn("Failed to process file", "path", filePath, "error", err)
			continue
		}

		messages = append(messages, fileMessages...)

		// Collect metadata if available
		if fileAsset.Metadata != nil {
			allMetadata = append(allMetadata, fileAsset.Metadata)
		}

		// Store uploaded file reference
		if fileAsset.uploadedFile != nil {
			b.uploadedFiles = append(b.uploadedFiles, fileAsset.uploadedFile)
		}
	}

	// Call final progress callback
	if b.ProgressCallback != nil {
		b.ProgressCallback(len(b.FilePaths), len(b.FilePaths), "")
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no files were successfully processed")
	}

	// If metadata is included, add a summary message
	if b.IncludeMetadata && len(allMetadata) > 0 {
		summaryText := fmt.Sprintf("\nBatch Processing Summary:\n- Total files processed: %d\n- Total size: %d bytes\n",
			len(allMetadata), func() int64 {
				var total int64
				for _, md := range allMetadata {
					total += md.FileSize
				}
				return total
			}())

		messages = append(messages, NewUserMessage(NewTextPart(summaryText)))
	}

	return messages, nil
}

// Cleanup removes all uploaded files from the Files API if AutoCleanup is enabled
func (b *BatchFileAsset) Cleanup(ctx context.Context) error {
	if !b.AutoCleanup {
		return nil
	}

	for _, file := range b.uploadedFiles {
		// Note: The Files API doesn't provide a direct delete method in the current genai library
		// This would need to be implemented when the API becomes available
		_ = file
	}

	return nil
}

// NewBatchFileAsset creates a new batch file asset
func NewBatchFileAsset(client *genai.Client, filePaths []string, options ...func(*BatchFileAsset)) *BatchFileAsset {
	asset := &BatchFileAsset{
		client:    client,
		FilePaths: filePaths,
	}
	for _, opt := range options {
		opt(asset)
	}
	return asset
}

// WithBatchProgressCallback sets the progress callback for batch processing
func WithBatchProgressCallback(callback ProgressCallback) func(*BatchFileAsset) {
	return func(b *BatchFileAsset) {
		b.ProgressCallback = callback
	}
}

// WithBatchAutoCleanup enables automatic cleanup for batch processing
func WithBatchAutoCleanup(cleanup bool) func(*BatchFileAsset) {
	return func(b *BatchFileAsset) {
		b.AutoCleanup = cleanup
	}
}

// WithBatchIncludeMetadata includes metadata for batch processing
func WithBatchIncludeMetadata(include bool) func(*BatchFileAsset) {
	return func(b *BatchFileAsset) {
		b.IncludeMetadata = include
	}
}

// WithBatchRetentionDays sets retention days for batch processing
func WithBatchRetentionDays(days int) func(*BatchFileAsset) {
	return func(b *BatchFileAsset) {
		b.RetentionDays = days
	}
}
