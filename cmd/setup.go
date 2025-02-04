/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/melbahja/goph"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup k3s cluster and required components",
	Long: `Setup k3s cluster on the server specified in deploy.yml.
This command will:
1. Connect to the server using SSH
2. Install k3s
3. Install cert-manager
4. Configure local kubeconfig`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := setupCluster(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func setupCluster() error {
	// Read deploy.yml
	data, err := os.ReadFile("deploy.yml")
	if err != nil {
		return fmt.Errorf("failed to read deploy.yml: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse deploy.yml: %v", err)
	}

	// Generate Helm charts before SSH connection

	// Create SSH client
	var auth goph.Auth
	if config.Server.SSHKey != "" {
		expandedPath := os.ExpandEnv(config.Server.SSHKey)
		// Handle tilde expansion
		if strings.HasPrefix(expandedPath, "~") {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %v", err)
			}
			expandedPath = filepath.Join(home, expandedPath[1:])
		}
		auth, err = goph.Key(expandedPath, "")
		if err != nil {
			return fmt.Errorf("failed to setup SSH key auth: %v", err)
		}
	} else if config.Server.Password != "" {
		auth = goph.Password(config.Server.Password)
	} else {
		return fmt.Errorf("neither ssh_key nor password provided in deploy.yml")
	}

	// Create SSH client config
	sshConfig := &goph.Config{
		Auth:     auth,
		User:     config.Server.User,
		Addr:     config.Server.IP,
		Port:     22,
		Timeout:  30 * time.Second,
		Callback: ssh.InsecureIgnoreHostKey(), // TODO: Change this to a proper callback
	}

	// Connect to server
	client, err := goph.NewConn(sshConfig)
	if err != nil {
		return fmt.Errorf("failed to create SSH client: %v", err)
	}
	defer client.Close()

	// Install k3s
	fmt.Println("Installing k3s...")
	_, err = client.Run("curl -sfL https://get.k3s.io | sh -")
	if err != nil {
		return fmt.Errorf("failed to install k3s: %v", err)
	}

	// Wait for k3s to be ready
	fmt.Println("Waiting for k3s to be ready...")
	time.Sleep(10 * time.Second)

	// Get kubeconfig
	fmt.Println("Fetching kubeconfig...")
	kubeconfig, err := client.Run("cat /etc/rancher/k3s/k3s.yaml")
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %v", err)
	}

	// Replace localhost with server IP
	kubeconfigContent := strings.ReplaceAll(string(kubeconfig), "127.0.0.1", config.Server.IP)

	// Save kubeconfig
	fmt.Println("Saving kubeconfig...")
	if err := saveKubeconfig(kubeconfigContent); err != nil {
		return fmt.Errorf("failed to save kubeconfig: %v", err)
	}

	// Install cert-manager
	fmt.Println("Installing cert-manager...")
	_, err = client.Run("kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.yaml")
	if err != nil {
		return fmt.Errorf("failed to install cert-manager: %v", err)
	}

	fmt.Println("Waiting for cert-manager to be ready...")
	time.Sleep(30 * time.Second)

	fmt.Println("Creating ClusterIssuer for Let's Encrypt...")

	clusterIssuerCmd := fmt.Sprintf(`echo 'apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: lets-encrypt-issuer
spec:
  acme:
    email: %s
    server: https://acme-v02.api.letsencrypt.org/directory
    privateKeySecretRef:
      name: letsencrypt-account-key
    solvers:
      - http01:
          ingress:
            class: traefik' | kubectl apply -f -`, config.Service.Email)

	_, err = client.Run(clusterIssuerCmd)
	if err != nil {
		return fmt.Errorf("failed to create cluster issuer: %v", err)
	}

	fmt.Println("Setup completed successfully!")

	return nil
}

