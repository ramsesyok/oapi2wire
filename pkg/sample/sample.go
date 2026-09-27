// Package sample は OpenAPI の定義から、テストやモックの雛形に入れる値 (サンプル値) を作る。
//
// oapi2wire init (モックの case YAML と応答 JSON) と runnora generate (テストケース) の両方がこの
// パッケージを使う。同じ OpenAPI から作ったフロント向けのモックと API のテストで、
// パラメータ・リクエスト本文・応答本文の初期値を一致させるため。
//
// 規則は runnora の docs/design/sample-generation.md の 5 章。
//
//   - 本文: media type の example → examples の最初 → スキーマの値
//   - パラメータ: parameter の example → examples の最初 → スキーマの値 (配列なら要素の値)
//   - スキーマの値: example → examples の最初 (3.1) → default → const → enum の最初
//     → allOf の合成 / oneOf・anyOf の最初 → format の値 → 制約と型の値
//
// まだ入れていないもの: pattern に一致する文字列の生成 (5.4。いまは "TODO: pattern <pattern>")、
// 仮の値の一覧 (5.5)、浮動小数の書き出し (5.6)。
package sample

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

// Placeholder は値を決められない文字列に入れる目印。
const Placeholder = "TODO"

// Mode はリクエストとレスポンスのどちらの値を作るか。readOnly / writeOnly のプロパティの扱いが変わる。
type Mode int

const (
	// Response は応答の値 (writeOnly のプロパティを含めない)。
	Response Mode = iota
	// Request はリクエストの値 (readOnly のプロパティを含めない)。
	Request
)

// formatValues は format ごとの値 (5.2)。乱数は使わず毎回同じ値にする。
var formatValues = map[string]string{
	"date":      "2026-01-01",
	"date-time": "2026-01-01T00:00:00Z",
	"time":      "00:00:00",
	"email":     "user@example.com",
	"uri":       "https://example.com/",
	"url":       "https://example.com/",
	"hostname":  "example.com",
	"ipv4":      "192.0.2.1",
	"ipv6":      "2001:db8::1",
	"uuid":      "00000000-0000-4000-8000-000000000001",
	"byte":      "c2FtcGxl",
	"binary":    "TODO: path/to/file",
}

// FormatValue は format の値を返す。format に決まった値がなければ ""。
func FormatValue(format string) string {
	return formatValues[format]
}

// maxDepth はスキーマをたどる深さの上限。
const maxDepth = 10

// Schema はスキーマの値を返す。決まらなければ nil。
func Schema(proxy *base.SchemaProxy, mode Mode) any {
	return newGen(mode).proxy(proxy, 0)
}

// SchemaOf は構築済みのスキーマの値を返す。決まらなければ nil。
func SchemaOf(s *base.Schema, mode Mode) any {
	return newGen(mode).schema(s, 0)
}

// MediaType は本文の値を返す (example → examples の最初 → スキーマの値)。決まらなければ nil。
func MediaType(mt *v3.MediaType, mode Mode) any {
	if mt == nil {
		return nil
	}
	if v := nodeValue(mt.Example); v != nil {
		return v
	}
	if v := firstExample(mt.Examples); v != nil {
		return v
	}
	return Schema(mt.Schema, mode)
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
	g := newGen(Request)
	if schemaType(s) == "array" && s.Items != nil && s.Items.IsA() {
		if v := g.proxy(s.Items.A, 1); v != nil {
			return v
		}
	}
	return g.schema(s, 0)
}

// ParameterString はパラメータの値を、URL やモックの一致条件に書く文字列にする。決まらなければ Placeholder。
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
		if strings.Contains(pair.Key, "application/json") || strings.HasSuffix(pair.Key, "+json") {
			return pair.Value
		}
	}
	return nil
}

type gen struct {
	mode Mode
	// path はたどっている途中のスキーマ (定義の YAML ノード)。同じスキーマが経路上に 2 回目に現れたら
	// 打ち切る ($ref の循環の対策)。
	path map[*yaml.Node]bool
}

func newGen(mode Mode) *gen {
	return &gen{mode: mode, path: map[*yaml.Node]bool{}}
}

func (g *gen) proxy(p *base.SchemaProxy, depth int) any {
	if p == nil {
		return nil
	}
	s, err := p.BuildSchema()
	if err != nil || s == nil {
		return nil
	}
	return g.schema(s, depth)
}

