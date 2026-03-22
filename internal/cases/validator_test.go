package cases

import (
	"testing"

	"github.com/ramsesyok/oapi2wire/internal/model"
)

func makeOps(opIDs ...string) map[string]model.ResolvedOperation {
	ops := make(map[string]model.ResolvedOperation)
	for _, id := range opIDs {
		ops[id] = model.ResolvedOperation{
			OperationID: id,
			Method:      "GET",
			Path:        "/test",
		}
	}
	return ops
}

func ptr(s string) *string { return &s }

func TestCheckDuplicateCaseIDs(t *testing.T) {
	cases := []model.CaseSpec{
		{ID: "a", OperationID: "op1", Response: model.ResponseSpec{Status: 200, BodyFile: "a/a.json"}},
		{ID: "b", OperationID: "op1", Response: model.ResponseSpec{Status: 200, BodyFile: "b/b.json"}},
		{ID: "a", OperationID: "op2", Response: model.ResponseSpec{Status: 200, BodyFile: "a2/a2.json"}},
	}
	diags := checkDuplicateCaseIDs(cases)
	if !model.HasErrors(diags) {
		t.Error("expected error for duplicate case id")
	}
}

func TestCheckOperationIDsExist_Error(t *testing.T) {
	ops := makeOps("getUser")
	cases := []model.CaseSpec{
		{ID: "c1", OperationID: "nonExistent", Response: model.ResponseSpec{Status: 200, BodyFile: "x/x.json"}},
	}
	diags := checkOperationIDsExist(cases, ops, true)
	if !model.HasErrors(diags) {
		t.Error("expected error when FailOnMissingOperation=true")
	}
}

func TestCheckOperationIDsExist_Warning(t *testing.T) {
	ops := makeOps("getUser")
	cases := []model.CaseSpec{
		{ID: "c1", OperationID: "nonExistent", Response: model.ResponseSpec{Status: 200, BodyFile: "x/x.json"}},
	}
	diags := checkOperationIDsExist(cases, ops, false)
	if model.HasErrors(diags) {
		t.Error("expected warning (not error) when FailOnMissingOperation=false")
	}
	if len(diags) == 0 {
		t.Error("expected at least one diagnostic")
	}
}

func TestCheckFallbackRules_MultipleFallbacks(t *testing.T) {
	cases := []model.CaseSpec{
		{ID: "f1", OperationID: "getUser", Fallback: true, Response: model.ResponseSpec{Status: 501, BodyFile: "fb/f1.json"}},
		{ID: "f2", OperationID: "getUser", Fallback: true, Response: model.ResponseSpec{Status: 501, BodyFile: "fb/f2.json"}},
	}
	diags := checkFallbackRules(cases)
	if !model.HasErrors(diags) {
		t.Error("expected error for multiple fallbacks on same operationId")
	}
}

func TestCheckFallbackRules_FallbackWithRequest(t *testing.T) {
	cases := []model.CaseSpec{
		{
			ID:          "f1",
			OperationID: "getUser",
			Fallback:    true,
			Request:     &model.RequestSpec{PathParams: map[string]model.StringMatcher{"id": {EqualTo: ptr("1")}}},
			Response:    model.ResponseSpec{Status: 501, BodyFile: "fb/f1.json"},
		},
	}
	diags := checkFallbackRules(cases)
	if !model.HasErrors(diags) {
		t.Error("expected error for fallback case with request")
	}
}

func TestCheckPathParamNames(t *testing.T) {
	ops := map[string]model.ResolvedOperation{
		"getUser": {OperationID: "getUser", Method: "GET", Path: "/users/{id}", PathParams: []string{"id"}},
	}
	cases := []model.CaseSpec{
		{
			ID:          "c1",
			OperationID: "getUser",
			Request: &model.RequestSpec{
				PathParams: map[string]model.StringMatcher{
					"badParam": {EqualTo: ptr("1")},
				},
			},
			Response: model.ResponseSpec{Status: 200, BodyFile: "g/g.json"},
		},
	}
	diags := checkPathParamNames(cases, ops)
	if !model.HasErrors(diags) {
		t.Error("expected error for invalid path param name")
	}
}

func TestCheckBodyFilePaths_ForbiddenPath(t *testing.T) {
	cases := []model.CaseSpec{
		{ID: "c1", OperationID: "op", Response: model.ResponseSpec{Status: 200, BodyFile: "../escape.json"}},
	}
	diags := checkBodyFilePaths(cases, "", false)
	if !model.HasErrors(diags) {
		t.Error("expected error for forbidden path")
	}
}

func TestCheckEmptyMatchers(t *testing.T) {
	cases := []model.CaseSpec{
		{ID: "c1", OperationID: "getUser", Fallback: false, Request: nil,
			Response: model.ResponseSpec{Status: 200, BodyFile: "x/x.json"}},
	}
	diags := checkEmptyMatchers(cases)
	if len(diags) == 0 {
		t.Error("expected warning for non-fallback case with no matchers")
	}
	if diags[0].Severity != model.SeverityWarning {
		t.Error("expected warning severity")
	}
}

func TestValidate_Integration(t *testing.T) {
	ops := map[string]model.ResolvedOperation{
		"getUser": {
			OperationID: "getUser", Method: "GET", Path: "/users/{id}",
			PathParams: []string{"id"},
		},
	}
	cf := &model.CaseFile{
		Version: 1,
		Cases: []model.CaseSpec{
			{
				ID:          "getUser_default",
				OperationID: "getUser",
				Priority:    100,
				Request: &model.RequestSpec{
					PathParams: map[string]model.StringMatcher{
						"id": {EqualTo: ptr("100")},
					},
				},
				Response: model.ResponseSpec{Status: 200, BodyFile: "getUser/getUser_default.json"},
			},
		},
	}
	diags := Validate(cf, ops, ValidateConfig{})
	if model.HasErrors(diags) {
		for _, d := range diags {
			t.Logf("diag: %s", d)
		}
		t.Error("expected no errors for valid case file")
	}
}
