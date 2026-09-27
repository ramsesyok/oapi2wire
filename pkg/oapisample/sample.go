// Package oapisample は OpenAPI 定義から、テストやモックの雛形に入れる値 (サンプル値) を決める。
//
// oapi2wire init (モックの case YAML と応答 JSON) と runnora generate (テストケース) の両方がこの
// パッケージを使う。同じ OpenAPI からフロントのモックと API のテストを作ったときに、
// パラメータ・リクエスト本文・応答本文の初期値が一致するようにするため。
//
// 値の決め方 (上から順に、最初に見つかったもの):
//
//   - パラメータ: parameter.example → parameter.examples の最初 → スキーマの値 (配列なら要素の値)
//   - 本文 (リクエスト・応答): media type の example → examples の最初 → スキーマの値
//   - スキーマの値: example → default → enum の最初 → allOf / anyOf / oneOf の最初 → 型による値
//
// 型による値:
//
//	string   "TODO" (format が date / date-time / uuid ならその形の値)
//	integer  0
//	number   0
//	boolean  true
//	array    要素 1 つの配列
//	object   properties を再帰的に展開したオブジェクト
package oapisample

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

// Placeholder は値が決まらない文字列に入れる値。雛形を編集するときの目印になる。
const Placeholder = "TODO"

// Format ごとの値。
const (
	DateSample     = "2006-01-02"
	DateTimeSample = "2006-01-02T15:04:05Z"
	UUIDSample     = "00000000-0000-0000-0000-000000000000"
)

// maxDepth はスキーマをたどる深さの上限 (循環参照の対策)。
const maxDepth = 10

// Schema はスキーマの値を返す。決まらなければ nil。
func Schema(proxy *base.SchemaProxy) any {
	if proxy == nil {
		return nil
	}
	s, err := proxy.BuildSchema()
	if err != nil || s == nil {
		return nil
	}
	return fromSchema(s, 0)
}

// SchemaOf は構築済みのスキーマの値を返す。決まらなければ nil。
func SchemaOf(s *base.Schema) any {
	return fromSchema(s, 0)
}

// MediaType は本文の値を返す (example → examples の最初 → スキーマの値)。決まらなければ nil。
func MediaType(mt *v3.MediaType) any {
	if mt == nil {
		return nil
	}
	if v := nodeValue(mt.Example); v != nil {
		return v
	}
	if v := firstExample(mt.Examples); v != nil {
		return v
	}
	return Schema(mt.Schema)
}

// Parameter はパラメータの値を返す (example → examples の最初 → スキーマの値)。
// 配列のパラメータは要素の値を返す。決まらなければ nil。
func Parameter(p *v3.Parameter) any {
	if p == nil {
		return nil
	}
	if v := nodeValue(p.Example); v != nil {
		return v
	}
	if v := firstExample(p.Examples); v != nil {
		return v
	}
	if p.Schema == nil {
		return nil
	}
	s, err := p.Schema.BuildSchema()
	if err != nil || s == nil {
		return nil
	}
	if schemaType(s) == "array" {
		if item := itemSchema(s); item != nil {
			if v := fromSchema(item, 1); v != nil {
				return v
			}
		}
	}
	return fromSchema(s, 0)
}

// ParameterString はパラメータの値を、URL やモックの一致条件に書く文字列にする。
// 値が決まらなければ Placeholder。
func ParameterString(p *v3.Parameter) string {
	return String(Parameter(p))
}

// String は値を URL やモックの一致条件に書く文字列にする。nil は Placeholder。
func String(v any) string {
	switch val := v.(type) {
	case nil:
		return Placeholder
	case string:
		return val
	case bool:
		return strconv.FormatBool(val)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	default:
		return fmt.Sprint(val)
	}
}

// JSONMediaType は content のうち application/json (…+json を含む) の media type を返す。なければ nil。
func JSONMediaType(content *orderedmap.Map[string, *v3.MediaType]) *v3.MediaType {
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

func fromSchema(s *base.Schema, depth int) any {
	if s == nil || depth > maxDepth {
		return nil
	}
	if v := nodeValue(s.Example); v != nil {
		return v
	}
	if v := nodeValue(s.Default); v != nil {
		return v
	}
	if len(s.Enum) > 0 {
		if v := nodeValue(s.Enum[0]); v != nil {
			return v
		}
	}
	for _, group := range [][]*base.SchemaProxy{s.AllOf, s.AnyOf, s.OneOf} {
		if len(group) > 0 && group[0] != nil {
			if sub, err := group[0].BuildSchema(); err == nil && sub != nil {
				return fromSchema(sub, depth+1)
			}
		}
	}

	switch schemaType(s) {
	case "object":
		return object(s, depth)
	case "array":
		return array(s, depth)
	case "string":
		switch s.Format {
		case "date":
			return DateSample
		case "date-time":
			return DateTimeSample
		case "uuid":
			return UUIDSample
		}
		return Placeholder
	case "integer", "number":
		return 0
	case "boolean":
		return true
	case "null":
		return nil
	case "":
		if s.Properties != nil && s.Properties.Len() > 0 {
			return object(s, depth)
		}
		if s.Items != nil {
			return array(s, depth)
		}
	}
	return nil
}

func object(s *base.Schema, depth int) map[string]any {
	obj := map[string]any{}
	if s.Properties == nil {
		return obj
	}
	for pair := s.Properties.Oldest(); pair != nil; pair = pair.Next() {
		if pair.Value == nil {
			obj[pair.Key] = Placeholder
			continue
		}
		prop, err := pair.Value.BuildSchema()
		if err != nil || prop == nil {
			obj[pair.Key] = Placeholder
			continue
		}
		obj[pair.Key] = fromSchema(prop, depth+1)
	}
	return obj
}

func array(s *base.Schema, depth int) []any {
	item := itemSchema(s)
	if item == nil {
		return []any{}
	}
	return []any{fromSchema(item, depth+1)}
}

func itemSchema(s *base.Schema) *base.Schema {
	if s.Items == nil || !s.Items.IsA() || s.Items.A == nil {
		return nil
	}
	item, err := s.Items.A.BuildSchema()
	if err != nil {
		return nil
	}
	return item
}

// schemaType は type の最初の要素を返す (OpenAPI 3.1 では配列で書ける)。null は後回しにする。
func schemaType(s *base.Schema) string {
	for _, t := range s.Type {
		if t != "null" {
			return t
		}
	}
	if len(s.Type) > 0 {
		return s.Type[0]
	}
	return ""
}

func firstExample(examples *orderedmap.Map[string, *base.Example]) any {
	if examples == nil {
		return nil
	}
	for pair := examples.Oldest(); pair != nil; pair = pair.Next() {
		if pair.Value != nil {
			if v := nodeValue(pair.Value.Value); v != nil {
				return v
			}
		}
		return nil
	}
	return nil
}

func nodeValue(node *yaml.Node) any {
	if node == nil {
		return nil
	}
	var v any
	if err := node.Decode(&v); err != nil {
		return nil
	}
	return v
}
