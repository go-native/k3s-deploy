/*
Copyright © 2025 Taron Mehrabyan <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := deployApplication(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// deployCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// deployCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func deployApplication() error {
	// Read deploy.yml
	data, err := os.ReadFile("deploy.yml")
	if err != nil {
		return fmt.Errorf("failed to read deploy.yml: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse deploy.yml: %v", err)
	}

	// Generate Helm charts
	if err := generateHelmCharts(&config); err != nil {
		return fmt.Errorf("failed to generate Helm charts: %v", err)
	}

	fmt.Println("Successfully generated Helm charts")
	// Build and push Docker image
	if err := buildAndPushImage(&config); err != nil {
		return fmt.Errorf("failed to build and push Docker image: %v", err)
	}

	// Deploy with Helm
	if err := deployWithHelm(&config); err != nil {
		return fmt.Errorf("failed to deploy with Helm: %v", err)
	}

	return nil
}

func generateHelmCharts(config *Config) error {
	helmDir := "helm"
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
		"deployment.yaml": generateDeploymentYAML,
		"service.yaml":    generateServiceYAML,
		"secrets.yaml":    generateSecretsYAML,
		"ingress.yaml":    generateIngressYAML,
	}

	// Process each template
	for filename, generator := range templates {
		if err := mergeTemplate(templatesDir, filename, generator, config); err != nil {
			return fmt.Errorf("failed to merge %s: %v", filename, err)
		}
	}

	return nil
}

type templateGenerator func(*Config) string

func mergeValuesYAML(config *Config) error {
	// Generate new values.yaml content
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

	return mergeYAMLFile(filepath.Join("helm", "values.yaml"), []byte(newContent))
}

func generateEnvValues(config *Config) string {
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

func mergeTemplate(templatesDir, filename string, generator templateGenerator, config *Config) error {
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

func mergeChartYAML(helmDir string, config *Config) error {
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

func generateIngressRule(domain string) string {
	return fmt.Sprintf(`    - host: "%s"
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: {{ .Release.Name }}
                port:
                  number: 80
`, domain)
}

func generateIngressYAML(config *Config) string {
	var content strings.Builder

	if config.Traffic.RedirectWWW {
		content.WriteString(fmt.Sprintf(`apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: redirect-www
  namespace: {{ .Release.Namespace }}
spec:
  redirectRegex:
    regex: ^https://www\.%s/(.*)
    replacement: https://%s/${1}
    permanent: true
---
`, config.Traffic.Domain, config.Traffic.Domain))
	}

	content.WriteString(`apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ .Release.Name }}-ingress
  namespace: {{ .Release.Namespace }}
  annotations:
`)

	if config.Traffic.TSL {
		content.WriteString(`    traefik.ingress.kubernetes.io/router.entrypoints: websecure
    cert-manager.io/cluster-issuer: "lets-encrypt-issuer"
    traefik.ingress.kubernetes.io/router.tls: "true"
`)
	}

	if config.Traffic.RedirectWWW {
		content.WriteString(`    traefik.ingress.kubernetes.io/router.middlewares: {{ .Release.Namespace }}-redirect-www@kubernetescrd
`)
	}

	content.WriteString(fmt.Sprintf(`spec:
  tls:
    - hosts:
        - "%s"
`, config.Traffic.Domain))

	if config.Traffic.RedirectWWW {
		content.WriteString(fmt.Sprintf(`        - "www.%s"
`, config.Traffic.Domain))
	}

	content.WriteString(`      secretName: {{ .Release.Name }}-ingress-tls
  rules:
`)

	// Add main domain rule
	content.WriteString(generateIngressRule(config.Traffic.Domain))

	// Add www domain rule if redirect is enabled
	if config.Traffic.RedirectWWW {
		content.WriteString(generateIngressRule("www." + config.Traffic.Domain))
	}

	return content.String()
}

func generateDeploymentYAML(config *Config) string {
	var content strings.Builder

	content.WriteString(fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Release.Name }}
  namespace: {{ .Release.Namespace }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      app: {{ .Release.Name }}
  template:
    metadata:
      labels:
        app: {{ .Release.Name }}
    spec:
      containers:
        - name: {{ .Release.Name }}
          image: %s
          ports:
            - containerPort: %d
`, config.Image.Name, config.Traffic.Port))

	// Add environment variables from clear section
	switch v := config.Env.Clear.(type) {
	case map[interface{}]interface{}:
		content.WriteString("          env:\n")
		for key, value := range v {
			content.WriteString(fmt.Sprintf("            - name: %v\n              value: \"%v\"\n", key, value))
		}
	case []interface{}:
		content.WriteString("          env:\n")
		for _, key := range v {
			content.WriteString(fmt.Sprintf("            - name: %v\n              valueFrom:\n              configMapKeyRef:\n                name: {{ .Release.Name }}-config\n                key: %v\n", key, key))
		}
	}

	// Add secret references if there are any secrets
	if len(config.Env.Secrets) > 0 {
		content.WriteString(`          envFrom:
            - secretRef:
                name: {{ .Release.Name }}-secrets
`)
	}

	// Add resources section
	content.WriteString(`          resources:
            {{- toYaml .Values.resources | nindent 12 }}
      imagePullSecrets:
        - name: registry-secret
`)

	return content.String()
}

func generateServiceYAML(config *Config) string {
	return fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: {{ .Release.Name }}
  namespace: {{ .Release.Namespace }}
spec:
  type: ClusterIP
  ports:
    - port: 80
      targetPort: %d
  selector:
    app: {{ .Release.Name }}
`, config.Traffic.Port)
}

func generateSecretsYAML(config *Config) string {
	var content strings.Builder

	// Registry secret
	content.WriteString(fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: registry-secret
  namespace: {{ .Release.Namespace }}
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: %s
---
`, generateDockerConfig(config)))

	// Application secrets
	content.WriteString(`apiVersion: v1
kind: Secret
metadata:
  name: {{ .Release.Name }}-secrets
  namespace: {{ .Release.Namespace }}
type: Opaque
data:
`)

	// Add secret references
	for _, secretName := range config.Env.Secrets {
		content.WriteString(fmt.Sprintf("  %s: {{ .Values.env.%s | b64enc }}\n", secretName, secretName))
	}

	return content.String()
}

func generateValuesYAML(config *Config) error {
	var valuesContent strings.Builder
	valuesContent.WriteString("replicaCount: 1\n\n")

	// Add environment variables section
	valuesContent.WriteString("env:\n")
	// Add secrets as environment variable references
	for _, secretName := range config.Env.Secrets {
		valuesContent.WriteString(fmt.Sprintf("  %s: \"${%s}\"\n", secretName, secretName))
	}

	// Add resources section
	valuesContent.WriteString(`
resources:
  limits:
    cpu: "500m"
    memory: "512Mi"
  requests:
    cpu: "250m"
    memory: "256Mi"
`)

	return os.WriteFile("helm/values.yaml", []byte(valuesContent.String()), 0644)
}

func generateDockerConfig(config *Config) string {
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

func buildAndPushImage(config *Config) error {
	fmt.Println("Building Docker image...")

	// Build Docker image
	buildCmd := exec.Command("docker", "build", "-t", config.Image.Name, ".")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("failed to build Docker image: %v", err)
	}

	// Tag image with registry
	fullImageName := fmt.Sprintf("%s/%s", config.Image.Registry.Server, config.Image.Name)
	tagCmd := exec.Command("docker", "tag", config.Image.Name, fullImageName)
	if err := tagCmd.Run(); err != nil {
		return fmt.Errorf("failed to tag Docker image: %v", err)
	}

	// Get registry password from environment
	registryPassword := os.Getenv(config.Image.Registry.Password[0])
	if registryPassword == "" {
		return fmt.Errorf("environment variable for registry password is not set")
	}

	// Login to registry
	loginCmd := exec.Command("docker", "login",
		config.Image.Registry.Server,
		"-u", config.Image.Registry.Username,
		"-p", registryPassword)
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

func deployWithHelm(config *Config) error {
	fmt.Println("Deploying with Helm...")

	// Prepare helm upgrade command
	args := []string{
		"upgrade",
		"--install",
		config.Service,
		"./helm",
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
			if value == "" {
				return fmt.Errorf("environment variable %s is not set", key)
			}
			args = append(args, "--set", fmt.Sprintf("env.%s=%s", key, value))
		}
	}

	// Handle secret environment variables
	for _, secretName := range config.Env.Secrets {
		value := os.Getenv(secretName)
		if value == "" {
			return fmt.Errorf("environment variable %s is not set", secretName)
		}
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
