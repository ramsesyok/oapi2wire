package openapi

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/ramsesyok/oapi2wire/internal/model"
)

// BuildOperationIndex scans all paths/methods in doc and returns:
//   - map[operationId]ResolvedOperation for fast lookup
//   - []Diagnostic for duplicate operationIds (error)
func BuildOperationIndex(doc *v3.Document) (map[string]model.ResolvedOperation, []model.Diagnostic) {
	index := make(map[string]model.ResolvedOperation)
	var diags []model.Diagnostic

	if doc == nil || doc.Paths == nil || doc.Paths.PathItems == nil {
		return index, diags
	}

	paths := make([]string, 0)
	pathItems := make(map[string]*v3.PathItem)
	for pair := doc.Paths.PathItems.Oldest(); pair != nil; pair = pair.Next() {
		paths = append(paths, pair.Key)
		pathItems[pair.Key] = pair.Value
	}
	sort.Strings(paths)

	for _, path := range paths {
		pathItem := pathItems[path]
		if pathItem == nil {
			continue
		}
		ops := pathItem.GetOperations()
		if ops == nil {
			continue
		}

		methods := make([]string, 0, ops.Len())
		operationByMethod := make(map[string]*v3.Operation)
		for pair := ops.Oldest(); pair != nil; pair = pair.Next() {
			methods = append(methods, pair.Key)
			operationByMethod[pair.Key] = pair.Value
		}
		sort.Strings(methods)

		for _, method := range methods {
			op := operationByMethod[method]
			if op == nil || op.OperationId == "" {
				continue
			}

			if _, exists := index[op.OperationId]; exists {
				diags = append(diags, model.Diagnostic{
					Severity: model.SeverityError,
					Path:     fmt.Sprintf("paths[%s].%s", path, strings.ToLower(method)),
					Message:  fmt.Sprintf("duplicate operationId %q", op.OperationId),
				})
				continue
			}

			resolved := model.ResolvedOperation{
				OperationID:          op.OperationId,
				Method:               strings.ToUpper(method),
				Path:                 path,
				Tags:                 append([]string(nil), op.Tags...),
				PathParams:           extractPathParams(op, pathItem),
				QueryParams:          extractQueryParams(op, pathItem),
				HasJSONBody:          hasJSONRequestBody(op),
				RepresentativeStatus: resolveRepresentativeStatus(op.Responses),
			}
			index[op.OperationId] = resolved
		}
	}

	return index, diags
}

// FilterOperationsByTags returns operations that have at least one exact tag match.
// Empty tags keep the original operation index unchanged.
func FilterOperationsByTags(ops map[string]model.ResolvedOperation, tags []string) (map[string]model.ResolvedOperation, []model.Diagnostic) {
	if len(tags) == 0 {
		return ops, nil
	}

	wanted := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		wanted[tag] = struct{}{}
	}

	filtered := make(map[string]model.ResolvedOperation)
	for opID, op := range ops {
		if operationHasAnyTag(op, wanted) {
			filtered[opID] = op
		}
	}

	if len(filtered) == 0 {
		return filtered, []model.Diagnostic{{
			Severity: model.SeverityWarning,
			Path:     "tags",
			Message:  fmt.Sprintf("no operations matched tags: %s", strings.Join(tags, ", ")),
		}}
	}

	return filtered, nil
}

func operationHasAnyTag(op model.ResolvedOperation, wanted map[string]struct{}) bool {
	for _, tag := range op.Tags {
		if _, ok := wanted[tag]; ok {
			return true
		}
	}
	return false
}

// resolveRepresentativeStatus picks the best response status for init templates.
// Priority: first 2xx → first response → 200.
func resolveRepresentativeStatus(responses *v3.Responses) int {
	if responses == nil {
		return 200
	}

	codes := make([]string, 0)
	if responses.Codes != nil {
		for pair := responses.Codes.Oldest(); pair != nil; pair = pair.Next() {
			codes = append(codes, pair.Key)
		}
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
func extractPathParams(op *v3.Operation, pathItem *v3.PathItem) []string {
	seen := make(map[string]bool)
	var names []string

	collect := func(params []*v3.Parameter) {
		for _, p := range params {
			if p == nil {
				continue
			}
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
func extractQueryParams(op *v3.Operation, pathItem *v3.PathItem) []model.QueryParam {
	seen := make(map[string]bool)
	var params []model.QueryParam

	collect := func(ps []*v3.Parameter) {
		for _, p := range ps {
			if p == nil {
				continue
			}
			if p.In == "query" && !seen[p.Name] {
				seen[p.Name] = true
				required := p.Required != nil && *p.Required
				params = append(params, model.QueryParam{
					Name:     p.Name,
					Required: required,
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
func hasJSONRequestBody(op *v3.Operation) bool {
	if op.RequestBody == nil || op.RequestBody.Content == nil {
		return false
	}
	for pair := op.RequestBody.Content.Oldest(); pair != nil; pair = pair.Next() {
		if strings.Contains(pair.Key, "application/json") {
			return true
		}
	}
	return false
}
