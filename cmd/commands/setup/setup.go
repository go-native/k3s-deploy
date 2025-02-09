package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-native/k3s-deploy/cmd/helm"
	"github.com/go-native/k3s-deploy/cmd/types"
	"github.com/melbahja/goph"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
)

func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Setup k3s cluster and required components",
		Long: `Setup k3s cluster on the server specified in deploy.yml.
This command will:
1. Connect to the server using SSH
2. Install k3s
3. Install cert-manager
4. Configure local kubeconfig
5. Generate Helm charts`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return setupCluster()
		},
	}
}

func setupCluster() error {
	// Read deploy.yml
	data, err := os.ReadFile("deploy.yml")
	if err != nil {
		return fmt.Errorf("failed to read deploy.yml: %v", err)
	}

	var config types.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse deploy.yml: %v", err)
	}

	// Setup server
	if err := setupServer(&config); err != nil {
		return err
	}

	// Generate Helm charts after server setup
	fmt.Println("Generating Helm charts...")
	if err := helm.GenerateCharts(&config); err != nil {
		return fmt.Errorf("failed to generate Helm charts: %v", err)
	}
	fmt.Println("Successfully generated Helm charts")

	return nil
}

func setupServer(config *types.Config) error {
	// Generate Helm charts before SSH connection

	// Create SSH client
	var auth goph.Auth
	var err error
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

	// Check if k3s is already installed
	checkK3sCmd := "which k3s || true"
	output, err := client.Run(checkK3sCmd)
	if err != nil {
		return fmt.Errorf("failed to check k3s installation: %v", err)
	}

	if strings.TrimSpace(string(output)) == "" {
		// Install k3s if not found
		fmt.Println("Installing k3s...")
		_, err = client.Run("curl -sfL https://get.k3s.io | sh -")
		if err != nil {
			return fmt.Errorf("failed to install k3s: %v", err)
		}

		// Wait for k3s to be ready
		fmt.Println("Waiting for k3s to be ready...")
		time.Sleep(10 * time.Second)
	} else {
		fmt.Println("k3s is already installed, skipping installation...")
	}

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
	// Checking if cert-manager is already installed
	checkCertManagerCmd := "kubectl get deployment cert-manager --output name 2>/dev/null || true"
	output, err = client.Run(checkCertManagerCmd)
	if err != nil {
		return fmt.Errorf("failed to check for existing cert-manager: %v", err)
	}

	if strings.TrimSpace(string(output)) == "" {
		// Install cert-manager if not found
		fmt.Println("Installing cert-manager...")
		_, err = client.Run("kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.yaml")
		if err != nil {
			return fmt.Errorf("failed to install cert-manager: %v", err)
		}

		fmt.Println("Waiting for cert-manager to be ready...")
		time.Sleep(30 * time.Second)
	} else {
		fmt.Println("cert-manager is already installed, skipping installation...")
	}

	fmt.Println("Creating ClusterIssuer for Let's Encrypt...")

	fmt.Println("Checking for existing ClusterIssuer...")
	checkIssuerCmd := "kubectl get clusterissuer lets-encrypt-issuer --output name 2>/dev/null || true"
	output, err = client.Run(checkIssuerCmd)
	if err != nil {
		return fmt.Errorf("failed to check for existing cluster issuer: %v", err)
	}

	if strings.TrimSpace(string(output)) != "" {
		fmt.Println("ClusterIssuer already exists, skipping creation...")
		return nil
	}

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
            class: traefik' | kubectl apply -f -`, config.Traffic.Email)

	_, err = client.Run(clusterIssuerCmd)
	if err != nil {
		return fmt.Errorf("failed to create cluster issuer: %v", err)
	}

	fmt.Println("Setup completed successfully!")
	return nil
}
