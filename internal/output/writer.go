package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ramsesyok/oapi2wire/internal/model"
)

// WriterConfig controls output behaviour.
type WriterConfig struct {
	OutDir string
	Clean  bool
}

// PrepareOutDir optionally cleans and creates outDir/mappings/ and outDir/__files/.
func PrepareOutDir(cfg WriterConfig) error {
	if cfg.Clean {
		if err := os.RemoveAll(cfg.OutDir); err != nil {
			return fmt.Errorf("cleaning output dir %s: %w", cfg.OutDir, err)
		}
	}

	for _, subdir := range []string{"mappings", "__files"} {
		dir := filepath.Join(cfg.OutDir, subdir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}
	return nil
}

// WriteMapping serialises m and writes it to outDir/mappings/<filename>.
func WriteMapping(outDir, filename string, m model.WireMockMapping) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("serialising mapping %s: %w", filename, err)
	}
	data = append(data, '\n')

	path := filepath.Join(outDir, "mappings", filename)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing mapping %s: %w", path, err)
	}
	return nil
}

// WriteBodyFile writes content to outDir/__files/<relPath>.
func WriteBodyFile(outDir, relPath string, content []byte) error {
	path := filepath.Join(outDir, "__files", relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}
