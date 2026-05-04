package generator

import (
	"regexp"
	"strings"
	"testing"

	"github.com/ramsesyok/oapi2wire/internal/model"
)

func ptr(s string) *string { return &s }

var sampleOp = model.ResolvedOperation{
	OperationID: "getUser",
	Method:      "GET",
	Path:        "/users/{id}",
	PathParams:  []string{"id"},
}

func TestBuildMapping_PathAndQuery(t *testing.T) {
	cs := model.CaseSpec{
		ID:          "getUser_detail_100",
		OperationID: "getUser",
		Priority:    10,
		Request: &model.RequestSpec{
			PathParams: map[string]model.StringMatcher{
				"id": {EqualTo: ptr("100")},
			},
			Query: map[string]model.StringMatcher{
				"mode": {EqualTo: ptr("detail")},
			},
		},
		Response: model.ResponseSpec{Status: 200, BodyFile: "getUser/getUser_detail_100.json"},
	}
	defaults := model.Defaults{Response: model.DefaultResponse{Headers: map[string]string{"Content-Type": "application/json"}}}

	m, err := BuildMapping(cs, sampleOp, defaults)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	uuidRE := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !uuidRE.MatchString(m.ID) {
		t.Errorf("expected id to be a UUID v4, got %s", m.ID)
	}
	if m.Request.Method != "GET" {
		t.Errorf("expected method GET, got %s", m.Request.Method)
	}
	if m.Request.URLPathTemplate != "/users/{id}" {
		t.Errorf("expected urlPathTemplate /users/{id}, got %s", m.Request.URLPathTemplate)
	}
	if m.Request.PathParameters["id"].EqualTo == nil || *m.Request.PathParameters["id"].EqualTo != "100" {
		t.Errorf("expected pathParameters.id.equalTo=100")
	}
	if m.Request.QueryParameters["mode"].EqualTo == nil || *m.Request.QueryParameters["mode"].EqualTo != "detail" {
		t.Errorf("expected queryParameters.mode.equalTo=detail")
	}
	if m.Response.Headers["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type header from defaults")
	}
}

func TestBuildMapping_EqualToJson(t *testing.T) {
	cs := model.CaseSpec{
		ID:          "createUser_default",
		OperationID: "createUser",
		Priority:    100,
		Request: &model.RequestSpec{
			Body: &model.BodyMatcher{
				EqualToJson: map[string]interface{}{"role": "admin"},
			},
		},
		Response: model.ResponseSpec{Status: 201, BodyFile: "createUser/createUser_default.json"},
	}
	op := model.ResolvedOperation{OperationID: "createUser", Method: "POST", Path: "/users"}

	m, err := BuildMapping(cs, op, model.Defaults{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(m.Request.BodyPatterns) != 1 {
		t.Fatalf("expected 1 bodyPattern, got %d", len(m.Request.BodyPatterns))
	}
	val, ok := m.Request.BodyPatterns[0]["equalToJson"]
	if !ok {
		t.Fatal("expected equalToJson in bodyPatterns")
	}
	jsonStr, ok := val.(string)
	if !ok {
		t.Fatalf("expected equalToJson to be a string, got %T", val)
	}
	if !strings.Contains(jsonStr, `"role"`) {
		t.Errorf("expected JSON string to contain role, got %s", jsonStr)
	}
}

func TestBuildMapping_MatchesJsonPath(t *testing.T) {
	cs := model.CaseSpec{
		ID:          "createUser_admin_error",
		OperationID: "createUser",
		Priority:    10,
		Request: &model.RequestSpec{
			Body: &model.BodyMatcher{
				MatchesJsonPath: []string{
					"$.role[?(@ == 'admin')]",
					"$.active[?(@ == true)]",
				},
			},
		},
		Response: model.ResponseSpec{Status: 403, BodyFile: "createUser/createUser_admin_error.json"},
	}
	op := model.ResolvedOperation{OperationID: "createUser", Method: "POST", Path: "/users"}

	m, err := BuildMapping(cs, op, model.Defaults{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(m.Request.BodyPatterns) != 2 {
		t.Fatalf("expected 2 bodyPatterns, got %d", len(m.Request.BodyPatterns))
	}
	for _, p := range m.Request.BodyPatterns {
		if _, ok := p["matchesJsonPath"]; !ok {
			t.Error("expected matchesJsonPath in bodyPattern")
		}
	}
}

func TestBuildMapping_BothBodyMatchers(t *testing.T) {
	cs := model.CaseSpec{
		ID:          "c1",
		OperationID: "createUser",
		Request: &model.RequestSpec{
			Body: &model.BodyMatcher{
				EqualToJson:     map[string]interface{}{"type": "admin"},
				MatchesJsonPath: []string{"$.options.mode[?(@ == 'detail')]"},
			},
		},
		Response: model.ResponseSpec{Status: 200, BodyFile: "x/x.json"},
	}
	op := model.ResolvedOperation{OperationID: "createUser", Method: "POST", Path: "/users"}

	m, err := BuildMapping(cs, op, model.Defaults{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(m.Request.BodyPatterns) != 2 {
		t.Fatalf("expected 2 bodyPatterns (1 equalToJson + 1 matchesJsonPath), got %d", len(m.Request.BodyPatterns))
	}
}

func TestMergeHeaders(t *testing.T) {
	defaults := map[string]string{"Content-Type": "application/json", "X-Default": "yes"}
	caseH := map[string]string{"Content-Type": "text/plain", "X-Case": "yes"}

	merged := mergeHeaders(defaults, caseH)
	if merged["Content-Type"] != "text/plain" {
		t.Error("expected case header to override default")
	}
	if merged["X-Default"] != "yes" {
		t.Error("expected default header to be preserved")
	}
	if merged["X-Case"] != "yes" {
		t.Error("expected case header to be present")
	}
}

func TestMappingFileName(t *testing.T) {
	got := MappingFileName("getUser", "getUser_detail_100")
	want := "getUser__getUser_detail_100.json"
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestMappingFileName_SanitizesPathSeparators(t *testing.T) {
	got := MappingFileName("../admin/getUser", `..\evil\case`)
	want := "admin_getUser__evil_case.json"
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}