func (g *gen) schema(s *base.Schema, depth int) any {
	if s == nil || depth > maxDepth {
		return nil
	}
	if low := s.GoLow(); low != nil && low.RootNode != nil {
		if g.path[low.RootNode] {
			return nil
		}
		g.path[low.RootNode] = true
		defer delete(g.path, low.RootNode)
	}
	if v := nodeValue(s.Example); v != nil {
		return v
	}
	if len(s.Examples) > 0 {
		if v := nodeValue(s.Examples[0]); v != nil {
			return v
		}
	}
	if v := nodeValue(s.Default); v != nil {
		return v
	}
	if v := nodeValue(s.Const); v != nil {
		return v
	}
	for _, e := range s.Enum {
		if v := nodeValue(e); v != nil {
			return v
		}
	}
	if len(s.AllOf) > 0 {
		return g.allOf(s, depth)
	}
	if len(s.OneOf) > 0 {
		return g.choice(s, s.OneOf, depth)
	}
	if len(s.AnyOf) > 0 {
		return g.choice(s, s.AnyOf, depth)
	}

	switch schemaType(s) {
	case "string":
		return stringValue(s)
	case "integer":
		return integerValue(s)
	case "number":
		return numberValue(s)
	case "boolean":
		return true
	case "array":
		return g.array(s, depth)
	case "object":
		return g.object(s, depth)
	case "null":
		return nil
	case "":
		switch {
		case s.Properties != nil && s.Properties.Len() > 0, s.AdditionalProperties != nil:
			return g.object(s, depth)
		case s.Items != nil:
			return g.array(s, depth)
		case s.Format != "":
			return stringValue(s)
		}
	}
	return nil
}

// allOf はすべての schema の properties と required を合成してから値を作る。
// どの schema も properties を持たなければ (値の制約を重ねただけなど)、最初の schema の値を使う。
func (g *gen) allOf(s *base.Schema, depth int) any {
	props := orderedmap.New[string, *base.SchemaProxy]()
	var add func(*base.Schema)
	add = func(sub *base.Schema) {
		if sub == nil || sub.Properties == nil {
			return
		}
		for pair := sub.Properties.Oldest(); pair != nil; pair = pair.Next() {
			props.Set(pair.Key, pair.Value)
		}
	}
	var first *base.SchemaProxy
	for _, p := range s.AllOf {
		if p == nil {
			continue
		}
		if first == nil {
			first = p
		}
		sub, err := p.BuildSchema()
		if err != nil || sub == nil {
			continue
		}
		add(sub)
		for _, nested := range sub.AllOf { // allOf の入れ子も合成する
			if nested == nil {
				continue
			}
			if ns, err := nested.BuildSchema(); err == nil {
				add(ns)
			}
		}
	}
	add(s)
	if props.Len() == 0 {
		return g.proxy(first, depth+1)
	}
	return g.properties(props, depth)
}

// choice は oneOf / anyOf の最初の選択肢の値を作る。discriminator があれば、最初の mapping の値を判別プロパティに入れる。
func (g *gen) choice(s *base.Schema, options []*base.SchemaProxy, depth int) any {
	if options[0] == nil {
		return nil
	}
	v := g.proxy(options[0], depth+1)
	if d := s.Discriminator; d != nil && d.PropertyName != "" && d.Mapping != nil && d.Mapping.Len() > 0 {
		if obj, ok := v.(map[string]any); ok {
			obj[d.PropertyName] = d.Mapping.Oldest().Key
		}
	}
	return v
}

func (g *gen) object(s *base.Schema, depth int) map[string]any {
	if s.Properties != nil && s.Properties.Len() > 0 {
		return g.properties(s.Properties, depth)
	}
	// additionalProperties だけの object は {"TODO": <値>}
	if ap := s.AdditionalProperties; ap != nil {
		if ap.IsA() && ap.A != nil {
			return map[string]any{Placeholder: g.proxy(ap.A, depth+1)}
		}
		if ap.IsB() && ap.B {
			return map[string]any{Placeholder: Placeholder}
		}
	}
	return map[string]any{}
}

