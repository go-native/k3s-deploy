package cmd

import (
	"os"

	"github.com/go-native/k3s-deploy/cmd/commands/deploy"
	initcmd "github.com/go-native/k3s-deploy/cmd/commands/init"
	"github.com/go-native/k3s-deploy/cmd/commands/setup"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "k3s-deploy",
	Short: "Automate deployments to single-node K3s servers",
	Long: `k3s-deploy is a CLI tool that simplifies the process of deploying applications 
to single-node K3s servers. It automates the entire deployment workflow including:

- Server setup and K3s installation
- Container registry configuration
- SSL certificate management with cert-manager
- Environment variable and secret management
- Domain configuration and ingress setup`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initcmd.NewCommand())
	rootCmd.AddCommand(setup.NewCommand())
	rootCmd.AddCommand(deploy.NewCommand())
}
