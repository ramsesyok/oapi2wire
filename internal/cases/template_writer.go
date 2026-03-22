package cases

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	oapipkg "github.com/ramsesyok/oapi2wire/internal/openapi"
	"github.com/ramsesyok/oapi2wire/internal/model"
)

// TemplateResult is the full generated init output for one run.
type TemplateResult struct {
	// CaseYAML is the rendered mock-cases.yaml content (with comments preserved).
	CaseYAML []byte
	// ResponseFiles maps relative bodyFile path → content.
	ResponseFiles map[string]interface{}
}

// WriteTemplate writes CaseYAML to outCasesPath and response stubs under responsesRoot.
func (r *TemplateResult) Write(outCasesPath, responsesRoot string, force bool) (int, error) {
	// Write case YAML
	if err := os.MkdirAll(filepath.Dir(outCasesPath), 0o755); err != nil {
		return 0, err
	}
	if err := os.WriteFile(outCasesPath, r.CaseYAML, 0o644); err != nil {
		return 0, err
	}

	written := 0
	for relPath, content := range r.ResponseFiles {
		fullPath := filepath.Join(responsesRoot, relPath)
		if !force {
			if _, err := os.Stat(fullPath); err == nil {
				continue
			}
		}
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return written, err
		}
		data, err := json.MarshalIndent(content, "", "  ")
		if err != nil {
			return written, err
		}
		data = append(data, '\n')
		if err := os.WriteFile(fullPath, data, 0o644); err != nil {
			return written, err
		}
		written++
	}
	return written, nil
}

// GenerateTemplate produces the case YAML template and response stubs from an OpenAPI doc.
func GenerateTemplate(doc *openapi3.T, ops map[string]model.ResolvedOperation) (*TemplateResult, error) {
	result := &TemplateResult{
		ResponseFiles: make(map[string]interface{}),
	}

	// Sort operationIds for deterministic output
	opIDs := make([]string, 0, len(ops))
	for id := range ops {
		opIDs = append(opIDs, id)
	}
	sort.Strings(opIDs)

	var sb strings.Builder
	sb.WriteString("version: 1\n")
	sb.WriteString("defaults:\n")
	sb.WriteString("  response:\n")
	sb.WriteString("    headers:\n")
	sb.WriteString("      Content-Type: application/json\n")
	sb.WriteString("\n")
	sb.WriteString("cases:\n")

	for _, opID := range opIDs {
		op := ops[opID]
		rawOp := findRawOperation(doc, op)

		caseID := fmt.Sprintf("%s_default", op.OperationID)
		bodyFile := fmt.Sprintf("%s/%s_default.json", op.OperationID, op.OperationID)

		sb.WriteString(fmt.Sprintf("  - id: %s\n", caseID))
		sb.WriteString(fmt.Sprintf("    operationId: %s\n", op.OperationID))
		sb.WriteString("    priority: 100\n")

		// Build request block
		requestYAML := buildRequestYAML(op, rawOp)
		if requestYAML != "" {
			sb.WriteString("    request:\n")
			sb.WriteString(requestYAML)
		}

		sb.WriteString("    response:\n")
		sb.WriteString(fmt.Sprintf("      status: %d\n", op.RepresentativeStatus))
		sb.WriteString(fmt.Sprintf("      bodyFile: %s\n", bodyFile))
		sb.WriteString("\n")

		// Generate response stub
		result.ResponseFiles[bodyFile] = responseBodyFor(doc, op)
	}

	result.CaseYAML = []byte(sb.String())
	return result, nil
}

