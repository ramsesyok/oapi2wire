package oapisample_test

import (
	"encoding/json"
	"testing"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"

	"github.com/ramsesyok/oapi2wire/pkg/oapisample"
)

const spec = `openapi: 3.0.3
info: {title: t, version: "1"}
paths:
  /items/{id}:
    parameters:
      - {name: id, in: path, required: true, schema: {type: string}, example: "B0001"}
    get:
      operationId: getItem
      parameters:
        - {name: examples, in: query, schema: {type: string}, examples: {first: {value: "E1"}, second: {value: "E2"}}}
        - {name: schemaExample, in: query, schema: {type: integer, example: 7}}
        - {name: withDefault, in: query, schema: {type: boolean, default: false}}
        - {name: withEnum, in: query, schema: {type: string, enum: [NOVEL, TECH]}}
        - {name: date, in: query, schema: {type: string, format: date}}
        - {name: dateTime, in: query, schema: {type: string, format: date-time}}
        - {name: uuid, in: query, schema: {type: string, format: uuid}}
        - {name: plain, in: query, schema: {type: string}}
        - {name: flag, in: query, schema: {type: boolean}}
        - {name: number, in: query, schema: {type: number}}
        - {name: list, in: query, schema: {type: array, items: {type: string, example: "a"}}}
        - {name: noSchema, in: query}
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                properties:
                  id: {type: string}
                  count: {type: integer}
                  active: {type: boolean}
                  tags: {type: array, items: {type: string}}
                  owner:
                    type: object
                    properties:
                      name: {type: string, example: "花子"}
                  kind:
                    allOf:
                      - {type: string, enum: [A, B]}
        "201":
          description: example wins
          content:
            application/json:
              example: {id: "X"}
              schema: {type: object, properties: {id: {type: string}}}
        "202":
          description: examples
          content:
            application/json:
              examples:
                first: {value: {id: "first"}}
              schema: {type: object}
`

func operation(t *testing.T) (*v3.Operation, *v3.PathItem) {
	t.Helper()
	doc, err := libopenapi.NewDocument([]byte(spec))
	if err != nil {
		t.Fatal(err)
	}
	model, err := doc.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	item := model.Model.Paths.PathItems.GetOrZero("/items/{id}")
	return item.Get, item
}

func TestParameterString(t *testing.T) {
	op, item := operation(t)
	params := map[string]*v3.Parameter{}
	for _, p := range append(op.Parameters, item.Parameters...) {
		params[p.Name] = p
	}
	tests := []struct {
		name, want string
	}{
		{"id", "B0001"},                 // parameter.example (path item のパラメータ)
		{"examples", "E1"},              // examples の最初
		{"schemaExample", "7"},          // schema.example
		{"withDefault", "false"},        // schema.default
		{"withEnum", "NOVEL"},           // enum の最初
		{"date", oapisample.DateSample}, // format
		{"dateTime", oapisample.DateTimeSample},
		{"uuid", oapisample.UUIDSample},
		{"plain", oapisample.Placeholder}, // 型による値
		{"flag", "true"},
		{"number", "0"},
		{"list", "a"},                        // 配列は要素の値
		{"noSchema", oapisample.Placeholder}, // 決まらない
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, ok := params[tt.name]
			if !ok {
				t.Fatalf("parameter %s not found", tt.name)
			}
			if got := oapisample.ParameterString(p); got != tt.want {
				t.Errorf("ParameterString(%s) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
	if got := oapisample.ParameterString(nil); got != oapisample.Placeholder {
		t.Errorf("nil parameter = %q", got)
	}
}

func TestMediaType(t *testing.T) {
	op, _ := operation(t)
	tests := []struct {
		status, want string
	}{
		{"200", `{"active":true,"count":0,"id":"TODO","kind":"A","owner":{"name":"花子"},"tags":["TODO"]}`},
		{"201", `{"id":"X"}`},     // media type の example がスキーマより優先
		{"202", `{"id":"first"}`}, // examples の最初
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			resp := op.Responses.Codes.GetOrZero(tt.status)
			got, err := json.Marshal(oapisample.MediaType(oapisample.JSONMediaType(resp.Content)))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("MediaType(%s) = %s, want %s", tt.status, got, tt.want)
			}
		})
	}
	if oapisample.MediaType(nil) != nil {
		t.Error("nil media type should be nil")
	}
}

func TestString(t *testing.T) {
	for _, tt := range []struct {
		in   any
		want string
	}{
		{nil, oapisample.Placeholder},
		{"x", "x"},
		{true, "true"},
		{3, "3"},
		{int64(4), "4"},
		{1.5, "1.5"},
		{float64(10), "10"},
	} {
		if got := oapisample.String(tt.in); got != tt.want {
			t.Errorf("String(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
