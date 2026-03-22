package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ramsesyok/oapi2wire/internal/cases"
	"github.com/ramsesyok/oapi2wire/internal/model"
	"github.com/ramsesyok/oapi2wire/internal/openapi"
	"github.com/ramsesyok/oapi2wire/internal/output"
)

const updateGolden = false

func TestBuildMappings_Golden(t *testing.T) {
	fixturesDir := filepath.Join("..", "..", "testdata", "fixtures")
	goldenDir := filepath.Join("..", "..", "testdata", "golden", "build", "mappings")

	doc, err := openapi.Load(filepath.Join(fixturesDir, "openapi.yaml"))
	if err != nil {
		t.Fatalf("loading openapi: %v", err)
	}

	ops, diags := openapi.BuildOperationIndex(doc)
	if model.HasErrors(diags) {
		t.Fatalf("OpenAPI errors: %v", diags)
	}

	cf, err := cases.Load(filepath.Join(fixturesDir, "mock-cases.yaml"))
	if err != nil {
		t.Fatalf("loading cases: %v", err)
	}

	outDir := t.TempDir()
	if err := output.PrepareOutDir(output.WriterConfig{OutDir: outDir}); err != nil {
		t.Fatalf("prepare outdir: %v", err)
	}

	// Build case mappings
	for _, cs := range cf.Cases {
		op, ok := ops[cs.OperationID]
		if !ok {
			continue
		}
		mapping, err := BuildMapping(cs, op, cf.Defaults)
		if err != nil {
			t.Fatalf("BuildMapping: %v", err)
		}
		filename := MappingFileName(op.OperationID, cs.ID)
		if err := output.WriteMapping(outDir, filename, mapping); err != nil {
			t.Fatalf("WriteMapping: %v", err)
		}
	}

	// Auto-fallback
	for opID, op := range ops {
		if !NeedsAutoFallback(opID, cf.Cases) {
			continue
		}
		fb := BuildFallback(op)
		if err := output.WriteMapping(outDir, fb.MappingFileName, fb.Mapping); err != nil {
			t.Fatalf("WriteMapping fallback: %v", err)
		}
	}

	// Compare generated mappings with golden files
	entries, err := os.ReadDir(filepath.Join(outDir, "mappings"))
	if err != nil {
		t.Fatalf("reading mappings dir: %v", err)
	}

	for _, e := range entries {
		gotPath := filepath.Join(outDir, "mappings", e.Name())
		goldenPath := filepath.Join(goldenDir, e.Name())

		gotData, err := os.ReadFile(gotPath)
		if err != nil {
			t.Fatalf("reading %s: %v", gotPath, err)
		}

		if updateGolden {
			if err := os.WriteFile(goldenPath, gotData, 0o644); err != nil {
				t.Fatalf("writing golden %s: %v", goldenPath, err)
			}
			continue
		}

		wantData, err := os.ReadFile(goldenPath)
		if err != nil {
			t.Skipf("golden file not found for %s (run with updateGolden=true to generate)", e.Name())
		}

		// Compare as JSON to ignore whitespace differences
		var got, want interface{}
		if err := json.Unmarshal(gotData, &got); err != nil {
			t.Fatalf("parsing got JSON %s: %v", e.Name(), err)
		}
		if err := json.Unmarshal(wantData, &want); err != nil {
			t.Fatalf("parsing want JSON %s: %v", e.Name(), err)
		}

		gotNorm, _ := json.MarshalIndent(got, "", "  ")
		wantNorm, _ := json.MarshalIndent(want, "", "  ")
		if string(gotNorm) != string(wantNorm) {
			t.Errorf("mapping %s mismatch\ngot:\n%s\nwant:\n%s", e.Name(), gotNorm, wantNorm)
		}
	}
}
