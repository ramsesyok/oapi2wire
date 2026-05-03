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
	buildOpenAPI           string
	buildCases             string
	buildResponsesRoot     string
	buildOut               string
	buildClean             bool
	buildStrict            bool
	buildFailOnMissingOp   bool
	buildFailOnMissingBody bool
	buildNoAutoFallback    bool
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Generate WireMock mappings and __files from OpenAPI + case YAML",
	Long: `Generate WireMock artifacts (mappings/ and __files/) from OpenAPI definitions,
case YAML, and response JSON files.`,
	RunE: runBuild,
}

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().StringVar(&buildOpenAPI, "openapi", "", "Path to OpenAPI file (required)")
	buildCmd.Flags().StringVar(&buildCases, "cases", "mock-cases.yaml", "Path to case YAML file")
	buildCmd.Flags().StringVar(&buildResponsesRoot, "responses-root", "mock-responses", "Responses root directory")
	buildCmd.Flags().StringVar(&buildOut, "out", "wiremock-out", "Output directory")
	buildCmd.Flags().BoolVar(&buildClean, "clean", false, "Delete output directory before generating")
	buildCmd.Flags().BoolVar(&buildStrict, "strict", false, "Treat warnings as errors")
	buildCmd.Flags().BoolVar(&buildFailOnMissingOp, "fail-on-missing-operation", false, "Fail if case operationId not found in OpenAPI")
	buildCmd.Flags().BoolVar(&buildFailOnMissingBody, "fail-on-missing-body-file", false, "Fail if bodyFile not found in responses-root")
	buildCmd.Flags().BoolVar(&buildNoAutoFallback, "no-auto-fallback", false, "Disable automatic fallback generation")
	if err := buildCmd.MarkFlagRequired("openapi"); err != nil {
		panic(err)
	}
}

func runBuild(cmd *cobra.Command, args []string) error {
	result, err := lib.Build(lib.BuildOptions{
		OpenAPIPath:            buildOpenAPI,
		CasesPath:              buildCases,
		ResponsesRoot:          buildResponsesRoot,
		OutDir:                 buildOut,
		Clean:                  buildClean,
		Strict:                 buildStrict,
		FailOnMissingOperation: buildFailOnMissingOp,
		FailOnMissingBodyFile:  buildFailOnMissingBody,
		NoAutoFallback:         buildNoAutoFallback,
	})
	printDiags(result.Diagnostics)
	if err != nil {
		return err
	}

	fmt.Printf("build complete → %s\n", result.OutDir)
	return nil
}