func saveKubeconfig(content string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	kubeDir := filepath.Join(home, ".kube")
	if err := os.MkdirAll(kubeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .kube directory: %v", err)
	}

	configPath := filepath.Join(kubeDir, "config")

	// Check for existing config and read it
	existingContent := ""
	if _, err := os.Stat(configPath); err == nil {
		existingBytes, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to read existing kubeconfig: %v", err)
		}
		existingContent = string(existingBytes)

		// Create backup of existing config
		backupPath := configPath + ".backup." + time.Now().Format("20060102150405")
		if err := os.WriteFile(backupPath, existingBytes, 0600); err != nil {
			return fmt.Errorf("failed to create backup of existing kubeconfig: %v", err)
		}
		fmt.Printf("Backed up existing kubeconfig to %s\n", backupPath)
	}

	// Modify the new config with unique names
	modifiedContent, err := modifyKubeconfigNames(content, existingContent)
	if err != nil {
		return fmt.Errorf("failed to modify kubeconfig names: %v", err)
	}

	if existingContent != "" {
		// Parse existing config
		var existingConfig map[string]interface{}
		if err := yaml.Unmarshal([]byte(existingContent), &existingConfig); err != nil {
			return fmt.Errorf("failed to parse existing kubeconfig: %v", err)
		}

		// Parse modified config
		var newConfig map[string]interface{}
		if err := yaml.Unmarshal([]byte(modifiedContent), &newConfig); err != nil {
			return fmt.Errorf("failed to parse modified kubeconfig: %v", err)
		}

		// Merge clusters
		existingClusters := existingConfig["clusters"].([]interface{})
		newClusters := newConfig["clusters"].([]interface{})
		existingConfig["clusters"] = append(existingClusters, newClusters...)

		// Merge contexts
		existingContexts := existingConfig["contexts"].([]interface{})
		newContexts := newConfig["contexts"].([]interface{})
		existingConfig["contexts"] = append(existingContexts, newContexts...)

		// Merge users
		existingUsers := existingConfig["users"].([]interface{})
		newUsers := newConfig["users"].([]interface{})
		existingConfig["users"] = append(existingUsers, newUsers...)

		// Set current-context to the new one
		existingConfig["current-context"] = newConfig["current-context"]

		// Marshal merged config
		mergedContent, err := yaml.Marshal(existingConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal merged kubeconfig: %v", err)
		}

		// Write merged config
		if err := os.WriteFile(configPath, mergedContent, 0600); err != nil {
			return fmt.Errorf("failed to write merged kubeconfig: %v", err)
		}
	} else {
		// No existing config - write modified content directly
		if err := os.WriteFile(configPath, []byte(modifiedContent), 0600); err != nil {
			return fmt.Errorf("failed to write kubeconfig: %v", err)
		}
	}

	return nil
}

func mergeKubeconfigs(existing, new yaml.MapSlice) yaml.MapSlice {
	// Initialize merged config with existing content
	merged := existing

	// Helper function to find an item in MapSlice by key
	findItem := func(slice yaml.MapSlice, key string) (int, bool) {
		for i, item := range slice {
			if item.Key.(string) == key {
				return i, true
			}
		}
		return -1, false
	}

	// Merge clusters
	if idx, found := findItem(merged, "clusters"); found {
		existingClusters := merged[idx].Value.([]interface{})
		newClusters := new[idx].Value.([]interface{})
		merged[idx].Value = append(existingClusters, newClusters...)
	}

	// Merge contexts
	if idx, found := findItem(merged, "contexts"); found {
		existingContexts := merged[idx].Value.([]interface{})
		newContexts := new[idx].Value.([]interface{})
		merged[idx].Value = append(existingContexts, newContexts...)
	}

	// Merge users
	if idx, found := findItem(merged, "users"); found {
		existingUsers := merged[idx].Value.([]interface{})
		newUsers := new[idx].Value.([]interface{})
		merged[idx].Value = append(existingUsers, newUsers...)
	}

	return merged
}

