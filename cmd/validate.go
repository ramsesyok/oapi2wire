/*
Copyright © 2026 oapi2wire authors
*/
package cmd

import (
	"fmt"
	"os"

	lib "github.com/ramsesyok/oapi2wire/pkg/oapi2wire"
	"github.com/spf13/cobra"
)

var (
	validateOpenAPI       string
	validateCases         string
	validateResponsesRoot string
	validateTags          []string
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
	validateCmd.Flags().StringSliceVar(&validateTags, "tags", nil, "OpenAPI operation tags to include")
	if err := validateCmd.MarkFlagRequired("openapi"); err != nil {
		panic(err)
	}
}

func runValidate(cmd *cobra.Command, args []string) error {
	result, err := lib.Validate(lib.ValidateOptions{
		OpenAPIPath:   validateOpenAPI,
		CasesPath:     validateCases,
		ResponsesRoot: validateResponsesRoot,
		Tags:          validateTags,
	})
	printDiags(result.Diagnostics)
	if err != nil {
		return err
	}

	if len(result.Diagnostics) == 0 {
		fmt.Println("OK: no issues found")
	}
	return nil
}

func printDiags(diags []lib.Diagnostic) {
	for _, d := range diags {
		fmt.Fprintln(os.Stderr, d.String())
	}
}
