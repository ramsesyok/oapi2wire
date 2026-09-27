package sample_test

import (
	"encoding/json"
	"testing"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"

	"github.com/ramsesyok/oapi2wire/pkg/sample"
)

// 規則は runnora の docs/design/sample-generation.md の 5 章。
// スキーマの規則は components.schemas に 1 つずつ書き、名前で引いて確かめる。
const spec31 = `openapi: 3.1.0
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
        - {name: list, in: query, schema: {type: array, items: {type: string, example: "a"}}}
        - {name: page, in: query, schema: {type: integer, minimum: 1}}
        - {name: plain, in: query, schema: {type: string}}
        - {name: noSchema, in: query}
      responses:
        "200":
          description: example wins
          content:
            application/json:
              example: {id: "X"}
              schema: {type: object, properties: {id: {type: string}}}
        "201":
          description: examples
          content:
            application/json:
              examples:
                first: {value: {id: "first"}}
              schema: {type: object}
components:
  schemas:
    Example: {type: string, example: "ex", default: "def"}
    Examples31: {type: string, examples: ["first", "second"], default: "def"}
    Default: {type: string, default: "def", enum: [a, b]}
    Const: {type: string, const: "fixed", enum: [a, b]}
    Enum: {type: string, enum: [a, b]}
    Date: {type: string, format: date}
    DateTime: {type: string, format: date-time}
    Time: {type: string, format: time}
    Email: {type: string, format: email}
    URI: {type: string, format: uri}
    Hostname: {type: string, format: hostname}
    IPv4: {type: string, format: ipv4}
    IPv6: {type: string, format: ipv6}
    UUID: {type: string, format: uuid}
    Byte: {type: string, format: byte}
    Binary: {type: string, format: binary}
    String: {type: string, minLength: 10}
    Pattern: {type: string, pattern: "^[0-9]{3}$"}
    Integer: {type: integer}
    IntMin: {type: integer, minimum: 1}
    IntExclMin: {type: integer, exclusiveMinimum: 0}
    IntMax: {type: integer, maximum: -1}
    IntMultiple: {type: integer, minimum: 5, multipleOf: 4}
    Number: {type: number}
    NumberExclMin: {type: number, exclusiveMinimum: 0}
    NumberMin: {type: number, minimum: 2.5}
    Boolean: {type: boolean}
    Nullable: {type: [string, "null"]}
    Array: {type: array, items: {type: integer}}
    ArrayMin: {type: array, minItems: 3, items: {type: integer}}
    ArrayZero: {type: array, minItems: 0, items: {type: integer}}
    ArrayUnique: {type: array, minItems: 2, uniqueItems: true, items: {type: string}}
    Object:
      type: object
      properties:
        name: {type: string}
        nested: {type: object, properties: {flag: {type: boolean}}}
    Additional: {type: object, additionalProperties: {type: integer}}
    ReadWrite:
      type: object
      properties:
        id: {type: string, readOnly: true}
        password: {type: string, writeOnly: true}
        name: {type: string}
    AllOf:
      allOf:
        - {type: object, properties: {a: {type: string}}, required: [a]}
        - {type: object, properties: {b: {type: integer}}}
    AllOfScalar:
      allOf:
        - {type: string, enum: [A, B]}
    OneOf:
      oneOf:
        - {$ref: "#/components/schemas/Cat"}
        - {$ref: "#/components/schemas/Dog"}
      discriminator:
        propertyName: kind
        mapping: {cat: "#/components/schemas/Cat", dog: "#/components/schemas/Dog"}
    AnyOf:
      anyOf:
        - {type: integer}
        - {type: string}
    Cat: {type: object, properties: {kind: {type: string}, meow: {type: boolean}}}
    Dog: {type: object, properties: {kind: {type: string}, bark: {type: boolean}}}
    Node:
      type: object
      properties:
        name: {type: string}
        child: {$ref: "#/components/schemas/Node"}
`

func model(t *testing.T) *v3.Document {
	t.Helper()
	doc, err := libopenapi.NewDocument([]byte(spec31))
	if err != nil {
		t.Fatal(err)
	}
	m, err := doc.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	return &m.Model
}

