package docker

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/go-native/k3s-deploy/cmd/types"
)

func GenerateConfig(config *types.Config) string {
	// Get registry password from environment
	if len(config.Image.Registry.Password) != 1 {
		return "" // or handle error appropriately
	}

	registryPassword := os.Getenv(config.Image.Registry.Password[0])
	if registryPassword == "" {
		return "" // or handle error appropriately
	}

	dockerConfig := struct {
		Auths map[string]struct {
			Auth string `json:"auth"`
		} `json:"auths"`
	}{
		Auths: map[string]struct {
			Auth string `json:"auth"`
		}{
			config.Image.Registry.Server: {
				Auth: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s",
					config.Image.Registry.Username,
					registryPassword))),
			},
		},
	}

	dockerConfigJSON, _ := json.Marshal(dockerConfig)
	return base64.StdEncoding.EncodeToString(dockerConfigJSON)
}

func BuildAndPushImage(config *types.Config) error {
	fmt.Println("Building Docker image...")

	// Build Docker image with full registry path
	fullImageName := fmt.Sprintf("%s/%s", config.Image.Registry.Server, config.Image.Name)
	buildCmd := exec.Command("docker", "build", "--platform", "linux/amd64", "-t", fullImageName, ".")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("failed to build Docker image: %v", err)
	}

	// Get registry password from environment
	registryPassword := os.Getenv(config.Image.Registry.Password[0])
	if registryPassword == "" {
		return fmt.Errorf("environment variable for registry password is not set")
	}

	// Login to registry using a more secure method
	loginCmd := exec.Command("docker", "login",
		config.Image.Registry.Server,
		"-u", config.Image.Registry.Username,
		"-p", registryPassword)

	// Provide password through stdin
	loginCmd.Stdin = strings.NewReader(registryPassword)
	loginCmd.Stdout = os.Stdout
	loginCmd.Stderr = os.Stderr

	if err := loginCmd.Run(); err != nil {
		return fmt.Errorf("failed to login to registry: %v", err)
	}

	// Push image
	fmt.Println("Pushing Docker image...")
	pushCmd := exec.Command("docker", "push", fullImageName)
	pushCmd.Stdout = os.Stdout
	pushCmd.Stderr = os.Stderr
	if err := pushCmd.Run(); err != nil {
		return fmt.Errorf("failed to push Docker image: %v", err)
	}

	return nil
}

func GenerateDockerConfig(config *types.Config) string {
	// Get registry password from environment
	if len(config.Image.Registry.Password) != 1 {
		return "" // or handle error appropriately
	}

	registryPassword := os.Getenv(config.Image.Registry.Password[0])
	if registryPassword == "" {
		return "" // or handle error appropriately
	}

	dockerConfig := struct {
		Auths map[string]struct {
			Auth string `json:"auth"`
		} `json:"auths"`
	}{
		Auths: map[string]struct {
			Auth string `json:"auth"`
		}{
			config.Image.Registry.Server: {
				Auth: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s",
					config.Image.Registry.Username,
					registryPassword))),
			},
		},
	}

	dockerConfigJSON, _ := json.Marshal(dockerConfig)
	return base64.StdEncoding.EncodeToString(dockerConfigJSON)
}
