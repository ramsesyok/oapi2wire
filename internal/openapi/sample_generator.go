package openapi

import (
	"sort"
	"strconv"

	"github.com/getkin/kin-openapi/openapi3"
)

const maxDepth = 10

// MinimalSample generates a minimal Go value from a JSON Schema.
// Used for init template body values.
// Returns nil if schema is nil.
func MinimalSample(schema *openapi3.SchemaRef) interface{} {
	if schema == nil || schema.Value == nil {
		return nil
	}
	return minimalFromSchema(schema.Value, 0)
}

// minimalFromSchema is the recursive helper.
func minimalFromSchema(s *openapi3.Schema, depth int) interface{} {
	if s == nil || depth > maxDepth {
		return nil
	}

	// Handle allOf/anyOf/oneOf by using the first option
	if len(s.AllOf) > 0 && s.AllOf[0] != nil && s.AllOf[0].Value != nil {
		return minimalFromSchema(s.AllOf[0].Value, depth+1)
	}
	if len(s.AnyOf) > 0 && s.AnyOf[0] != nil && s.AnyOf[0].Value != nil {
		return minimalFromSchema(s.AnyOf[0].Value, depth+1)
	}
	if len(s.OneOf) > 0 && s.OneOf[0] != nil && s.OneOf[0].Value != nil {
		return minimalFromSchema(s.OneOf[0].Value, depth+1)
	}

	types := s.Type
	if types == nil {
		// Try to infer from properties
		if len(s.Properties) > 0 {
			return buildObject(s, depth)
		}
		if s.Items != nil {
			return buildArray(s, depth)
		}
		return nil
	}

	for _, t := range *types {
		switch t {
		case "object":
			return buildObject(s, depth)
		case "array":
			return buildArray(s, depth)
		case "string":
			return "TODO"
		case "integer", "number":
			return 0
		case "boolean":
			return true
		case "null":
			return nil
		}
	}

	return nil
}

func buildObject(s *openapi3.Schema, depth int) interface{} {
	obj := make(map[string]interface{})
	if len(s.Properties) == 0 {
		return obj
	}
	// Sort property names for deterministic output
	keys := make([]string, 0, len(s.Properties))
	for k := range s.Properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		propRef := s.Properties[k]
		if propRef != nil && propRef.Value != nil {
			obj[k] = minimalFromSchema(propRef.Value, depth+1)
		} else {
			obj[k] = "TODO"
		}
	}
	return obj
}

func buildArray(s *openapi3.Schema, depth int) interface{} {
	if s.Items == nil || s.Items.Value == nil {
		return []interface{}{}
	}
	elem := minimalFromSchema(s.Items.Value, depth+1)
	return []interface{}{elem}
}

// FirstResponseExample returns the first example from a response's content for the given status.
// Returns nil if no example is found.
func FirstResponseExample(responses *openapi3.Responses, status int) interface{} {
	if responses == nil {
		return nil
	}

	statusStr := strconv.Itoa(status)
	respRef := responses.Value(statusStr)
	if respRef == nil {
		return nil
	}

	resp := respRef.Value
	if resp == nil {
		return nil
	}

	// Try application/json content
	mt, ok := resp.Content["application/json"]
	if !ok || mt == nil {
		return nil
	}

	// Try Example field
	if mt.Example != nil {
		return mt.Example
	}

	// Try Examples map (use first entry sorted by key)
	if len(mt.Examples) > 0 {
		keys := make([]string, 0, len(mt.Examples))
		for k := range mt.Examples {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		ex := mt.Examples[keys[0]]
		if ex != nil && ex.Value != nil {
			return ex.Value.Value
		}
	}

	// Try schema example
	if mt.Schema != nil && mt.Schema.Value != nil {
		if mt.Schema.Value.Example != nil {
			return mt.Schema.Value.Example
		}
	}

	return nil
}

// FirstRequestBodyExample returns the first example from a requestBody's application/json content.
func FirstRequestBodyExample(op *openapi3.Operation) interface{} {
	if op.RequestBody == nil || op.RequestBody.Value == nil {
		return nil
	}
	mt, ok := op.RequestBody.Value.Content["application/json"]
	if !ok || mt == nil {
		return nil
	}

	if mt.Example != nil {
		return mt.Example
	}

	if len(mt.Examples) > 0 {
		keys := make([]string, 0, len(mt.Examples))
		for k := range mt.Examples {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		ex := mt.Examples[keys[0]]
		if ex != nil && ex.Value != nil {
			return ex.Value.Value
		}
	}

	if mt.Schema != nil && mt.Schema.Value != nil {
		if mt.Schema.Value.Example != nil {
			return mt.Schema.Value.Example
		}
		// Generate minimal sample from schema
		return MinimalSample(mt.Schema)
	}

	return nil
}
