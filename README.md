# k3s-deploy

A CLI tool that simplifies the process of deploying applications to single-node K3s servers. This tool automates the entire deployment workflow, from server setup to application deployment.

## Features

- 🚀 Automated K3s server setup and installation
- 🔒 Secure SSH-based server configuration
- 📦 Container registry configuration
- 🔐 SSL certificate management with cert-manager
- 🌍 Domain and ingress configuration
- 🔑 Environment variable and secret management
- 📄 Simple YAML-based configuration

## Installation

### macOS

#### Intel / x86_64
```bash
wget https://github.com/go-native/k3s-deploy/releases/download/v0.1.0/k3s-deploy-macos-amd64
chmod +x k3s-deploy-macos-amd64
sudo mv k3s-deploy-macos-amd64 /usr/local/bin/k3s-deploy
```

#### Apple Silicon / ARM
```bash
wget https://github.com/go-native/k3s-deploy/releases/download/v0.1.0/k3s-deploy-macos-arm64
chmod +x k3s-deploy-macos-arm64
sudo mv k3s-deploy-macos-arm64 /usr/local/bin/k3s-deploy
```

### Linux

#### x86_64
```bash
wget https://github.com/go-native/k3s-deploy/releases/download/v0.1.0/k3s-deploy-linux-amd64
chmod +x k3s-deploy-linux-amd64
sudo mv k3s-deploy-linux-amd64 /usr/local/bin/k3s-deploy
```

## Quick Start

1. Initialize configuration:
```bash
k3s-deploy init
```

2. Edit the generated `deploy.yml` file with your server details:
```yaml
service: my-app # This becomes the name in the Chart.yaml 
image:
  name: my-user/my-app
  registry:
    server: ghcr.io
    username: my-user
    password:
	    - GITHUB_TOKEN # Injected from env variable
server: 
  ip: 192.168.1.100 # Server to setup k3s cluster
  user: root
  ssh_key: ~/.ssh/id_rsa # SSH key to connect to the server
  password: # Optional, if you want to use password instead of ssh key

traffic:
  domain: example.com # 
  tsl: true # If you want to use tsl
  redirect_www: true # If you want to redirect www to non-www
  email: my-email@example.com # Email to use for the certificate
env:
  clear:
    DB_HOST: localhost
  secrets:
    - DB_PASSWORD
```

3. Set up your K3s server:
```bash
k3s-deploy setup
```

4. Deploy your application:
```bash
k3s-deploy deploy
```

## Commands

- `init` - Generate a default deploy.yml configuration file
- `setup` - Install and configure K3s on your server
- `deploy` - Generate Helm templates based on deploy.yml and deploy your application to the K3s cluster

## Server Requirements

- A Linux server with SSH access
- Either password or SSH key authentication
- Sufficient permissions to install system packages
- Open ports:
  - 22 (SSH)
  - 80 (HTTP)
  - 443 (HTTPS)
  - 6443 (Kubernetes API)

## Configuration

The `deploy.yml` file supports the following configuration options:

### Service Configuration
- `service`: Name of your application
- `image`: Container image configuration
  - `name`: Image name
  - `registry`: Container registry settings
    - `server`: Registry server URL
    - `username`: Registry username
    - `password`: Registry password (from environment variable)
  - `port`: Application container port

### Server Configuration
- `server`: K3s server settings
  - `ip`: Server IP address
  - `user`: SSH username
  - `ssh_key`: Path to SSH private key
  - `password`: SSH password (alternative to SSH key)

### Traffic Configuration
- `traffic`: Domain and TLS settings
  - `domain`: Your application domain
  - `tls`: Enable HTTPS with Let's Encrypt
  - `redirect_www`: Enable www to non-www redirect
  - `email`: Email for Let's Encrypt certificate
  - `port`: Application port number

### Environment Variables
- `env.clear`: Non-sensitive environment variables
  - Can be direct values or environment variable references
- `env.secrets`: Sensitive environment variables
  - Always loaded from environment variables
  - Stored as Kubernetes secrets

## Security

- Supports both SSH key and password authentication
- Automatically configures SSL certificates via cert-manager
- Securely manages sensitive environment variables
- Uses HTTPS for all external access

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.