package helm

import (
	"fmt"
	"strings"

	"github.com/go-native/k3s-deploy/cmd/docker"
	"github.com/go-native/k3s-deploy/cmd/types"
)

func GenerateIngressRule(domain string) string {
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

func GenerateIngressYAML(config *types.Config) string {
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
	content.WriteString(GenerateIngressRule(config.Traffic.Domain))

	// Add www domain rule if redirect is enabled
	if config.Traffic.RedirectWWW {
		content.WriteString(GenerateIngressRule("www." + config.Traffic.Domain))
	}

	return content.String()
}

func GenerateDeploymentYAML(config *types.Config) string {
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
          image: %s/%s
          ports:
            - containerPort: %d
`, config.Image.Registry.Server, config.Image.Name, config.Traffic.Port))

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

func GenerateServiceYAML(config *types.Config) string {
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

func GenerateSecretsYAML(config *types.Config) string {
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
`, docker.GenerateConfig(config)))

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
