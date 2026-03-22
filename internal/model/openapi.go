package model

// ResolvedOperation is the result of resolving one operationId from OpenAPI.
type ResolvedOperation struct {
	OperationID          string
	Method               string       // uppercase: GET, POST, etc.
	Path                 string       // as written in OpenAPI: /users/{id}
	PathParams           []string     // ordered list of path parameter names
	QueryParams          []QueryParam // all query parameters
	HasJSONBody          bool         // true when requestBody has application/json
	RepresentativeStatus int          // chosen 2xx or first status or 200
}

// QueryParam carries metadata for a single query parameter.
type QueryParam struct {
	Name     string
	Required bool
}
