package cases

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/ramsesyok/oapi2wire/internal/openapi"
)

const updateGolden = false

func loadTestOpenAPI(t *testing.T) *v3.Document {
	t.Helper()
	doc, err := openapi.Load(filepath.Join("..", "..", "testdata", "fixtures", "openapi.yaml"))
	if err != nil {
		t.Fatalf("loading openapi: %v", err)
	}
	return doc
}

func TestGenerateTemplate_Golden(t *testing.T) {
	doc := loadTestOpenAPI(t)
	ops, diags := openapi.BuildOperationIndex(doc)
	if len(diags) != 0 {
		t.Fatalf("unexpected diags: %v", diags)
	}

	result, err := GenerateTemplate(doc, ops)
	if err != nil {
		t.Fatalf("GenerateTemplate: %v", err)
	}

	goldenPath := filepath.Join("..", "..", "testdata", "golden", "init", "mock-cases.yaml")

	if updateGolden {
		if err := os.WriteFile(goldenPath, result.CaseYAML, 0o644); err != nil {
			t.Fatalf("writing golden: %v", err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("reading golden: %v", err)
	}

	gotYAML := normalizeNewlines(string(result.CaseYAML))
	wantYAML := normalizeNewlines(string(want))
	if gotYAML != wantYAML {
		t.Errorf("case YAML mismatch\ngot:\n%s\nwant:\n%s", result.CaseYAML, want)
	}
}

func normalizeNewlines(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

func TestGenerateTemplate_ResponseBodies(t *testing.T) {
	doc := loadTestOpenAPI(t)
	ops, _ := openapi.BuildOperationIndex(doc)

	result, err := GenerateTemplate(doc, ops)
	if err != nil {
		t.Fatalf("GenerateTemplate: %v", err)
	}

	// getUser should have id and name fields
	getUserBody, ok := result.ResponseFiles["getUser/getUser_default.json"]
	if !ok {
		t.Fatal("expected getUser response stub")
	}
	m, ok := getUserBody.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", getUserBody)
	}
	if m["id"] != "TODO" || m["name"] != "TODO" {
		t.Errorf("unexpected getUser body: %v", m)
	}

	// createUser should have result field
	createUserBody, ok := result.ResponseFiles["createUser/createUser_default.json"]
	if !ok {
		t.Fatal("expected createUser response stub")
	}
	cm, ok := createUserBody.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", createUserBody)
	}
	if cm["result"] != "TODO" {
		t.Errorf("unexpected createUser body: %v", cm)
	}
}
