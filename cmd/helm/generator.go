package helm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-native/k3s-deploy/cmd/types"
)

type templateGenerator func(*types.Config) string

// GenerateCharts handles all Helm chart generation
func GenerateCharts(config *types.Config) error {
	helmDir := ".helm"
	templatesDir := filepath.Join(helmDir, "templates")

	// Create directories if they don't exist
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return fmt.Errorf("failed to create helm directories: %v", err)
	}

	// Generate Chart.yaml with merge support
	if err := mergeChartYAML(helmDir, config); err != nil {
		return fmt.Errorf("failed to merge Chart.yaml: %v", err)
	}

	// Generate values.yaml with merge support
	if err := mergeValuesYAML(config); err != nil {
		return fmt.Errorf("failed to merge values.yaml: %v", err)
	}

	// Define templates to generate/merge
	templates := map[string]templateGenerator{
		"deployment.yaml": GenerateDeploymentYAML,
		"service.yaml":    GenerateServiceYAML,
		"secrets.yaml":    GenerateSecretsYAML,
		"ingress.yaml":    GenerateIngressYAML,
	}

	// Process each template
	for filename, generator := range templates {
		if err := mergeTemplate(templatesDir, filename, generator, config); err != nil {
			return fmt.Errorf("failed to merge %s: %v", filename, err)
		}
	}

	return nil
}
