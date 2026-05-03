package generator

import (
	"strings"
	"testing"

	"github.com/ramsesyok/oapi2wire/internal/model"
)

func TestNeedsAutoFallback(t *testing.T) {
	tests := []struct {
		name        string
		operationID string
		cases       []model.CaseSpec
		want        bool
	}{
		{
			name:        "no fallback",
			operationID: "getUser",
			cases: []model.CaseSpec{
				{ID: "c1", OperationID: "getUser", Fallback: false},
			},
			want: true,
		},
		{
			name:        "explicit fallback present",
			operationID: "getUser",
			cases: []model.CaseSpec{
				{ID: "c1", OperationID: "getUser", Fallback: false},
				{ID: "c2", OperationID: "getUser", Fallback: true},
			},
			want: false,
		},
		{
			name:        "fallback for different op",
			operationID: "getUser",
			cases: []model.CaseSpec{
				{ID: "c1", OperationID: "createUser", Fallback: true},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NeedsAutoFallback(tt.operationID, tt.cases)
			if got != tt.want {
				t.Errorf("NeedsAutoFallback(%q) = %v, want %v", tt.operationID, got, tt.want)
			}
		})
	}
}

func TestBuildFallback(t *testing.T) {
	op := model.ResolvedOperation{
		OperationID: "getUser",
		Method:      "GET",
		Path:        "/users/{id}",
	}

	result := BuildFallback(op)

	if result.MappingFileName != "_generated__fallback__getUser.json" {
		t.Errorf("unexpected MappingFileName: %s", result.MappingFileName)
	}
	if result.BodyFileName != "_generated/fallback/getUser.json" {
		t.Errorf("unexpected BodyFileName: %s", result.BodyFileName)
	}
	if result.Mapping.Response.Status != 501 {
		t.Errorf("expected status 501, got %d", result.Mapping.Response.Status)
	}
	if result.Mapping.Priority != autoFallbackPriority {
		t.Errorf("expected priority %d, got %d", autoFallbackPriority, result.Mapping.Priority)
	}
	if !strings.Contains(string(result.BodyContent), "no mock case matched") {
		t.Error("expected body to contain 'no mock case matched'")
	}
	if result.Mapping.Request.Method != "GET" {
		t.Errorf("expected method GET, got %s", result.Mapping.Request.Method)
	}
	if result.Mapping.Request.URLPathTemplate != "/users/{id}" {
		t.Errorf("expected urlPathTemplate /users/{id}, got %s", result.Mapping.Request.URLPathTemplate)
	}
}

func TestBuildFallback_SanitizesOperationIDInFileNames(t *testing.T) {
	op := model.ResolvedOperation{
		OperationID: "../admin/getUser",
		Method:      "GET",
		Path:        "/users/{id}",
	}

	result := BuildFallback(op)

	if result.MappingFileName != "_generated__fallback__admin_getUser.json" {
		t.Errorf("unexpected MappingFileName: %s", result.MappingFileName)
	}
	if result.BodyFileName != "_generated/fallback/admin_getUser.json" {
		t.Errorf("unexpected BodyFileName: %s", result.BodyFileName)
	}
	if result.Mapping.Metadata.OperationID != "../admin/getUser" {
		t.Errorf("metadata operationId should keep original value, got %s", result.Mapping.Metadata.OperationID)
	}
}
