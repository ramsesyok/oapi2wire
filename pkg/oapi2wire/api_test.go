package oapi2wire

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitValidateBuildWorkflow(t *testing.T) {
	fixturesDir := filepath.Join("..", "..", "testdata", "fixtures")
	openAPIPath := filepath.Join(fixturesDir, "openapi.yaml")
	responsesRoot := filepath.Join(fixturesDir, "mock-responses")
	casesPath := filepath.Join(fixturesDir, "mock-cases.yaml")

	initDir := t.TempDir()
	initResult, err := Init(InitOptions{
		OpenAPIPath:   openAPIPath,
		OutCasesPath:  filepath.Join(initDir, "mock-cases.yaml"),
		ResponsesRoot: filepath.Join(initDir, "responses"),
		Force:         true,
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if initResult.GeneratedCases != 2 {
		t.Fatalf("Init() GeneratedCases = %d, want 2", initResult.GeneratedCases)
	}
	if initResult.ResponseFilesWritten != 2 {
		t.Fatalf("Init() ResponseFilesWritten = %d, want 2", initResult.ResponseFilesWritten)
	}
	assertFileExists(t, filepath.Join(initDir, "mock-cases.yaml"))
	assertFileExists(t, filepath.Join(initDir, "responses", "getUser", "getUser_default.json"))

	validateResult, err := Validate(ValidateOptions{
		OpenAPIPath:   openAPIPath,
		CasesPath:     casesPath,
		ResponsesRoot: responsesRoot,
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if len(validateResult.Diagnostics) != 0 {
		t.Fatalf("Validate() diagnostics = %v, want none", validateResult.Diagnostics)
	}

	outDir := filepath.Join(t.TempDir(), "wiremock-out")
	buildResult, err := Build(BuildOptions{
		OpenAPIPath:   openAPIPath,
		CasesPath:     casesPath,
		ResponsesRoot: responsesRoot,
		OutDir:        outDir,
		Clean:         true,
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if buildResult.MappingsWritten != 4 {
		t.Fatalf("Build() MappingsWritten = %d, want 4", buildResult.MappingsWritten)
	}
	if buildResult.BodyFilesCopied != 2 {
		t.Fatalf("Build() BodyFilesCopied = %d, want 2", buildResult.BodyFilesCopied)
	}
	if buildResult.FallbacksWritten != 2 {
		t.Fatalf("Build() FallbacksWritten = %d, want 2", buildResult.FallbacksWritten)
	}
	assertFileExists(t, filepath.Join(outDir, "mappings", "getUser__getUser_detail_100.json"))
	assertFileExists(t, filepath.Join(outDir, "__files", "getUser", "getUser_detail_100.json"))
	assertFileExists(t, filepath.Join(outDir, "mappings", "_generated__fallback__getUser.json"))
}

func TestValidateMissingBodyFileCanFail(t *testing.T) {
	fixturesDir := filepath.Join("..", "..", "testdata", "fixtures")

	result, err := Validate(ValidateOptions{
		OpenAPIPath:            filepath.Join(fixturesDir, "openapi.yaml"),
		CasesPath:              filepath.Join(fixturesDir, "mock-cases.yaml"),
		ResponsesRoot:          filepath.Join(t.TempDir(), "missing-responses"),
		FailOnMissingBodyFile:  true,
		FailOnMissingOperation: true,
	})
	if err == nil {
		t.Fatal("Validate() error = nil, want missing body file error")
	}
	if len(result.Diagnostics) == 0 {
		t.Fatal("Validate() diagnostics = none, want missing body diagnostics")
	}

	foundError := false
	for _, diag := range result.Diagnostics {
		if diag.Severity == SeverityError && diag.Path != "" {
			foundError = true
			break
		}
	}
	if !foundError {
		t.Fatalf("Validate() diagnostics = %v, want at least one error with path", result.Diagnostics)
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file %s to exist: %v", path, err)
	}
}
