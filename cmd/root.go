/*
Copyright © 2025 Taron Mehrabyan <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

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

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.indie-deploy.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
