package commands

import (
	"github.com/KYVENetwork/trustless-api/config"
	"github.com/KYVENetwork/trustless-api/crawler"
	"github.com/spf13/cobra"
)

func init() {
	crawlerCmd.Flags().StringVar(&configPath, "config", "./config.yml", "sets the config that is used")

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
