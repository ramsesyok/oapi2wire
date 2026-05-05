package cases

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ramsesyok/oapi2wire/internal/model"
	"github.com/ramsesyok/oapi2wire/internal/util"
)

// ValidateConfig holds flags that affect which rules produce errors vs warnings.
type ValidateConfig struct {
	FailOnMissingOperation bool
	FailOnMissingBodyFile  bool
	ResponsesRoot          string
	KnownOperationIDs      map[string]struct{}
	TagFilterActive        bool
}

// Validate runs all validation rules against cf and the operation index.
// Returns accumulated []Diagnostic; caller checks model.HasErrors to decide exit code.
func Validate(cf *model.CaseFile, ops map[string]model.ResolvedOperation, cfg ValidateConfig) []model.Diagnostic {
	var diags []model.Diagnostic
	scopedCases := casesInScope(cf.Cases, ops, cfg)

	diags = append(diags, checkDuplicateCaseIDs(cf.Cases)...)
	diags = append(diags, checkOperationIDsExist(cf.Cases, ops, cfg)...)
	diags = append(diags, checkFallbackRules(scopedCases)...)
	diags = append(diags, checkPathParamNames(scopedCases, ops)...)
	diags = append(diags, checkBodyFilePaths(scopedCases, cfg.ResponsesRoot, cfg.FailOnMissingBodyFile)...)
	diags = append(diags, checkEmptyMatchers(scopedCases)...)
	diags = append(diags, checkDuplicateConditions(scopedCases)...)
	diags = append(diags, checkDuplicatePriorities(scopedCases)...)
	diags = append(diags, checkMissingBodyMatcher(scopedCases, ops)...)

	return diags
}

func casesInScope(cases []model.CaseSpec, ops map[string]model.ResolvedOperation, cfg ValidateConfig) []model.CaseSpec {
	if !cfg.TagFilterActive {
		return cases
	}

	scoped := make([]model.CaseSpec, 0, len(cases))
	for _, c := range cases {
		if _, ok := ops[c.OperationID]; ok {
			scoped = append(scoped, c)
			continue
		}
		if _, known := cfg.KnownOperationIDs[c.OperationID]; !known {
			scoped = append(scoped, c)
		}
	}
	return scoped
}

func checkDuplicateCaseIDs(cases []model.CaseSpec) []model.Diagnostic {
	seen := make(map[string]int) // id → first index
	var diags []model.Diagnostic

	for i, c := range cases {
		if first, exists := seen[c.ID]; exists {
			diags = append(diags, model.Diagnostic{
				Severity: model.SeverityError,
				Path:     fmt.Sprintf("cases[%d].id", i),
				Message:  fmt.Sprintf("duplicate case id %q (first seen at cases[%d])", c.ID, first),
			})
		} else {
			seen[c.ID] = i
		}
	}
	return diags
}

func checkOperationIDsExist(cases []model.CaseSpec, ops map[string]model.ResolvedOperation, cfg ValidateConfig) []model.Diagnostic {
	var diags []model.Diagnostic

	for i, c := range cases {
		if _, ok := ops[c.OperationID]; !ok {
			if cfg.TagFilterActive {
				if _, known := cfg.KnownOperationIDs[c.OperationID]; known {
					diags = append(diags, model.Diagnostic{
						Severity: model.SeverityWarning,
						Path:     fmt.Sprintf("cases[%d].operationId", i),
						Message:  fmt.Sprintf("operationId %q is outside the tag filter", c.OperationID),
					})
					continue
				}
			}
			sev := model.SeverityWarning
			if cfg.FailOnMissingOperation {
				sev = model.SeverityError
			}
			diags = append(diags, model.Diagnostic{
				Severity: sev,
				Path:     fmt.Sprintf("cases[%d].operationId", i),
				Message:  fmt.Sprintf("operationId %q was not found in OpenAPI", c.OperationID),
			})
		}
	}
	return diags
}

func checkFallbackRules(cases []model.CaseSpec) []model.Diagnostic {
	var diags []model.Diagnostic
	fallbackCount := make(map[string]int) // operationId → count

	for i, c := range cases {
		if c.Fallback {
			fallbackCount[c.OperationID]++
			if c.Request != nil {
				diags = append(diags, model.Diagnostic{
					Severity: model.SeverityError,
					Path:     fmt.Sprintf("cases[%d]", i),
					Message:  "fallback case must not have a request matcher",
				})
			}
		}
	}

	for opID, count := range fallbackCount {
		if count > 1 {
			diags = append(diags, model.Diagnostic{
				Severity: model.SeverityError,
				Path:     fmt.Sprintf("cases[operationId=%s]", opID),
				Message:  fmt.Sprintf("operationId %q has %d fallback cases; only 1 is allowed", opID, count),
			})
		}
	}

	return diags
}

