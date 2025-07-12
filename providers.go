package unstruct

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/tyler-sommer/stick"
)

// → StickPromptProvider is fs-agnostic
type StickPromptProvider struct {
	env       *stick.Env
	templates map[string]string
}

// → Option pattern keeps the constructor flexible
type Option func(*StickPromptProvider) error

// WithFS loads every *.twig* file found under dir in the supplied FS.
func WithFS[F fs.FS](fsys F, dir string) Option {
	return func(p *StickPromptProvider) error {
		return fs.WalkDir(fsys, dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".twig") {
				return nil
			}
			content, readErr := fs.ReadFile(fsys, path)
			if readErr != nil {
				return fmt.Errorf("read %s: %w", path, readErr)
			}
			tag := strings.TrimSuffix(filepath.Base(path), ".twig")
			p.templates[tag] = string(content)
			return nil
		})
	}
}

// WithTemplates lets you inject an in-memory map.
func WithTemplates(m map[string]string) Option {
	return func(p *StickPromptProvider) error {
		for k, v := range m {
			p.templates[k] = v
		}
		return nil
	}
}

// NewStickPromptProvider builds a provider from any combination of options.
func NewStickPromptProvider(opts ...Option) (*StickPromptProvider, error) {
	p := &StickPromptProvider{
		env:       stick.New(nil),
		templates: make(map[string]string),
	}
	for _, opt := range opts {
		if err := opt(p); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// AddTemplate updates or inserts one template.
func (p *StickPromptProvider) AddTemplate(tag, tpl string) { p.templates[tag] = tpl }

// GetPrompt renders the template for the given tag.
func (p *StickPromptProvider) GetPrompt(tag string, version int) (string, error) {
	tpl, ok := p.templates[tag]
	if !ok {
		return "", fmt.Errorf("template %q not found", tag)
	}
	var out strings.Builder
	templateCtx := map[string]stick.Value{
		"version": version,
		"tag":     tag,
	}
	if err := p.env.Execute(tpl, &out, templateCtx); err != nil {
		return "", fmt.Errorf("execute %q: %w", tag, err)
	}
	return out.String(), nil
}

// → SimplePromptProvider stays untouched
type SimplePromptProvider map[string]string

func (s SimplePromptProvider) GetPrompt(tag string, version int) (string, error) {
	if tpl, ok := s[tag]; ok {
		return tpl, nil
	}
	return "", fmt.Errorf("prompt %q not found", tag)
}