func jsonOf(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestSchema(t *testing.T) {
	doc := model(t)
	tests := []struct {
		name string
		mode sample.Mode
		want string
	}{
		// 5.1 優先順位
		{"Example", sample.Response, `"ex"`},
		{"Examples31", sample.Response, `"first"`},
		{"Default", sample.Response, `"def"`},
		{"Const", sample.Response, `"fixed"`},
		{"Enum", sample.Response, `"a"`},
		// 5.2 format
		{"Date", sample.Response, `"2026-01-01"`},
		{"DateTime", sample.Response, `"2026-01-01T00:00:00Z"`},
		{"Time", sample.Response, `"00:00:00"`},
		{"Email", sample.Response, `"user@example.com"`},
		{"URI", sample.Response, `"https://example.com/"`},
		{"Hostname", sample.Response, `"example.com"`},
		{"IPv4", sample.Response, `"192.0.2.1"`},
		{"IPv6", sample.Response, `"2001:db8::1"`},
		{"UUID", sample.Response, `"00000000-0000-4000-8000-000000000001"`},
		{"Byte", sample.Response, `"c2FtcGxl"`},
		{"Binary", sample.Response, `"TODO: path/to/file"`},
		// 5.3 制約と型
		{"String", sample.Response, `"TODO"`},
		{"Pattern", sample.Response, `"TODO: pattern ^[0-9]{3}$"`},
		{"Integer", sample.Response, `0`},
		{"IntMin", sample.Response, `1`},
		{"IntExclMin", sample.Response, `1`},
		{"IntMax", sample.Response, `-1`},
		{"IntMultiple", sample.Response, `8`},
		{"Number", sample.Response, `0`},
		{"NumberExclMin", sample.Response, `1`},
		{"NumberMin", sample.Response, `2.5`},
		{"Boolean", sample.Response, `true`},
		{"Nullable", sample.Response, `"TODO"`},
		{"Array", sample.Response, `[0]`},
		{"ArrayMin", sample.Response, `[0,0,0]`},
		{"ArrayZero", sample.Response, `[]`},
		{"ArrayUnique", sample.Response, `["TODO-1","TODO-2"]`},
		{"Object", sample.Response, `{"name":"TODO","nested":{"flag":true}}`},
		{"Additional", sample.Response, `{"TODO":0}`},
		{"ReadWrite", sample.Response, `{"id":"TODO","name":"TODO"}`},
		{"ReadWrite", sample.Request, `{"name":"TODO","password":"TODO"}`},
		// 構成
		{"AllOf", sample.Response, `{"a":"TODO","b":0}`},
		{"AllOfScalar", sample.Response, `"A"`},
		{"OneOf", sample.Response, `{"kind":"cat","meow":true}`},
		{"AnyOf", sample.Response, `0`},
		{"Node", sample.Response, `{"child":null,"name":"TODO"}`}, // 循環は 2 回目で打ち切る
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxy := doc.Components.Schemas.GetOrZero(tt.name)
			if proxy == nil {
				t.Fatalf("schema %s not found", tt.name)
			}
			if got := jsonOf(t, sample.Schema(proxy, tt.mode)); got != tt.want {
				t.Errorf("Schema(%s) = %s, want %s", tt.name, got, tt.want)
			}
		})
	}
}

func TestParameterString(t *testing.T) {
	doc := model(t)
	item := doc.Paths.PathItems.GetOrZero("/items/{id}")
	params := map[string]*v3.Parameter{}
	for _, p := range append(item.Get.Parameters, item.Parameters...) {
		params[p.Name] = p
	}
	tests := []struct{ name, want string }{
		{"id", "B0001"},                  // parameter.example (path item のパラメータ)
		{"examples", "E1"},               // examples の最初
		{"schemaExample", "7"},           // schema.example
		{"withDefault", "false"},         // schema.default
		{"withEnum", "NOVEL"},            // enum の最初
		{"list", "a"},                    // 配列は要素の値
		{"page", "1"},                    // 制約 (minimum)
		{"plain", sample.Placeholder},    // 型による値
		{"noSchema", sample.Placeholder}, // 決まらない
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sample.ParameterString(params[tt.name]); got != tt.want {
				t.Errorf("ParameterString(%s) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestMediaType(t *testing.T) {
	doc := model(t)
	op := doc.Paths.PathItems.GetOrZero("/items/{id}").Get
	for status, want := range map[string]string{"200": `{"id":"X"}`, "201": `{"id":"first"}`} {
		resp := op.Responses.Codes.GetOrZero(status)
		if got := jsonOf(t, sample.MediaType(sample.JSONMediaType(resp.Content), sample.Response)); got != want {
			t.Errorf("MediaType(%s) = %s, want %s", status, got, want)
		}
	}
	if sample.MediaType(nil, sample.Response) != nil {
		t.Error("nil media type should be nil")
	}
}

func TestString(t *testing.T) {
	for _, tt := range []struct {
		in   any
		want string
	}{
		{nil, sample.Placeholder}, {"x", "x"}, {true, "true"}, {3, "3"}, {int64(4), "4"}, {1.5, "1.5"}, {float64(10), "10"},
	} {
		if got := sample.String(tt.in); got != tt.want {
			t.Errorf("String(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
