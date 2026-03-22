/*
Copyright © 2026 oapi2wire authors
*/
package cmd

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/ramsesyok/oapi2wire/internal/cases"
	"github.com/ramsesyok/oapi2wire/internal/generator"
	"github.com/ramsesyok/oapi2wire/internal/model"
	"github.com/ramsesyok/oapi2wire/internal/openapi"
	"github.com/ramsesyok/oapi2wire/internal/output"
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
	// 1. Load OpenAPI
	doc, err := openapi.Load(buildOpenAPI)
	if err != nil {
		return fmt.Errorf("loading OpenAPI: %w", err)
	}

	// 2. Build operation index
	ops, diags := openapi.BuildOperationIndex(doc)
	if model.HasErrors(diags) {
		printDiags(diags)
		return fmt.Errorf("OpenAPI has errors")
	}

	// 3. Load case YAML
	cf, err := cases.Load(buildCases)
	if err != nil {
		return fmt.Errorf("loading case YAML: %w", err)
	}

	// 4. Validate
	valDiags := cases.Validate(cf, ops, cases.ValidateConfig{
		FailOnMissingOperation: buildFailOnMissingOp,
		FailOnMissingBodyFile:  buildFailOnMissingBody,
		ResponsesRoot:          buildResponsesRoot,
	})
	diags = append(diags, valDiags...)

	printDiags(diags)
	if model.HasErrors(diags) {
		return fmt.Errorf("validation failed with errors")
	}

	// 5. Prepare output directory
	if err := output.PrepareOutDir(output.WriterConfig{OutDir: buildOut, Clean: buildClean}); err != nil {
		return fmt.Errorf("preparing output dir: %w", err)
	}

	// 6. Generate mappings for each case
	for _, cs := range cf.Cases {
		op, ok := ops[cs.OperationID]
		if !ok {
			continue // already reported in validation
		}

		mapping, err := generator.BuildMapping(cs, op, cf.Defaults)
		if err != nil {
			return fmt.Errorf("building mapping for case %q: %w", cs.ID, err)
		}

		filename := generator.MappingFileName(op.OperationID, cs.ID)
		if err := output.WriteMapping(buildOut, filename, mapping); err != nil {
			return fmt.Errorf("writing mapping %s: %w", filename, err)
		}
	}

	// 7. Copy body files to __files/
	filesDir := filepath.Join(buildOut, "__files")
	if err := generator.CopyBodyFiles(cf.Cases, buildResponsesRoot, filesDir); err != nil {
		return fmt.Errorf("copying body files: %w", err)
	}

	// 8. Auto-fallback generation
	if !buildNoAutoFallback {
		// Sort opIDs for deterministic output
		opIDs := make([]string, 0, len(ops))
		for id := range ops {
			opIDs = append(opIDs, id)
		}
		sort.Strings(opIDs)

		for _, opID := range opIDs {
			if !generator.NeedsAutoFallback(opID, cf.Cases) {
				continue
			}
			fb := generator.BuildFallback(ops[opID])

			if err := output.WriteMapping(buildOut, fb.MappingFileName, fb.Mapping); err != nil {
				return fmt.Errorf("writing fallback mapping: %w", err)
			}
			if err := output.WriteBodyFile(buildOut, fb.BodyFileName, fb.BodyContent); err != nil {
				return fmt.Errorf("writing fallback body: %w", err)
			}
		}
	}

	fmt.Printf("build complete → %s\n", buildOut)
	return nil
}
