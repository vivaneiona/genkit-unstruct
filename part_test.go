package unstruct

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewImagePart(t *testing.T) {
	data := []byte("fake image data")
	mimeType := "image/png"

	part := NewImagePart(data, mimeType)

	assert.Equal(t, "image", part.Type)
	assert.Equal(t, data, part.Data)
	assert.Equal(t, mimeType, part.MimeType)
	assert.Empty(t, part.Text)
}

func TestNewFilePart(t *testing.T) {
	uri := "gs://bucket/file.txt"
	mimeType := "text/plain"

	part := NewFilePart(uri, mimeType)

	assert.Equal(t, "file", part.Type)
	assert.Equal(t, uri, part.FileURI)
	assert.Equal(t, mimeType, part.MimeType)
	assert.Empty(t, part.Text)
	assert.Nil(t, part.Data)
}
