package generator

import (
	"encoding/json"
	"fmt"

	"github.com/ramsesyok/oapi2wire/internal/model"
)

const autoFallbackPriority = 1000000

// FallbackResult holds one auto-generated fallback mapping + its body content.
type FallbackResult struct {
	MappingFileName string
	Mapping         model.WireMockMapping
	BodyFileName    string // relative path inside __files/
	BodyContent     []byte // JSON bytes
}

// NeedsAutoFallback returns true if no case in cases is an explicit fallback for operationID.
func NeedsAutoFallback(operationID string, cases []model.CaseSpec) bool {
	for _, c := range cases {
		if c.OperationID == operationID && c.Fallback {
			return false
		}
	}
	return true
}

// BuildFallback creates the auto-generated fallback for one operation.
func BuildFallback(op model.ResolvedOperation) FallbackResult {
	safeOperationID := safeFileNamePart(op.OperationID)
	mappingFile := fmt.Sprintf("_generated__fallback__%s.json", safeOperationID)
	bodyFile := fmt.Sprintf("_generated/fallback/%s.json", safeOperationID)

	body := map[string]interface{}{
		"message":     "no mock case matched",
		"operationId": op.OperationID,
		"method":      op.Method,
		"path":        op.Path,
	}
	bodyBytes, _ := json.MarshalIndent(body, "", "  ")
	bodyBytes = append(bodyBytes, '\n')

	mapping := model.WireMockMapping{
		ID:       newUUID(),
		Name:     fmt.Sprintf("_generated_fallback_%s", op.OperationID),
		Priority: autoFallbackPriority,
		Metadata: model.WireMockMeta{
			Generator:   "oapi2wire",
			OperationID: op.OperationID,
			CaseID:      fmt.Sprintf("_generated_fallback_%s", op.OperationID),
		},
		Request: model.WireMockRequest{
			Method:          op.Method,
			URLPathTemplate: op.Path,
		},
		Response: model.WireMockResponse{
			Status: 501,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			BodyFileName: bodyFile,
		},
	}

	return FallbackResult{
		MappingFileName: mappingFile,
		Mapping:         mapping,
		BodyFileName:    bodyFile,
		BodyContent:     bodyBytes,
	}
}
