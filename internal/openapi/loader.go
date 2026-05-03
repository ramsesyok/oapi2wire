package openapi

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// Load reads and validates an OpenAPI 3.x file (YAML or JSON).
// Returns a fatal error if the file cannot be parsed.
func Load(path string) (*v3.Document, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("read OpenAPI %s: %w", path, err)
	}

	doc, err := libopenapi.NewDocument(data)
	if err != nil {
		return nil, fmt.Errorf("parse OpenAPI: %w", err)
	}

	model, err := doc.BuildV3Model()
	if err != nil {
		return nil, fmt.Errorf("build OpenAPI v3 model: %w", err)
	}

	return &model.Model, nil
}
