package util

import (
	"errors"
	"strings"
)

// ValidateBodyFilePath checks that p is a safe relative path.
// Returns an error if p is empty, absolute, contains .., or contains backslash.
func ValidateBodyFilePath(p string) error {
	if p == "" {
		return errors.New("bodyFile path must not be empty")
	}
	if strings.HasPrefix(p, "/") {
		return errors.New("bodyFile must not be an absolute path")
	}
	if strings.Contains(p, "..") {
		return errors.New("bodyFile must not contain '..'")
	}
	if strings.Contains(p, "\\") {
		return errors.New("bodyFile must not contain backslash")
	}
	return nil
}

// IsSafeRelative is the boolean form of ValidateBodyFilePath.
func IsSafeRelative(p string) bool {
	return ValidateBodyFilePath(p) == nil
}
