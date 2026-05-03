// Package oapi2wire exposes the generator workflow for CLI and library users.
package oapi2wire

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/ramsesyok/oapi2wire/internal/cases"
	"github.com/ramsesyok/oapi2wire/internal/generator"
	"github.com/ramsesyok/oapi2wire/internal/model"
	"github.com/ramsesyok/oapi2wire/internal/openapi"
	"github.com/ramsesyok/oapi2wire/internal/output"
)

// Severity is the diagnostic severity.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Diagnostic is a human-actionable validation or generation diagnostic.
type Diagnostic struct {
	Severity Severity
	Path     string
	Message  string
	Hint     string
}

func (d Diagnostic) String() string {
	text := fmt.Sprintf("%s: %s", d.Severity, d.Message)
	if d.Path != "" {
		text = fmt.Sprintf("%s: %s", d.Severity, d.Path+": "+d.Message)
	}
	if d.Hint != "" {
		text += "\nhint: " + d.Hint
	}
	return text
}

// InitOptions controls case YAML and response stub generation.
type InitOptions struct {
	OpenAPIPath   string
	OutCasesPath  string
	ResponsesRoot string
	Force         bool
	Strict        bool
}

// InitResult describes generated init outputs.
type InitResult struct {
	Diagnostics          []Diagnostic
	GeneratedCases       int
	ResponseFilesWritten int
	OutCasesPath         string
	ResponsesRoot        string
}

// BuildOptions controls WireMock artifact generation.
type BuildOptions struct {
	OpenAPIPath            string
	CasesPath              string
	ResponsesRoot          string
	OutDir                 string
	Clean                  bool
	Strict                 bool
	FailOnMissingOperation bool
	FailOnMissingBodyFile  bool
	NoAutoFallback         bool
}

// BuildResult describes generated WireMock outputs.
type BuildResult struct {
	Diagnostics       []Diagnostic
	MappingsWritten   int
	BodyFilesCopied   int
	FallbacksWritten  int
	OutDir            string
	ResponsesRoot     string
	AutoFallbacksUsed bool
}

// ValidateOptions controls consistency validation.
type ValidateOptions struct {
	OpenAPIPath            string
	CasesPath              string
	ResponsesRoot          string
	Strict                 bool
	FailOnMissingOperation bool
	FailOnMissingBodyFile  bool
}

// ValidateResult contains accumulated validation diagnostics.
type ValidateResult struct {
	Diagnostics []Diagnostic
}

// Init generates a case YAML template and response JSON stubs from OpenAPI.
func Init(opts InitOptions) (*InitResult, error) {
	result := &InitResult{
		OutCasesPath:  opts.OutCasesPath,
		ResponsesRoot: opts.ResponsesRoot,
	}

	doc, err := openapi.Load(opts.OpenAPIPath)
	if err != nil {
		return result, fmt.Errorf("loading OpenAPI: %w", err)
	}

	ops, modelDiags := openapi.BuildOperationIndex(doc)
	diags := convertDiagnostics(modelDiags)
	result.Diagnostics = diags
	if hasErrors(diags) {
		return result, fmt.Errorf("OpenAPI has errors")
	}
	if opts.Strict && len(diags) > 0 {
		return result, fmt.Errorf("OpenAPI has warnings (--strict)")
	}

	template, err := cases.GenerateTemplate(doc, ops)
	if err != nil {
		return result, fmt.Errorf("generating template: %w", err)
	}

	if !opts.Force {
		if _, err := os.Stat(opts.OutCasesPath); err == nil {
			return result, fmt.Errorf("file already exists: %s (use force to overwrite)", opts.OutCasesPath)
		}
	}

	written, err := template.Write(opts.OutCasesPath, opts.ResponsesRoot, opts.Force)
	if err != nil {
		return result, fmt.Errorf("writing init output: %w", err)
	}

	result.GeneratedCases = len(ops)
	result.ResponseFilesWritten = written
	return result, nil
}

