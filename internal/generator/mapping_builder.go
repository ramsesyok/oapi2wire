package generator

import (
	"encoding/json"
	"fmt"

	"github.com/ramsesyok/oapi2wire/internal/model"
)

// BuildMapping converts one CaseSpec + its ResolvedOperation into a WireMockMapping.
// It merges default headers from CaseFile.Defaults into the response headers.
func BuildMapping(cs model.CaseSpec, op model.ResolvedOperation, defaults model.Defaults) (model.WireMockMapping, error) {
	req, err := buildRequest(cs, op)
	if err != nil {
		return model.WireMockMapping{}, fmt.Errorf("building request for case %q: %w", cs.ID, err)
	}

	headers := mergeHeaders(defaults.Response.Headers, cs.Response.Headers)

	return model.WireMockMapping{
		ID:       cs.ID,
		Name:     cs.ID,
		Priority: cs.Priority,
		Metadata: model.WireMockMeta{
			Generator:   "oapi2wire",
			OperationID: op.OperationID,
			CaseID:      cs.ID,
		},
		Request: req,
		Response: model.WireMockResponse{
			Status:       cs.Response.Status,
			Headers:      headers,
			BodyFileName: cs.Response.BodyFile,
		},
	}, nil
}

// buildRequest builds the WireMockRequest from a CaseSpec and operation.
func buildRequest(cs model.CaseSpec, op model.ResolvedOperation) (model.WireMockRequest, error) {
	req := model.WireMockRequest{
		Method:          op.Method,
		URLPathTemplate: op.Path,
	}

	if cs.Request == nil {
		return req, nil
	}

	if len(cs.Request.PathParams) > 0 {
		req.PathParameters = buildPathParameters(cs.Request.PathParams)
	}

	if len(cs.Request.Query) > 0 {
		req.QueryParameters = buildQueryParameters(cs.Request.Query)
	}

	if cs.Request.Body != nil {
		patterns, err := buildBodyPatterns(cs.Request.Body)
		if err != nil {
			return model.WireMockRequest{}, err
		}
		req.BodyPatterns = patterns
	}

	return req, nil
}

// buildPathParameters converts RequestSpec.PathParams to WireMock pathParameters.
func buildPathParameters(pm map[string]model.StringMatcher) map[string]model.WireMockMatcher {
	result := make(map[string]model.WireMockMatcher, len(pm))
	for name, sm := range pm {
		result[name] = stringMatcherToWireMock(sm)
	}
	return result
}

// buildQueryParameters converts RequestSpec.Query to WireMock queryParameters.
func buildQueryParameters(qm map[string]model.StringMatcher) map[string]model.WireMockMatcher {
	result := make(map[string]model.WireMockMatcher, len(qm))
	for name, sm := range qm {
		result[name] = stringMatcherToWireMock(sm)
	}
	return result
}

func stringMatcherToWireMock(sm model.StringMatcher) model.WireMockMatcher {
	return model.WireMockMatcher{
		EqualTo: sm.EqualTo,
		Matches: sm.Matches,
	}
}

// buildBodyPatterns converts RequestSpec.Body to WireMock bodyPatterns.
// equalToJson value is JSON-serialised to a string (WireMock expects a JSON string).
func buildBodyPatterns(bm *model.BodyMatcher) ([]map[string]interface{}, error) {
	var patterns []map[string]interface{}

	if bm.EqualToJson != nil {
		jsonBytes, err := json.MarshalIndent(bm.EqualToJson, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("serialising equalToJson: %w", err)
		}
		patterns = append(patterns, map[string]interface{}{
			"equalToJson": string(jsonBytes),
		})
	}

	for _, path := range bm.MatchesJsonPath {
		patterns = append(patterns, map[string]interface{}{
			"matchesJsonPath": path,
		})
	}

	return patterns, nil
}

// mergeHeaders merges defaults first, then case headers (case wins on collision).
func mergeHeaders(defaults, caseHeaders map[string]string) map[string]string {
	if len(defaults) == 0 && len(caseHeaders) == 0 {
		return nil
	}
	result := make(map[string]string)
	for k, v := range defaults {
		result[k] = v
	}
	for k, v := range caseHeaders {
		result[k] = v
	}
	return result
}

// MappingFileName returns the canonical file name for a mapping file.
// Format: <operationId>__<caseId>.json
func MappingFileName(operationID, caseID string) string {
	return fmt.Sprintf("%s__%s.json", safeFileNamePart(operationID), safeFileNamePart(caseID))
}