// buildRequestYAML generates the request: sub-block as indented YAML lines (8-space indent base).
// Returns empty string if nothing to render.
func buildRequestYAML(op model.ResolvedOperation, rawOp *openapi3.Operation) string {
	var sb strings.Builder

	hasPathParams := len(op.PathParams) > 0
	hasRequiredQuery := false
	var optionalQueryNames []string
	for _, qp := range op.QueryParams {
		if qp.Required {
			hasRequiredQuery = true
		} else {
			optionalQueryNames = append(optionalQueryNames, qp.Name)
		}
	}
	hasBody := op.HasJSONBody

	if !hasPathParams && !hasRequiredQuery && len(optionalQueryNames) == 0 && !hasBody {
		return ""
	}

	// pathParams
	if hasPathParams {
		sb.WriteString("      pathParams:\n")
		for _, name := range op.PathParams {
			sb.WriteString(fmt.Sprintf("        %s:\n", name))
			sb.WriteString("          equalTo: \"TODO\"\n")
		}
	}

	// query: required params as real entries; optional params as comments (no query: key when only optional)
	if hasRequiredQuery {
		sb.WriteString("      query:\n")
		for _, qp := range op.QueryParams {
			if qp.Required {
				sb.WriteString(fmt.Sprintf("        %s:\n", qp.Name))
				sb.WriteString("          equalTo: \"TODO\"\n")
			}
		}
	}
	if len(optionalQueryNames) > 0 {
		sb.WriteString("      # optional query parameters:\n")
		for _, name := range optionalQueryNames {
			sb.WriteString(fmt.Sprintf("      # %s:\n", name))
			sb.WriteString("      #   equalTo: \"TODO\"\n")
		}
	}

	// body (from requestBody example or schema)
	if hasBody && rawOp != nil {
		bodyYAML := buildBodyYAML(rawOp)
		if bodyYAML != "" {
			sb.WriteString("      body:\n")
			sb.WriteString(bodyYAML)
		}
	}

	return sb.String()
}

// buildBodyYAML generates the body: sub-block lines (10-space indent base).
func buildBodyYAML(rawOp *openapi3.Operation) string {
	if rawOp.RequestBody == nil || rawOp.RequestBody.Value == nil {
		return ""
	}
	mt, ok := rawOp.RequestBody.Value.Content["application/json"]
	if !ok || mt == nil {
		return ""
	}

	var sample interface{}
	if mt.Example != nil {
		sample = mt.Example
	} else if mt.Schema != nil {
		sample = oapipkg.MinimalSample(mt.Schema)
	}
	if sample == nil {
		sample = map[string]interface{}{}
	}

	return fmt.Sprintf("        equalToJson:\n%s", valueToYAML(sample, "          "))
}

// valueToYAML renders a Go value as YAML lines with the given indent prefix.
func valueToYAML(v interface{}, indent string) string {
	var sb strings.Builder
	writeValue(&sb, v, indent)
	return sb.String()
}

func writeValue(sb *strings.Builder, v interface{}, indent string) {
	switch val := v.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			child := val[k]
			switch child.(type) {
			case map[string]interface{}, []interface{}:
				sb.WriteString(fmt.Sprintf("%s%s:\n", indent, k))
				writeValue(sb, child, indent+"  ")
			default:
				sb.WriteString(fmt.Sprintf("%s%s: %s\n", indent, k, scalarYAML(child)))
			}
		}
	case []interface{}:
		for _, elem := range val {
			switch elem.(type) {
			case map[string]interface{}, []interface{}:
				sb.WriteString(fmt.Sprintf("%s-\n", indent))
				writeValue(sb, elem, indent+"  ")
			default:
				sb.WriteString(fmt.Sprintf("%s- %s\n", indent, scalarYAML(elem)))
			}
		}
	default:
		sb.WriteString(fmt.Sprintf("%s%s\n", indent, scalarYAML(v)))
	}
}

func scalarYAML(v interface{}) string {
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%g", val)
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func responseBodyFor(doc *openapi3.T, op model.ResolvedOperation) interface{} {
	rawOp := findRawOperation(doc, op)
	if rawOp == nil {
		return map[string]interface{}{}
	}

	// Try response example
	if rawOp.Responses != nil {
		example := oapipkg.FirstResponseExample(rawOp.Responses, op.RepresentativeStatus)
		if example != nil {
			return example
		}

		// Try response schema
		statusStr := fmt.Sprintf("%d", op.RepresentativeStatus)
		respRef := rawOp.Responses.Value(statusStr)
		if respRef != nil && respRef.Value != nil {
			mt, ok := respRef.Value.Content["application/json"]
			if ok && mt != nil && mt.Schema != nil {
				sample := oapipkg.MinimalSample(mt.Schema)
				if sample != nil {
					return sample
				}
			}
		}
	}

	return map[string]interface{}{}
}

func findRawOperation(doc *openapi3.T, op model.ResolvedOperation) *openapi3.Operation {
	if doc.Paths == nil {
		return nil
	}
	pathItem := doc.Paths.Find(op.Path)
	if pathItem == nil {
		return nil
	}
	return pathItem.GetOperation(strings.ToUpper(op.Method))
}
