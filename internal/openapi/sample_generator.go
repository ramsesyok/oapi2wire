package openapi

import (
	"sort"
	"strconv"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

const maxDepth = 10

// MinimalSample generates a minimal Go value from a JSON Schema.
// Used for init template body values.
// Returns nil if schema is nil.
func MinimalSample(schemaProxy *base.SchemaProxy) interface{} {
	if schemaProxy == nil {
		return nil
	}
	schema, err := schemaProxy.BuildSchema()
	if err != nil || schema == nil {
		return nil
	}
	return minimalFromSchema(schema, 0)
}

// minimalFromSchema is the recursive helper.
func minimalFromSchema(s *base.Schema, depth int) interface{} {
	if s == nil || depth > maxDepth {
		return nil
	}

	if s.Example != nil {
		if v := yamlNodeToInterface(s.Example); v != nil {
			return v
		}
	}

	// Handle allOf/anyOf/oneOf by using the first option
	if len(s.AllOf) > 0 && s.AllOf[0] != nil {
		if schema, err := s.AllOf[0].BuildSchema(); err == nil && schema != nil {
			return minimalFromSchema(schema, depth+1)
		}
	}
	if len(s.AnyOf) > 0 && s.AnyOf[0] != nil {
		if schema, err := s.AnyOf[0].BuildSchema(); err == nil && schema != nil {
			return minimalFromSchema(schema, depth+1)
		}
	}
	if len(s.OneOf) > 0 && s.OneOf[0] != nil {
		if schema, err := s.OneOf[0].BuildSchema(); err == nil && schema != nil {
			return minimalFromSchema(schema, depth+1)
		}
	}

	types := s.Type
	if len(types) == 0 {
		if s.Properties != nil && s.Properties.Len() > 0 {
			return buildObject(s, depth)
		}
		if s.Items != nil {
			return buildArray(s, depth)
		}
	}

	for _, t := range types {
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

func buildObject(s *base.Schema, depth int) interface{} {
	obj := make(map[string]interface{})
	if s.Properties == nil || s.Properties.Len() == 0 {
		return obj
	}

	keys := make([]string, 0, s.Properties.Len())
	props := make(map[string]*base.SchemaProxy)
	for pair := s.Properties.Oldest(); pair != nil; pair = pair.Next() {
		keys = append(keys, pair.Key)
		props[pair.Key] = pair.Value
	}
	sort.Strings(keys)

	for _, k := range keys {
		propRef := props[k]
		if propRef != nil {
			if prop, err := propRef.BuildSchema(); err == nil && prop != nil {
				obj[k] = minimalFromSchema(prop, depth+1)
				continue
			}
		}
		obj[k] = "TODO"
	}
	return obj
}

func buildArray(s *base.Schema, depth int) interface{} {
	if s.Items == nil {
		return []interface{}{}
	}
	if !s.Items.IsA() || s.Items.A == nil {
		return []interface{}{}
	}
	item, err := s.Items.A.BuildSchema()
	if err != nil || item == nil {
		return []interface{}{}
	}
	elem := minimalFromSchema(item, depth+1)
	return []interface{}{elem}
}

// FirstResponseExample returns the first example from a response's content for the given status.
// Returns nil if no example is found.
func FirstResponseExample(responses *v3.Responses, status int) interface{} {
	if responses == nil {
		return nil
	}

	statusStr := strconv.Itoa(status)
	if responses.Codes == nil {
		return nil
	}
	resp := responses.Codes.GetOrZero(statusStr)
	if resp == nil {
		return nil
	}

	mt := jsonMediaType(resp.Content)
	if mt == nil {
		return nil
	}

	if mt.Example != nil {
		return yamlNodeToInterface(mt.Example)
	}

	if mt.Examples != nil && mt.Examples.Len() > 0 {
		ex := mt.Examples.First().Value()
		if ex != nil && ex.Value != nil {
			return yamlNodeToInterface(ex.Value)
		}
	}

	if mt.Schema != nil {
		if schema, err := mt.Schema.BuildSchema(); err == nil && schema != nil && schema.Example != nil {
			return yamlNodeToInterface(schema.Example)
		}
	}

	return nil
}

// FirstRequestBodyExample returns the first example from a requestBody's application/json content.
func FirstRequestBodyExample(op *v3.Operation) interface{} {
	if op == nil || op.RequestBody == nil {
		return nil
	}
	mt := jsonMediaType(op.RequestBody.Content)
	if mt == nil {
		return nil
	}

	if mt.Example != nil {
		return yamlNodeToInterface(mt.Example)
	}

	if mt.Examples != nil && mt.Examples.Len() > 0 {
		ex := mt.Examples.First().Value()
		if ex != nil && ex.Value != nil {
			return yamlNodeToInterface(ex.Value)
		}
	}

	if mt.Schema != nil {
		return MinimalSample(mt.Schema)
	}

	return nil
}

func jsonMediaType(content *orderedmap.Map[string, *v3.MediaType]) *v3.MediaType {
	if content == nil {
		return nil
	}
	for pair := content.Oldest(); pair != nil; pair = pair.Next() {
		if strings.Contains(pair.Key, "application/json") {
			return pair.Value
		}
	}
	return nil
}

func yamlNodeToInterface(node *yaml.Node) interface{} {
	if node == nil {
		return nil
	}
	var v interface{}
	if err := node.Decode(&v); err != nil {
		return nil
	}
	return v
}
