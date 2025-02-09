package helm

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/go-native/k3s-deploy/cmd/types"
)

func Deploy(config *types.Config) error {
	fmt.Println("Deploying with Helm...")

	// Prepare helm upgrade command
	args := []string{
		"upgrade",
		"--install",
		config.Service,
		".helm",
		"-n", config.Service,
		"--create-namespace",
		"--history-max", "1",
	}

	switch v := config.Env.Clear.(type) {
	case map[interface{}]interface{}:
		// Direct values from yaml
		for key, value := range v {
			args = append(args, "--set", fmt.Sprintf("env.%s=%v", key, value))
		}
	case []interface{}:
		// Keys to get from environment
		for _, key := range v {
			value := os.Getenv(key.(string))
			args = append(args, "--set", fmt.Sprintf("env.%s=%s", key, value))
		}
	}

	// Handle secret environment variables
	for _, secretName := range config.Env.Secrets {
		value := os.Getenv(secretName)
		args = append(args, "--set", fmt.Sprintf("env.%s=%s", secretName, value))
	}

	// Execute helm upgrade command
	cmd := exec.Command("helm", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to deploy with Helm: %v", err)
	}

	fmt.Println("Successfully deployed application")
	return nil
}
