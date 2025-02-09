package helm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-native/k3s-deploy/cmd/types"
	"gopkg.in/yaml.v2"
)

func mergeValuesYAML(config *types.Config) error {
	helmDir := ".helm"
	if err := os.MkdirAll(helmDir, 0755); err != nil {
		return fmt.Errorf("failed to create helm directory: %v", err)
	}
	newContent := fmt.Sprintf(`replicaCount: 1

env:
%s
resources:
  limits:
    cpu: "500m"
    memory: "512Mi"
  requests:
    cpu: "250m"
    memory: "256Mi"
`, generateEnvValues(config))

	return mergeYAMLFile(filepath.Join(helmDir, "values.yaml"), []byte(newContent))
}

func generateEnvValues(config *types.Config) string {
	var envValues strings.Builder

	// Add clear environment variables
	switch v := config.Env.Clear.(type) {
	case map[interface{}]interface{}:
		// Direct values from yaml
		for key, value := range v {
			envValues.WriteString(fmt.Sprintf("  %s: %q\n", key, value))
		}
	case []interface{}:
		// Keys to get from environment
		for _, key := range v {
			value := os.Getenv(key.(string))
			if value == "" {
				// If env var is not set, skip it
				continue
			}
			envValues.WriteString(fmt.Sprintf("  %s: %q\n", key, value))
		}
	}

	// Add secret environment variables
	for _, key := range config.Env.Secrets {
		envValues.WriteString(fmt.Sprintf("  %s: \"${%s}\"\n", key, key))
	}

	return envValues.String()
}

func mergeTemplate(templatesDir, filename string, generator templateGenerator, config *types.Config) error {
	filepath := filepath.Join(templatesDir, filename)

	// Generate new content from deploy.yml
	newContent := generator(config)

	// Check if file exists
	existingContent, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, create new
			return os.WriteFile(filepath, []byte(newContent), 0644)
		}
		return err
	}

	// Parse both existing and new content as YAML
	var existing, new interface{}
	if err := yaml.Unmarshal(existingContent, &existing); err != nil {
		// If can't parse existing file, overwrite with new content
		return os.WriteFile(filepath, []byte(newContent), 0644)
	}
	if err := yaml.Unmarshal([]byte(newContent), &new); err != nil {
		return err
	}

	// Merge with new content taking precedence
	merged := mergeYAML(existing, new)

	// Marshal merged content
	mergedContent, err := yaml.Marshal(merged)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath, mergedContent, 0644)
}

func mergeYAML(existing, new interface{}) interface{} {
	switch newValue := new.(type) {
	case map[interface{}]interface{}:
		if existingMap, ok := existing.(map[interface{}]interface{}); ok {
			// Merge maps recursively
			result := make(map[interface{}]interface{})
			// Copy existing values
			for k, v := range existingMap {
				result[k] = v
			}
			// Override/add new values
			for k, v := range newValue {
				if existing, ok := result[k]; ok {
					// Recursive merge for nested maps
					result[k] = mergeYAML(existing, v)
				} else {
					result[k] = v
				}
			}
			return result
		}
	}
	// For non-maps or when existing is not a map, use new value
	return new
}

func mergeChartYAML(helmDir string, config *types.Config) error {
	filepath := filepath.Join(helmDir, "Chart.yaml")
	newContent := fmt.Sprintf(`apiVersion: v2
name: %s
type: application
version: 0.1.0
appVersion: "1.16.0"
`, config.Service)

	return mergeYAMLFile(filepath, []byte(newContent))
}

func mergeYAMLFile(filepath string, newContent []byte) error {
	// Check if file exists
	existingContent, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(filepath, newContent, 0644)
		}
		return err
	}

	// Parse both contents
	var existing, new interface{}
	if err := yaml.Unmarshal(existingContent, &existing); err != nil {
		return os.WriteFile(filepath, newContent, 0644)
	}
	if err := yaml.Unmarshal(newContent, &new); err != nil {
		return err
	}

	// Merge with new content taking precedence
	merged := mergeYAML(existing, new)

	// Marshal merged content
	mergedContent, err := yaml.Marshal(merged)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath, mergedContent, 0644)
}
