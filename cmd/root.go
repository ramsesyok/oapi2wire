/*
Copyright © 2026 oapi2wire authors
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "oapi2wire",
	// 検証エラーなどで毎回使い方を表示すると、エラーと警告の一覧が読みにくくなる。
	// 引数の誤り (必須フラグの不足など) は cobra がエラーメッセージで知らせる。
	SilenceUsage: true,
	Short:        "Generate WireMock stubs from OpenAPI definitions",
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
	rootCmd.Version = currentVersion()
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
