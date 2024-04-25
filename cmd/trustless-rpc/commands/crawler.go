package commands

import (
	"github.com/KYVENetwork/trustless-rpc/config"
	"github.com/KYVENetwork/trustless-rpc/crawler"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	crawlerCmd.Flags().StringVar(&configPath, "config", "./config.yml", "sets the config that is used")

	viper.BindPFlags(crawlerCmd.Flags())
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
