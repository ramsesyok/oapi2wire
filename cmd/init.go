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
	initOpenAPI       string
	initOutCases      string
	initResponsesRoot string
	initForce         bool
	initStrict        bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate case YAML template and response stubs from OpenAPI",
	Long: `Generate a case YAML template and response JSON stubs from an OpenAPI definition.
Edit the generated files to define your mock scenarios, then run build.`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVar(&initOpenAPI, "openapi", "", "Path to OpenAPI file (required)")
	initCmd.Flags().StringVar(&initOutCases, "out-cases", "mock-cases.yaml", "Output case YAML path")
	initCmd.Flags().StringVar(&initResponsesRoot, "responses-root", "mock-responses", "Responses root directory")
	initCmd.Flags().BoolVar(&initForce, "force", false, "Overwrite existing files")
	initCmd.Flags().BoolVar(&initStrict, "strict", false, "Treat OpenAPI inconsistencies as errors")
	if err := initCmd.MarkFlagRequired("openapi"); err != nil {
		panic(err)
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	// 1. Load OpenAPI
	doc, err := openapi.Load(initOpenAPI)
	if err != nil {
		return fmt.Errorf("loading OpenAPI: %w", err)
	}

	// 2. Build operation index
	ops, diags := openapi.BuildOperationIndex(doc)
	if model.HasErrors(diags) {
		printDiags(diags)
		return fmt.Errorf("OpenAPI has errors")
	}
	if initStrict && len(diags) > 0 {
		printDiags(diags)
		return fmt.Errorf("OpenAPI has warnings (--strict)")
	}
	printDiags(diags)

	// 3. Generate template
	result, err := cases.GenerateTemplate(doc, ops)
	if err != nil {
		return fmt.Errorf("generating template: %w", err)
	}

	// 4. Check case YAML existence before writing
	if !initForce {
		if _, err := os.Stat(initOutCases); err == nil {
			return fmt.Errorf("file already exists: %s (use --force to overwrite)", initOutCases)
		}
	}

	// 5. Write case YAML and response stubs
	written, err := result.Write(initOutCases, initResponsesRoot, initForce)
	if err != nil {
		return fmt.Errorf("writing init output: %w", err)
	}

	fmt.Printf("wrote case YAML → %s\n", initOutCases)
	fmt.Printf("generated %d cases, %d response stubs\n", len(ops), written)
	return nil
}
