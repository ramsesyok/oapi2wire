package oapi2wire

import (
	"os"
	"path/filepath"
	"strings"
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

func TestInitWithTagsGeneratesOnlyMatchingOperations(t *testing.T) {
	fixturesDir := filepath.Join("..", "..", "testdata", "fixtures")
	initDir := t.TempDir()

	result, err := Init(InitOptions{
		OpenAPIPath:   filepath.Join(fixturesDir, "openapi.yaml"),
		OutCasesPath:  filepath.Join(initDir, "mock-cases.yaml"),
		ResponsesRoot: filepath.Join(initDir, "responses"),
		Force:         true,
		Tags:          []string{"pet"},
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if result.GeneratedCases != 1 {
		t.Fatalf("Init() GeneratedCases = %d, want 1", result.GeneratedCases)
	}
	if result.ResponseFilesWritten != 1 {
		t.Fatalf("Init() ResponseFilesWritten = %d, want 1", result.ResponseFilesWritten)
	}

	casesYAML := readFile(t, filepath.Join(initDir, "mock-cases.yaml"))
	if !strings.Contains(casesYAML, "operationId: getUser") {
		t.Fatalf("generated cases missing getUser:\n%s", casesYAML)
	}
	if strings.Contains(casesYAML, "operationId: createUser") {
		t.Fatalf("generated cases included non-pet operation:\n%s", casesYAML)
	}
	assertFileExists(t, filepath.Join(initDir, "responses", "getUser", "getUser_default.json"))
	assertFileNotExists(t, filepath.Join(initDir, "responses", "createUser", "createUser_default.json"))
}

func TestInitWithMultipleTagsGeneratesUnion(t *testing.T) {
	fixturesDir := filepath.Join("..", "..", "testdata", "fixtures")
	initDir := t.TempDir()

	result, err := Init(InitOptions{
		OpenAPIPath:   filepath.Join(fixturesDir, "openapi.yaml"),
		OutCasesPath:  filepath.Join(initDir, "mock-cases.yaml"),
		ResponsesRoot: filepath.Join(initDir, "responses"),
		Force:         true,
		Tags:          []string{"pet", "user"},
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if result.GeneratedCases != 2 {
		t.Fatalf("Init() GeneratedCases = %d, want 2", result.GeneratedCases)
	}

	casesYAML := readFile(t, filepath.Join(initDir, "mock-cases.yaml"))
	if !strings.Contains(casesYAML, "operationId: getUser") || !strings.Contains(casesYAML, "operationId: createUser") {
		t.Fatalf("generated cases should include pet and user operations:\n%s", casesYAML)
	}
}

func TestInitWithEmptyTagsKeepsAllOperations(t *testing.T) {
	fixturesDir := filepath.Join("..", "..", "testdata", "fixtures")
	initDir := t.TempDir()

	result, err := Init(InitOptions{
		OpenAPIPath:   filepath.Join(fixturesDir, "openapi.yaml"),
		OutCasesPath:  filepath.Join(initDir, "mock-cases.yaml"),
		ResponsesRoot: filepath.Join(initDir, "responses"),
		Force:         true,
		Tags:          []string{},
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if result.GeneratedCases != 2 {
		t.Fatalf("Init() GeneratedCases = %d, want 2", result.GeneratedCases)
	}
}

func TestValidateWithTagsDoesNotFailForFilteredOutOperations(t *testing.T) {
	fixturesDir := filepath.Join("..", "..", "testdata", "fixtures")

	result, err := Validate(ValidateOptions{
		OpenAPIPath:            filepath.Join(fixturesDir, "openapi.yaml"),
		CasesPath:              filepath.Join(fixturesDir, "mock-cases.yaml"),
		ResponsesRoot:          filepath.Join(fixturesDir, "mock-responses"),
		FailOnMissingOperation: true,
		Tags:                   []string{"pet"},
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	foundOutsideWarning := false
	for _, diag := range result.Diagnostics {
		if diag.Severity == SeverityError {
			t.Fatalf("Validate() returned error diagnostic for tag-filtered case: %v", result.Diagnostics)
		}
		if strings.Contains(diag.Message, "outside the tag filter") {
			foundOutsideWarning = true
		}
	}
	if !foundOutsideWarning {
		t.Fatalf("Validate() diagnostics = %v, want outside tag filter warning", result.Diagnostics)
	}
}

func TestBuildWithTagsGeneratesOnlyMatchingMappingsAndFallbacks(t *testing.T) {
	fixturesDir := filepath.Join("..", "..", "testdata", "fixtures")
	outDir := filepath.Join(t.TempDir(), "wiremock-out")

	result, err := Build(BuildOptions{
		OpenAPIPath:   filepath.Join(fixturesDir, "openapi.yaml"),
		CasesPath:     filepath.Join(fixturesDir, "mock-cases.yaml"),
		ResponsesRoot: filepath.Join(fixturesDir, "mock-responses"),
		OutDir:        outDir,
		Clean:         true,
		Tags:          []string{"pet"},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if result.MappingsWritten != 2 {
		t.Fatalf("Build() MappingsWritten = %d, want 2", result.MappingsWritten)
	}
	if result.BodyFilesCopied != 1 {
		t.Fatalf("Build() BodyFilesCopied = %d, want 1", result.BodyFilesCopied)
	}
	if result.FallbacksWritten != 1 {
		t.Fatalf("Build() FallbacksWritten = %d, want 1", result.FallbacksWritten)
	}
	assertFileExists(t, filepath.Join(outDir, "mappings", "getUser__getUser_detail_100.json"))
	assertFileExists(t, filepath.Join(outDir, "mappings", "_generated__fallback__getUser.json"))
	assertFileNotExists(t, filepath.Join(outDir, "mappings", "createUser__createUser_admin_error.json"))
	assertFileNotExists(t, filepath.Join(outDir, "mappings", "_generated__fallback__createUser.json"))
	assertFileNotExists(t, filepath.Join(outDir, "__files", "createUser", "createUser_admin_error.json"))
}

func TestBuildWithTagsAndNoAutoFallback(t *testing.T) {
	fixturesDir := filepath.Join("..", "..", "testdata", "fixtures")
	outDir := filepath.Join(t.TempDir(), "wiremock-out")

	result, err := Build(BuildOptions{
		OpenAPIPath:    filepath.Join(fixturesDir, "openapi.yaml"),
		CasesPath:      filepath.Join(fixturesDir, "mock-cases.yaml"),
		ResponsesRoot:  filepath.Join(fixturesDir, "mock-responses"),
		OutDir:         outDir,
		Clean:          true,
		NoAutoFallback: true,
		Tags:           []string{"pet"},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if result.MappingsWritten != 1 {
		t.Fatalf("Build() MappingsWritten = %d, want 1", result.MappingsWritten)
	}
	if result.FallbacksWritten != 0 {
		t.Fatalf("Build() FallbacksWritten = %d, want 0", result.FallbacksWritten)
	}
	assertFileExists(t, filepath.Join(outDir, "mappings", "getUser__getUser_detail_100.json"))
	assertFileNotExists(t, filepath.Join(outDir, "mappings", "_generated__fallback__getUser.json"))
}

func TestInitWithTagsNoMatchReturnsDiagnostic(t *testing.T) {
	fixturesDir := filepath.Join("..", "..", "testdata", "fixtures")
	initDir := t.TempDir()

	result, err := Init(InitOptions{
		OpenAPIPath:   filepath.Join(fixturesDir, "openapi.yaml"),
		OutCasesPath:  filepath.Join(initDir, "mock-cases.yaml"),
		ResponsesRoot: filepath.Join(initDir, "responses"),
		Force:         true,
		Tags:          []string{"store"},
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if result.GeneratedCases != 0 {
		t.Fatalf("Init() GeneratedCases = %d, want 0", result.GeneratedCases)
	}

	found := false
	for _, diag := range result.Diagnostics {
		if diag.Severity == SeverityWarning && strings.Contains(diag.Message, "no operations matched tags: store") {
			found = true
		}
	}
	if !found {
		t.Fatalf("Init() diagnostics = %v, want no matching tags warning", result.Diagnostics)
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file %s to exist: %v", path, err)
	}
}

func assertFileNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("expected file %s not to exist", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("checking file %s: %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}
