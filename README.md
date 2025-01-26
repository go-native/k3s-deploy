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

## Quick Start

1. Initialize configuration:
```bash
k3s-deploy init
```

2. Edit the generated `deploy.yml` file with your server details:
```yaml
service: my-app
image:
  name: my-user/my-app
  registry:
    server: ghcr.io
    username: my-user
    password: my-password
server: 
  ip: 192.168.1.100
  user: root
  ssh_key: ~/.ssh/id_rsa
  # password: optional_password  # Alternative to SSH key

domain: example.com
redirect_www: true
env:
  clear:
    DB_HOST: localhost
  secrets:
    DB_PASSWORD:
      fromFile: .env
      fromEnv: DB_PASSWORD
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
- `deploy` - Deploy your application to the K3s cluster

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
    - `password`: Registry password

### Server Configuration
- `server`: K3s server settings
  - `ip`: Server IP address
  - `user`: SSH username
  - `ssh_key`: Path to SSH private key
  - `password`: SSH password (alternative to SSH key)

### Domain Configuration
- `domain`: Your application domain
- `redirect_www`: Enable www to non-www redirect

### Environment Variables
- `env.clear`: Non-sensitive environment variables
- `env.secrets`: Sensitive environment variables
  - Support for loading from environment or file

## Security

- Supports both SSH key and password authentication
- Automatically configures SSL certificates via cert-manager
- Securely manages sensitive environment variables
- Uses HTTPS for all external access

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.