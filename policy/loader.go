package policy

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadFromFile loads policies from a YAML file
func LoadFromFile(path string) ([]*Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read policy file: %w", err)
	}

	var policies []*Policy
	if err := yaml.Unmarshal(data, &policies); err != nil {
		// Try single policy
		var single Policy
		if err2 := yaml.Unmarshal(data, &single); err2 != nil {
			return nil, fmt.Errorf("cannot parse policy file: %w", err)
		}
		policies = []*Policy{&single}
	}

	return policies, nil
}

// LoadFromDir loads all YAML policy files from a directory
func LoadFromDir(dir string) ([]*Policy, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot read policy dir: %w", err)
	}

	all := make([]*Policy, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		policies, err := LoadFromFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		all = append(all, policies...)
	}

	return all, nil
}

// SaveToFile saves policies to a YAML file
func SaveToFile(path string, policies []*Policy) error {
	data, err := yaml.Marshal(policies)
	if err != nil {
		return fmt.Errorf("cannot marshal policies: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("cannot create directory: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}
