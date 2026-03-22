package openapi

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/ramsesyok/oapi2wire/internal/model"
)

func loadFromYAML(t *testing.T, yaml string) *openapi3.T {
	t.Helper()
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData([]byte(yaml))
	if err != nil {
		t.Fatalf("failed to load OpenAPI: %v", err)
	}
	return doc
}

func TestBuildOperationIndex_Basic(t *testing.T) {
	yaml := `
openapi: 3.0.3
info:
  title: Test
  version: 1.0.0
paths:
  /users/{id}:
    get:
      operationId: getUser
      parameters:
        - in: path
          name: id
          required: true
          schema:
            type: string
        - in: query
          name: mode
          required: false
          schema:
            type: string
      responses:
        '200':
          description: OK
`
	doc := loadFromYAML(t, yaml)
	index, diags := BuildOperationIndex(doc)

	if len(diags) != 0 {
		t.Errorf("unexpected diagnostics: %v", diags)
	}

	op, ok := index["getUser"]
	if !ok {
		t.Fatal("expected getUser in index")
	}
	if op.Method != "GET" {
		t.Errorf("expected method GET, got %s", op.Method)
	}
	if op.Path != "/users/{id}" {
		t.Errorf("expected path /users/{id}, got %s", op.Path)
	}
	if len(op.PathParams) != 1 || op.PathParams[0] != "id" {
		t.Errorf("expected PathParams=[id], got %v", op.PathParams)
	}
	if len(op.QueryParams) != 1 || op.QueryParams[0].Name != "mode" || op.QueryParams[0].Required {
		t.Errorf("expected QueryParams=[{mode, false}], got %v", op.QueryParams)
	}
	if op.RepresentativeStatus != 200 {
		t.Errorf("expected status 200, got %d", op.RepresentativeStatus)
	}
}

func TestBuildOperationIndex_DuplicateOperationId(t *testing.T) {
	yaml := `
openapi: 3.0.3
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUser
      responses:
        '200':
          description: OK
  /users/{id}:
    get:
      operationId: getUser
      parameters:
        - in: path
          name: id
          required: true
          schema:
            type: string
      responses:
        '200':
          description: OK
`
	doc := loadFromYAML(t, yaml)
	_, diags := BuildOperationIndex(doc)

	if !model.HasErrors(diags) {
		t.Error("expected error diagnostic for duplicate operationId")
	}

	found := false
	for _, d := range diags {
		if d.Severity == model.SeverityError {
			found = true
		}
	}
	if !found {
		t.Error("no SeverityError diagnostic found")
	}
}

func TestBuildOperationIndex_PostWithJSONBody(t *testing.T) {
	yaml := `
openapi: 3.0.3
info:
  title: Test
  version: 1.0.0
paths:
  /users:
    post:
      operationId: createUser
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
      responses:
        '201':
          description: Created
`
	doc := loadFromYAML(t, yaml)
	index, diags := BuildOperationIndex(doc)

	if len(diags) != 0 {
		t.Errorf("unexpected diagnostics: %v", diags)
	}

	op, ok := index["createUser"]
	if !ok {
		t.Fatal("expected createUser in index")
	}
	if !op.HasJSONBody {
		t.Error("expected HasJSONBody=true")
	}
	if op.RepresentativeStatus != 201 {
		t.Errorf("expected status 201, got %d", op.RepresentativeStatus)
	}
}

func TestBuildOperationIndex_NoOperationId(t *testing.T) {
	yaml := `
openapi: 3.0.3
info:
  title: Test
  version: 1.0.0
paths:
  /health:
    get:
      responses:
        '200':
          description: OK
`
	doc := loadFromYAML(t, yaml)
	index, diags := BuildOperationIndex(doc)

	if len(diags) != 0 {
		t.Errorf("unexpected diagnostics: %v", diags)
	}
	if len(index) != 0 {
		t.Errorf("expected empty index, got %d entries", len(index))
	}
}
