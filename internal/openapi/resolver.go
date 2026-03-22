package openapi

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/ramsesyok/oapi2wire/internal/model"
)

// BuildOperationIndex scans all paths/methods in doc and returns:
//   - map[operationId]ResolvedOperation for fast lookup
//   - []Diagnostic for duplicate operationIds (error)
func BuildOperationIndex(doc *openapi3.T) (map[string]model.ResolvedOperation, []model.Diagnostic) {
	index := make(map[string]model.ResolvedOperation)
	var diags []model.Diagnostic

	if doc.Paths == nil {
		return index, diags
	}

	// Sort paths for deterministic iteration order
	pathMap := doc.Paths.Map()
	paths := make([]string, 0, len(pathMap))
	for p := range pathMap {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, path := range paths {
		pathItem := pathMap[path]
		ops := pathItem.Operations()

		// Sort methods for deterministic order
		methods := make([]string, 0, len(ops))
		for m := range ops {
			methods = append(methods, m)
		}
		sort.Strings(methods)

		for _, method := range methods {
			op := ops[method]
			if op.OperationID == "" {
				continue
			}

			if _, exists := index[op.OperationID]; exists {
				diags = append(diags, model.Diagnostic{
					Severity: model.SeverityError,
					Path:     fmt.Sprintf("paths[%s].%s", path, strings.ToLower(method)),
					Message:  fmt.Sprintf("duplicate operationId %q", op.OperationID),
				})
				continue
			}

			resolved := model.ResolvedOperation{
				OperationID:          op.OperationID,
				Method:               strings.ToUpper(method),
				Path:                 path,
				PathParams:           extractPathParams(op, pathItem),
				QueryParams:          extractQueryParams(op, pathItem),
				HasJSONBody:          hasJSONRequestBody(op),
				RepresentativeStatus: resolveRepresentativeStatus(op.Responses),
			}
			index[op.OperationID] = resolved
		}
	}

	return index, diags
}

// resolveRepresentativeStatus picks the best response status for init templates.
// Priority: first 2xx → first response → 200.
func resolveRepresentativeStatus(responses *openapi3.Responses) int {
	if responses == nil {
		return 200
	}

	// Sort status codes for deterministic iteration
	statusMap := responses.Map()
	codes := make([]string, 0, len(statusMap))
	for code := range statusMap {
		codes = append(codes, code)
	}
	sort.Strings(codes)

	// First, look for a 2xx
	for _, code := range codes {
		n, err := strconv.Atoi(code)
		if err == nil && n >= 200 && n < 300 {
			return n
		}
	}

	// Fall back to first numeric response
	for _, code := range codes {
		n, err := strconv.Atoi(code)
		if err == nil {
			return n
		}
	}

	return 200
}

// extractPathParams returns parameter names for in=path parameters.
func extractPathParams(op *openapi3.Operation, pathItem *openapi3.PathItem) []string {
	seen := make(map[string]bool)
	var names []string

	collect := func(params openapi3.Parameters) {
		for _, pRef := range params {
			if pRef == nil || pRef.Value == nil {
				continue
			}
			p := pRef.Value
			if p.In == "path" && !seen[p.Name] {
				seen[p.Name] = true
				names = append(names, p.Name)
			}
		}
	}

	// Path-level parameters first, then operation-level (op-level overrides)
	if pathItem.Parameters != nil {
		collect(pathItem.Parameters)
	}
	if op.Parameters != nil {
		collect(op.Parameters)
	}

	return names
}

// extractQueryParams returns QueryParam entries for in=query parameters.
func extractQueryParams(op *openapi3.Operation, pathItem *openapi3.PathItem) []model.QueryParam {
	seen := make(map[string]bool)
	var params []model.QueryParam

	collect := func(ps openapi3.Parameters) {
		for _, pRef := range ps {
			if pRef == nil || pRef.Value == nil {
				continue
			}
			p := pRef.Value
			if p.In == "query" && !seen[p.Name] {
				seen[p.Name] = true
				params = append(params, model.QueryParam{
					Name:     p.Name,
					Required: p.Required,
				})
			}
		}
	}

	if pathItem.Parameters != nil {
		collect(pathItem.Parameters)
	}
	if op.Parameters != nil {
		collect(op.Parameters)
	}

	return params
}

// hasJSONRequestBody returns true if the operation has a requestBody with application/json.
func hasJSONRequestBody(op *openapi3.Operation) bool {
	if op.RequestBody == nil || op.RequestBody.Value == nil {
		return false
	}
	_, ok := op.RequestBody.Value.Content["application/json"]
	return ok
}