func modifyKubeconfigNames(content string, existingConfig string) (string, error) {
	var config map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &config); err != nil {
		return "", fmt.Errorf("failed to parse new kubeconfig: %v", err)
	}

	// Parse existing config if it exists
	var existing map[string]interface{}
	if existingConfig != "" {
		if err := yaml.Unmarshal([]byte(existingConfig), &existing); err != nil {
			return "", fmt.Errorf("failed to parse existing kubeconfig: %v", err)
		}
	}

	// Helper function to find unique name
	findUniqueName := func(baseName string, existingNames map[string]bool) string {
		if !existingNames[baseName] {
			return baseName
		}
		counter := 1
		for {
			newName := fmt.Sprintf("%s_%d", baseName, counter)
			if !existingNames[newName] {
				return newName
			}
			counter++
		}
	}

	// Get existing names
	existingNames := make(map[string]bool)
	if existing != nil {
		// Collect existing cluster names
		if clusters, ok := existing["clusters"].([]interface{}); ok {
			for _, c := range clusters {
				if cluster, ok := c.(map[interface{}]interface{}); ok {
					if name, ok := cluster["name"].(string); ok {
						existingNames[name] = true
					}
				}
			}
		}
	}

	// Update clusters with unique name
	var clusterName string
	if clusters, ok := config["clusters"].([]interface{}); ok && len(clusters) > 0 {
		if cluster, ok := clusters[0].(map[interface{}]interface{}); ok {
			originalName := cluster["name"].(string)
			clusterName = findUniqueName(originalName, existingNames)
			cluster["name"] = clusterName
			clusters[0] = cluster
			config["clusters"] = clusters
		}
	}

	// Reset map for context names
	existingNames = make(map[string]bool)
	if existing != nil {
		// Collect existing context names
		if contexts, ok := existing["contexts"].([]interface{}); ok {
			for _, c := range contexts {
				if context, ok := c.(map[interface{}]interface{}); ok {
					if name, ok := context["name"].(string); ok {
						existingNames[name] = true
					}
				}
			}
		}
	}

	// Update contexts with unique name
	if contexts, ok := config["contexts"].([]interface{}); ok && len(contexts) > 0 {
		if context, ok := contexts[0].(map[interface{}]interface{}); ok {
			originalName := context["name"].(string)
			newContextName := findUniqueName(originalName, existingNames)
			context["name"] = newContextName

			// Update cluster reference in context
			if contextData, ok := context["context"].(map[interface{}]interface{}); ok {
				contextData["cluster"] = clusterName
				context["context"] = contextData
			}

			contexts[0] = context
			config["contexts"] = contexts
			config["current-context"] = newContextName
		}
	}

	// Reset map for user names
	existingNames = make(map[string]bool)
	if existing != nil {
		// Collect existing user names
		if users, ok := existing["users"].([]interface{}); ok {
			for _, u := range users {
				if user, ok := u.(map[interface{}]interface{}); ok {
					if name, ok := user["name"].(string); ok {
						existingNames[name] = true
					}
				}
			}
		}
	}

	// Update users with unique name
	if users, ok := config["users"].([]interface{}); ok && len(users) > 0 {
		if user, ok := users[0].(map[interface{}]interface{}); ok {
			originalName := user["name"].(string)
			newUserName := findUniqueName(originalName, existingNames)
			user["name"] = newUserName

			// Update user reference in context
			if contexts, ok := config["contexts"].([]interface{}); ok && len(contexts) > 0 {
				if context, ok := contexts[0].(map[interface{}]interface{}); ok {
					if contextData, ok := context["context"].(map[interface{}]interface{}); ok {
						contextData["user"] = newUserName
						context["context"] = contextData
					}
				}
			}

			users[0] = user
			config["users"] = users
		}
	}

	// Marshal back to YAML
	modifiedContent, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal modified kubeconfig: %v", err)
	}

	return string(modifiedContent), nil
}
