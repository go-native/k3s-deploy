package init

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate deploy.yml configuration file",
		Long:  `Generate a deploy.yml configuration file in the current directory with default values`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateConfig()
		},
	}
	return cmd
}

func generateConfig() error {
	// Check if deploy.yml already exists
	if _, err := os.Stat("deploy.yml"); err == nil {
		return fmt.Errorf("deploy.yml already exists. Please remove it before running this command again")
	}

	configTemplate := `service: my-app # This becomes the name in the Chart.yaml 
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
  port: 8080 # Application container port
env:
  clear:
    DB_HOST: localhost
  secrets:
    - DB_PASSWORD
`
	return os.WriteFile("deploy.yml", []byte(configTemplate), 0644)
}