// Build generates WireMock mappings and __files from OpenAPI, case YAML, and response files.
func Build(opts BuildOptions) (*BuildResult, error) {
	result := &BuildResult{
		OutDir:            opts.OutDir,
		ResponsesRoot:     opts.ResponsesRoot,
		AutoFallbacksUsed: !opts.NoAutoFallback,
	}

	doc, err := openapi.Load(opts.OpenAPIPath)
	if err != nil {
		return result, fmt.Errorf("loading OpenAPI: %w", err)
	}

	ops, modelDiags := openapi.BuildOperationIndex(doc)
	if model.HasErrors(modelDiags) {
		result.Diagnostics = convertDiagnostics(modelDiags)
		return result, fmt.Errorf("OpenAPI has errors")
	}

	cf, err := cases.Load(opts.CasesPath)
	if err != nil {
		result.Diagnostics = convertDiagnostics(modelDiags)
		return result, fmt.Errorf("loading case YAML: %w", err)
	}

	modelDiags = append(modelDiags, cases.Validate(cf, ops, cases.ValidateConfig{
		FailOnMissingOperation: opts.FailOnMissingOperation,
		FailOnMissingBodyFile:  opts.FailOnMissingBodyFile,
		ResponsesRoot:          opts.ResponsesRoot,
	})...)
	diags := convertDiagnostics(modelDiags)
	result.Diagnostics = diags
	if hasErrors(diags) {
		return result, fmt.Errorf("validation failed with errors")
	}
	if opts.Strict && len(diags) > 0 {
		return result, fmt.Errorf("validation has warnings (--strict)")
	}

	if err := output.PrepareOutDir(output.WriterConfig{OutDir: opts.OutDir, Clean: opts.Clean}); err != nil {
		return result, fmt.Errorf("preparing output dir: %w", err)
	}

	for _, cs := range cf.Cases {
		op, ok := ops[cs.OperationID]
		if !ok {
			continue
		}

		mapping, err := generator.BuildMapping(cs, op, cf.Defaults)
		if err != nil {
			return result, fmt.Errorf("building mapping for case %q: %w", cs.ID, err)
		}

		filename := generator.MappingFileName(op.OperationID, cs.ID)
		if err := output.WriteMapping(opts.OutDir, filename, mapping); err != nil {
			return result, fmt.Errorf("writing mapping %s: %w", filename, err)
		}
		result.MappingsWritten++
	}

	filesDir := filepath.Join(opts.OutDir, "__files")
	copied, err := generator.CopyBodyFiles(cf.Cases, opts.ResponsesRoot, filesDir)
	if err != nil {
		return result, fmt.Errorf("copying body files: %w", err)
	}
	result.BodyFilesCopied = copied

	if !opts.NoAutoFallback {
		opIDs := make([]string, 0, len(ops))
		for id := range ops {
			opIDs = append(opIDs, id)
		}
		sort.Strings(opIDs)

		for _, opID := range opIDs {
			if !generator.NeedsAutoFallback(opID, cf.Cases) {
				continue
			}
			fb := generator.BuildFallback(ops[opID])

			if err := output.WriteMapping(opts.OutDir, fb.MappingFileName, fb.Mapping); err != nil {
				return result, fmt.Errorf("writing fallback mapping: %w", err)
			}
			if err := output.WriteBodyFile(opts.OutDir, fb.BodyFileName, fb.BodyContent); err != nil {
				return result, fmt.Errorf("writing fallback body: %w", err)
			}
			result.MappingsWritten++
			result.FallbacksWritten++
		}
	}

	return result, nil
}

// Validate checks consistency between OpenAPI and case YAML without generating files.
func Validate(opts ValidateOptions) (*ValidateResult, error) {
	result := &ValidateResult{}

	doc, err := openapi.Load(opts.OpenAPIPath)
	if err != nil {
		return result, fmt.Errorf("loading OpenAPI: %w", err)
	}

	ops, modelDiags := openapi.BuildOperationIndex(doc)
	if model.HasErrors(modelDiags) {
		result.Diagnostics = convertDiagnostics(modelDiags)
		return result, fmt.Errorf("OpenAPI has errors")
	}

	cf, err := cases.Load(opts.CasesPath)
	if err != nil {
		result.Diagnostics = convertDiagnostics(modelDiags)
		return result, fmt.Errorf("loading case YAML: %w", err)
	}

	modelDiags = append(modelDiags, cases.Validate(cf, ops, cases.ValidateConfig{
		FailOnMissingOperation: opts.FailOnMissingOperation,
		FailOnMissingBodyFile:  opts.FailOnMissingBodyFile,
		ResponsesRoot:          opts.ResponsesRoot,
	})...)
	diags := convertDiagnostics(modelDiags)
	result.Diagnostics = diags
	if hasErrors(diags) {
		return result, fmt.Errorf("validation failed with errors")
	}
	if opts.Strict && len(diags) > 0 {
		return result, fmt.Errorf("validation has warnings (--strict)")
	}

	return result, nil
}

func convertDiagnostics(diags []model.Diagnostic) []Diagnostic {
	if len(diags) == 0 {
		return nil
	}
	result := make([]Diagnostic, 0, len(diags))
	for _, d := range diags {
		result = append(result, Diagnostic{
			Severity: convertSeverity(d.Severity),
			Path:     d.Path,
			Message:  d.Message,
			Hint:     d.Hint,
		})
	}
	return result
}

func convertSeverity(sev model.Severity) Severity {
	if sev == model.SeverityError {
		return SeverityError
	}
	return SeverityWarning
}

func hasErrors(diags []Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}