func (g *gen) properties(props *orderedmap.Map[string, *base.SchemaProxy], depth int) map[string]any {
	obj := map[string]any{}
	for pair := props.Oldest(); pair != nil; pair = pair.Next() {
		if pair.Value == nil {
			obj[pair.Key] = Placeholder
			continue
		}
		prop, err := pair.Value.BuildSchema()
		if err != nil || prop == nil {
			obj[pair.Key] = Placeholder
			continue
		}
		if g.mode == Request && isTrue(prop.ReadOnly) || g.mode == Response && isTrue(prop.WriteOnly) {
			continue
		}
		obj[pair.Key] = g.proxy(pair.Value, depth+1)
	}
	return obj
}

func (g *gen) array(s *base.Schema, depth int) []any {
	n := 1
	if s.MinItems != nil {
		n = int(*s.MinItems)
	}
	if n > 100 {
		n = 100
	}
	if s.Items == nil || !s.Items.IsA() || s.Items.A == nil || n <= 0 {
		return []any{}
	}
	item := g.proxy(s.Items.A, depth+1)
	out := make([]any, n)
	for i := range out {
		out[i] = item
		// uniqueItems で要素が文字列なら、番号を付けて重複を避ける ("TODO-1", "TODO-2")
		if str, ok := item.(string); ok && isTrue(s.UniqueItems) && n > 1 {
			out[i] = fmt.Sprintf("%s-%d", str, i+1)
		}
	}
	return out
}

// stringValue は string の値 (format → pattern → "TODO")。
func stringValue(s *base.Schema) string {
	if v := formatValues[s.Format]; v != "" {
		return v
	}
	if s.Pattern != "" {
		return Placeholder + ": pattern " + s.Pattern
	}
	return Placeholder
}

// bounds は minimum / maximum と exclusive の指定を返す。
func bounds(s *base.Schema) (lo *float64, loExcl bool, hi *float64, hiExcl bool) {
	lo, hi = s.Minimum, s.Maximum
	if e := s.ExclusiveMinimum; e != nil {
		if e.IsB() { // 3.1: exclusiveMinimum が数値
			v := e.B
			lo, loExcl = &v, true
		} else if e.A && lo != nil { // 3.0: exclusiveMinimum: true
			loExcl = true
		}
	}
	if e := s.ExclusiveMaximum; e != nil {
		if e.IsB() {
			v := e.B
			hi, hiExcl = &v, true
		} else if e.A && hi != nil {
			hiExcl = true
		}
	}
	return lo, loExcl, hi, hiExcl
}

// integerValue は 0。範囲が 0 を許さなければ 0 に最も近い許される値 (multipleOf も満たす)。
func integerValue(s *base.Schema) int {
	lo, loExcl, hi, hiExcl := bounds(s)
	v := 0.0
	if lo != nil {
		min := math.Ceil(*lo)
		if loExcl {
			min = math.Floor(*lo) + 1
		}
		if v < min {
			v = min
		}
	}
	if hi != nil {
		max := math.Floor(*hi)
		if hiExcl {
			max = math.Ceil(*hi) - 1
		}
		if v > max {
			v = max
		}
	}
	if m := s.MultipleOf; m != nil && *m > 0 && v != 0 {
		if v > 0 {
			v = math.Ceil(v / *m) * *m
		} else {
			v = math.Floor(v / *m) * *m
		}
	}
	return int(v)
}

// numberValue は 0.0。範囲が 0 を許さなければ境界の値 (exclusive なら境界 + 1 / - 1)。
func numberValue(s *base.Schema) float64 {
	lo, loExcl, hi, hiExcl := bounds(s)
	v := 0.0
	if lo != nil && (v < *lo || loExcl && v <= *lo) {
		v = *lo
		if loExcl {
			v = *lo + 1
		}
	}
	if hi != nil && (v > *hi || hiExcl && v >= *hi) {
		v = *hi
		if hiExcl {
			v = *hi - 1
		}
	}
	if m := s.MultipleOf; m != nil && *m > 0 && v != 0 {
		if v > 0 {
			v = math.Ceil(v / *m) * *m
		} else {
			v = math.Floor(v / *m) * *m
		}
	}
	return v
}

// schemaType は type の最初の要素を返す。null を許す型 (type: [string, "null"]) は null でない方。
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

func isTrue(b *bool) bool {
	return b != nil && *b
}

func firstExample(examples *orderedmap.Map[string, *base.Example]) any {
	if examples == nil || examples.Len() == 0 {
		return nil
	}
	if ex := examples.Oldest().Value; ex != nil {
		return nodeValue(ex.Value)
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
