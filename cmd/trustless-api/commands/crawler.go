package commands

import (
	"fmt"
	"os"

	"github.com/KYVENetwork/trustless-api/config"
	"github.com/KYVENetwork/trustless-api/crawler"
	"github.com/spf13/cobra"
)

func init() {
	home, _ := os.UserHomeDir()
	defaultPath := fmt.Sprintf("%v/.trustless-api/config.yml", home)
	crawlerCmd.Flags().StringVar(&configPath, "config", defaultPath, "sets the config that is used")

	rootCmd.AddCommand(crawlerCmd)
}

var crawlerCmd = &cobra.Command{
	Use:   "crawler",
	Short: "Indexes all bundles and saves them",
	Run: func(cmd *cobra.Command, args []string) {
		config.LoadConfig(configPath)
		crawler := crawler.Create()
		crawler.Start()
	},
}
