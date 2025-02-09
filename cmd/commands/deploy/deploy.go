package deploy

import (
	"fmt"
	"os"

	"github.com/go-native/k3s-deploy/cmd/docker"
	"github.com/go-native/k3s-deploy/cmd/helm"
	"github.com/go-native/k3s-deploy/cmd/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "deploy",
		Short: "Deploy application to K3s cluster",
		Long:  `Deploy application to K3s cluster using Helm charts`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return deployApplication()
		},
	}
}

func deployApplication() error {
	// Read deploy.yml
	data, err := os.ReadFile("deploy.yml")
	if err != nil {
		return fmt.Errorf("failed to read deploy.yml: %v", err)
	}

	var config types.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse deploy.yml: %v", err)
	}

	// Build and push Docker image
	if err := docker.BuildAndPushImage(&config); err != nil {
		return fmt.Errorf("failed to build and push Docker image: %v", err)
	}

	// Deploy with Helm
	if err := helm.Deploy(&config); err != nil {
		return fmt.Errorf("failed to deploy with Helm: %v", err)
	}

	return nil
}
