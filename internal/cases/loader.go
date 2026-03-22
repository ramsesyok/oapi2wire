package cases

import (
	"fmt"
	"os"

	"github.com/ramsesyok/oapi2wire/internal/model"
	"gopkg.in/yaml.v3"
)

// Load reads a case YAML file and decodes it into CaseFile.
// Returns error only for I/O and YAML parse failures.
func Load(path string) (*model.CaseFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read case file %q: %w", path, err)
	}

	var cf model.CaseFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return nil, fmt.Errorf("cannot parse case file %q: %w", path, err)
	}

	return &cf, nil
}
