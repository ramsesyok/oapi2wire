/*
Copyright © 2026 oapi2wire authors
*/
package cmd

import (
	"fmt"

	lib "github.com/ramsesyok/oapi2wire/pkg/oapi2wire"
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
	result, err := lib.Init(lib.InitOptions{
		OpenAPIPath:   initOpenAPI,
		OutCasesPath:  initOutCases,
		ResponsesRoot: initResponsesRoot,
		Force:         initForce,
		Strict:        initStrict,
	})
	printDiags(result.Diagnostics)
	if err != nil {
		return err
	}

	fmt.Printf("wrote case YAML → %s\n", result.OutCasesPath)
	fmt.Printf("generated %d cases, %d response stubs\n", result.GeneratedCases, result.ResponseFilesWritten)
	return nil
}
