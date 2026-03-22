/*
Copyright © 2026 oapi2wire authors
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "oapi2wire",
	Short: "Generate WireMock stubs from OpenAPI definitions",
	Long: `oapi2wire generates WireMock mappings/ and __files/
from OpenAPI definitions and case YAML files.

Commands:
  init     Generate case YAML template and response stubs from OpenAPI
  build    Generate WireMock artifacts from OpenAPI + case YAML
  validate Validate OpenAPI and case YAML consistency`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
