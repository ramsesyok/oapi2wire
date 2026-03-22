package util

import "testing"

func TestValidateBodyFilePath(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"getUser/foo.json", false},
		{"a/b/c.json", false},
		{"simple.json", false},
		{"", true},
		{"/absolute/path.json", true},
		{"../escape.json", true},
		{"a/../b.json", true},
		{"path\\with\\backslash.json", true},
		{"a/b/../c.json", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := ValidateBodyFilePath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBodyFilePath(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
