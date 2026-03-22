package model

// WireMockMapping is the top-level JSON object written to mappings/*.json.
type WireMockMapping struct {
	ID       string           `json:"id"`
	Name     string           `json:"name"`
	Priority int              `json:"priority,omitempty"`
	Metadata WireMockMeta     `json:"metadata"`
	Request  WireMockRequest  `json:"request"`
	Response WireMockResponse `json:"response"`
}

// WireMockMeta is included in every generated mapping.
type WireMockMeta struct {
	Generator   string `json:"generator"`
	OperationID string `json:"operationId"`
	CaseID      string `json:"caseId"`
}

// WireMockRequest is the "request" block in a WireMock mapping.
type WireMockRequest struct {
	Method          string                     `json:"method"`
	URLPathTemplate string                     `json:"urlPathTemplate"`
	PathParameters  map[string]WireMockMatcher `json:"pathParameters,omitempty"`
	QueryParameters map[string]WireMockMatcher `json:"queryParameters,omitempty"`
	BodyPatterns    []map[string]interface{}   `json:"bodyPatterns,omitempty"`
}

// WireMockMatcher is a single equalTo or matches entry.
// Exactly one field will be set.
type WireMockMatcher struct {
	EqualTo *string `json:"equalTo,omitempty"`
	Matches *string `json:"matches,omitempty"`
}

// WireMockResponse is the "response" block in a WireMock mapping.
type WireMockResponse struct {
	Status       int               `json:"status"`
	Headers      map[string]string `json:"headers,omitempty"`
	BodyFileName string            `json:"bodyFileName"`
}
