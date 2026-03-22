package model

// CaseFile is the root of a case YAML file.
type CaseFile struct {
	Version  int       `yaml:"version"`
	Defaults Defaults  `yaml:"defaults"`
	Cases    []CaseSpec `yaml:"cases"`
}

// Defaults carries global defaults applied to every case.
type Defaults struct {
	Response DefaultResponse `yaml:"response"`
}

// DefaultResponse holds the default response settings.
type DefaultResponse struct {
	Headers map[string]string `yaml:"headers"`
}

// CaseSpec is one mock case entry.
type CaseSpec struct {
	ID          string       `yaml:"id"`
	OperationID string       `yaml:"operationId"`
	Priority    int          `yaml:"priority"`
	Fallback    bool         `yaml:"fallback"`
	Request     *RequestSpec `yaml:"request,omitempty"`
	Response    ResponseSpec `yaml:"response"`
}

// RequestSpec holds the matcher conditions for a case.
type RequestSpec struct {
	PathParams map[string]StringMatcher `yaml:"pathParams,omitempty"`
	Query      map[string]StringMatcher `yaml:"query,omitempty"`
	Body       *BodyMatcher             `yaml:"body,omitempty"`
}

// StringMatcher matches a single string value via equalTo or matches.
// Exactly one field must be set.
type StringMatcher struct {
	EqualTo *string `yaml:"equalTo,omitempty"`
	Matches *string `yaml:"matches,omitempty"`
}

// BodyMatcher matches a JSON request body.
// At least one field must be set.
type BodyMatcher struct {
	EqualToJson     interface{} `yaml:"equalToJson,omitempty"`
	MatchesJsonPath []string    `yaml:"matchesJsonPath,omitempty"`
}

// ResponseSpec defines the response for a case.
type ResponseSpec struct {
	Status   int               `yaml:"status"`
	Headers  map[string]string `yaml:"headers,omitempty"`
	BodyFile string            `yaml:"bodyFile"`
}
