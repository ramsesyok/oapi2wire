package openapi

import (
	"context"

	"github.com/getkin/kin-openapi/openapi3"
)

// Load reads and validates an OpenAPI 3.x file (YAML or JSON).
// Returns a fatal error if the file cannot be parsed.
func Load(path string) (*openapi3.T, error) {
	loader := openapi3.NewLoader()

	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, err
	}

	// Validate the document structure
	if err := doc.Validate(context.Background()); err != nil {
		return nil, err
	}

	return doc, nil
}
