package generator

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ramsesyok/oapi2wire/internal/model"
)

// CopyBodyFiles copies all bodyFiles referenced in cases from responsesRoot to outFilesDir.
func CopyBodyFiles(cases []model.CaseSpec, responsesRoot, outFilesDir string) error {
	for _, c := range cases {
		if c.Response.BodyFile == "" {
			continue
		}

		src := filepath.Join(responsesRoot, c.Response.BodyFile)
		dst := filepath.Join(outFilesDir, c.Response.BodyFile)

		if _, err := os.Stat(src); os.IsNotExist(err) {
			// Skip missing files - validator already reported this
			continue
		}

		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("copying %s → %s: %w", src, dst, err)
		}
	}
	return nil
}

// copyFile copies src to dst, creating parent directories as needed.
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Close()
}