func checkPathParamNames(cases []model.CaseSpec, ops map[string]model.ResolvedOperation) []model.Diagnostic {
	var diags []model.Diagnostic

	for i, c := range cases {
		op, ok := ops[c.OperationID]
		if !ok || c.Request == nil || len(c.Request.PathParams) == 0 {
			continue
		}

		// Build set of valid path param names
		validNames := make(map[string]bool)
		for _, name := range op.PathParams {
			validNames[name] = true
		}

		for paramName := range c.Request.PathParams {
			if !validNames[paramName] {
				diags = append(diags, model.Diagnostic{
					Severity: model.SeverityError,
					Path:     fmt.Sprintf("cases[%d].request.pathParams.%s", i, paramName),
					Message:  fmt.Sprintf("path parameter %q not found in OpenAPI operation %q", paramName, c.OperationID),
					Hint:     fmt.Sprintf("valid path parameters: %v", op.PathParams),
				})
			}
		}
	}
	return diags
}

func checkBodyFilePaths(cases []model.CaseSpec, responsesRoot string, failOnMissing bool) []model.Diagnostic {
	var diags []model.Diagnostic

	for i, c := range cases {
		if c.Response.BodyFile == "" {
			continue
		}
		if err := util.ValidateBodyFilePath(c.Response.BodyFile); err != nil {
			diags = append(diags, model.Diagnostic{
				Severity: model.SeverityError,
				Path:     fmt.Sprintf("cases[%d].response.bodyFile", i),
				Message:  fmt.Sprintf("forbidden bodyFile path: %v", err),
			})
			continue
		}

		if responsesRoot != "" {
			fullPath := filepath.Join(responsesRoot, c.Response.BodyFile)
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				sev := model.SeverityWarning
				if failOnMissing {
					sev = model.SeverityError
				}
				diags = append(diags, model.Diagnostic{
					Severity: sev,
					Path:     fmt.Sprintf("cases[%d].response.bodyFile", i),
					Message:  fmt.Sprintf("body file not found: %s", fullPath),
				})
			}
		}
	}
	return diags
}

func checkEmptyMatchers(cases []model.CaseSpec) []model.Diagnostic {
	var diags []model.Diagnostic

	for i, c := range cases {
		if c.Fallback {
			continue
		}
		if c.Request == nil {
			diags = append(diags, model.Diagnostic{
				Severity: model.SeverityWarning,
				Path:     fmt.Sprintf("cases[%d]", i),
				Message:  fmt.Sprintf("case %q has no request matchers and is not a fallback", c.ID),
			})
			continue
		}
		if len(c.Request.PathParams) == 0 && len(c.Request.Query) == 0 && c.Request.Body == nil {
			diags = append(diags, model.Diagnostic{
				Severity: model.SeverityWarning,
				Path:     fmt.Sprintf("cases[%d].request", i),
				Message:  fmt.Sprintf("case %q has an empty request block with no matchers", c.ID),
			})
		}
	}
	return diags
}

func checkDuplicateConditions(cases []model.CaseSpec) []model.Diagnostic {
	var diags []model.Diagnostic
	type key struct {
		opID        string
		fingerprint string
	}
	seen := make(map[key]int)

	for i, c := range cases {
		fp := requestFingerprint(c.Request)
		k := key{opID: c.OperationID, fingerprint: fp}
		if first, exists := seen[k]; exists {
			diags = append(diags, model.Diagnostic{
				Severity: model.SeverityWarning,
				Path:     fmt.Sprintf("cases[%d]", i),
				Message:  fmt.Sprintf("case %q has the same operationId and request conditions as cases[%d]", c.ID, first),
			})
		} else {
			seen[k] = i
		}
	}
	return diags
}

func requestFingerprint(r *model.RequestSpec) string {
	if r == nil {
		return "nil"
	}
	data, _ := json.Marshal(r)
	return string(data)
}

func checkDuplicatePriorities(cases []model.CaseSpec) []model.Diagnostic {
	var diags []model.Diagnostic
	type key struct {
		opID     string
		priority int
	}
	seen := make(map[key]int)

	for i, c := range cases {
		if c.Priority == 0 {
			continue
		}
		k := key{opID: c.OperationID, priority: c.Priority}
		if first, exists := seen[k]; exists {
			diags = append(diags, model.Diagnostic{
				Severity: model.SeverityWarning,
				Path:     fmt.Sprintf("cases[%d].priority", i),
				Message:  fmt.Sprintf("case %q has the same priority %d as cases[%d] for operationId %q", c.ID, c.Priority, first, c.OperationID),
			})
		} else {
			seen[k] = i
		}
	}
	return diags
}

func checkMissingBodyMatcher(cases []model.CaseSpec, ops map[string]model.ResolvedOperation) []model.Diagnostic {
	var diags []model.Diagnostic

	for i, c := range cases {
		if c.Fallback {
			continue
		}
		op, ok := ops[c.OperationID]
		if !ok || !op.HasJSONBody {
			continue
		}
		if c.Request == nil || c.Request.Body == nil {
			diags = append(diags, model.Diagnostic{
				Severity: model.SeverityWarning,
				Path:     fmt.Sprintf("cases[%d]", i),
				Message:  fmt.Sprintf("case %q: OpenAPI operation %q has a requestBody but no body matcher is specified", c.ID, c.OperationID),
			})
		}
	}
	return diags
}
