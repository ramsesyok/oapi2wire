/*
Copyright © 2026 oapi2wire authors
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/ramsesyok/oapi2wire/internal/cases"
	"github.com/ramsesyok/oapi2wire/internal/model"
	"github.com/ramsesyok/oapi2wire/internal/openapi"
	"github.com/spf13/cobra"
)

var (
	validateOpenAPI       string
	validateCases         string
	validateResponsesRoot string
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate OpenAPI and case YAML consistency",
	Long:  `Check consistency between the OpenAPI definition and the case YAML file.`,
	RunE:  runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)
	validateCmd.Flags().StringVar(&validateOpenAPI, "openapi", "", "Path to OpenAPI file (required)")
	validateCmd.Flags().StringVar(&validateCases, "cases", "mock-cases.yaml", "Path to case YAML file")
	validateCmd.Flags().StringVar(&validateResponsesRoot, "responses-root", "mock-responses", "Responses root directory")
	if err := validateCmd.MarkFlagRequired("openapi"); err != nil {
		panic(err)
	}
}

func runValidate(cmd *cobra.Command, args []string) error {
	// Load OpenAPI
	doc, err := openapi.Load(validateOpenAPI)
	if err != nil {
		return fmt.Errorf("loading OpenAPI: %w", err)
	}

	// Build operation index
	ops, diags := openapi.BuildOperationIndex(doc)
	if model.HasErrors(diags) {
		printDiags(diags)
		return fmt.Errorf("OpenAPI has errors")
	}

	// Load case YAML
	cf, err := cases.Load(validateCases)
	if err != nil {
		return fmt.Errorf("loading case YAML: %w", err)
	}

	// Validate
	diags = append(diags, cases.Validate(cf, ops, cases.ValidateConfig{
		ResponsesRoot: validateResponsesRoot,
	})...)

	printDiags(diags)

	if model.HasErrors(diags) {
		return fmt.Errorf("validation failed with errors")
	}

	if len(diags) == 0 {
		fmt.Println("OK: no issues found")
	}
	return nil
}

func printDiags(diags []model.Diagnostic) {
	for _, d := range diags {
		fmt.Fprintln(os.Stderr, d.String())
	}
}
