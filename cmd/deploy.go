/*
Copyright © 2025 Taron Mehrabyan <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
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
	// TODO: Add Docker build and push logic
	// TODO: Add Helm deployment logic

	return nil
}

func generateHelmCharts(config *Config) error {
	// Create helm directory structure
	helmDir := "helm"
	templatesDir := filepath.Join(helmDir, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return fmt.Errorf("failed to create helm directories: %v", err)
	}

	// Generate Chart.yaml
	chartContent := fmt.Sprintf(`apiVersion: v2
name: %s
type: application
version: 0.1.0
appVersion: "1.16.0"
`, config.Name)

	if err := os.WriteFile(filepath.Join(helmDir, "Chart.yaml"), []byte(chartContent), 0644); err != nil {
		return fmt.Errorf("failed to create Chart.yaml: %v", err)
	}

	// Generate values.yaml
	if err := generateValuesYAML(config); err != nil {
		return fmt.Errorf("failed to create values.yaml: %v", err)
	}

	// Generate templates
	templates := map[string]string{
		"deployment.yaml": generateDeploymentYAML(config),
		"service.yaml":    generateServiceYAML(config),
		"secrets.yaml":    generateSecretsYAML(config),
		"ingress.yaml":    generateIngressYAML(config),
	}

	for filename, content := range templates {
		if err := os.WriteFile(filepath.Join(templatesDir, filename), []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create %s: %v", filename, err)
		}
	}

	return nil
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

	if config.Service.RedirectWWW {
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
`, config.Service.Domain, config.Service.Domain))
	}

	content.WriteString(`apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ .Release.Name }}-ingress
  namespace: {{ .Release.Namespace }}
  annotations:
`)

	if config.Service.TSL {
		content.WriteString(`    traefik.ingress.kubernetes.io/router.entrypoints: websecure
    cert-manager.io/cluster-issuer: "lets-encrypt-issuer"
    traefik.ingress.kubernetes.io/router.tls: "true"
`)
	}

	if config.Service.RedirectWWW {
		content.WriteString(`    traefik.ingress.kubernetes.io/router.middlewares: {{ .Release.Namespace }}-redirect-www@kubernetescrd
`)
	}

	content.WriteString(fmt.Sprintf(`spec:
  tls:
    - hosts:
        - "%s"
`, config.Service.Domain))

	if config.Service.RedirectWWW {
		content.WriteString(fmt.Sprintf(`        - "www.%s"
`, config.Service.Domain))
	}

	content.WriteString(`      secretName: {{ .Release.Name }}-ingress-tls
  rules:
`)

	// Add main domain rule
	content.WriteString(generateIngressRule(config.Service.Domain))

	// Add www domain rule if redirect is enabled
	if config.Service.RedirectWWW {
		content.WriteString(generateIngressRule("www." + config.Service.Domain))
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
`, config.Image.Name, config.Service.Port))

	// Add environment variables from clear section
	if len(config.Env.Clear) > 0 {
		content.WriteString("          env:\n")
		for key, value := range config.Env.Clear {
			content.WriteString(fmt.Sprintf("            - name: %s\n              value: \"%s\"\n", key, value))
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
`, config.Service.Port)
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
					config.Image.Registry.Password))),
			},
		},
	}

	dockerConfigJSON, _ := json.Marshal(dockerConfig)
	return base64.StdEncoding.EncodeToString(dockerConfigJSON)
}
