package commands

import (
	"github.com/KYVENetwork/trustless-api/config"

	"github.com/KYVENetwork/trustless-api/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	startCmd.Flags().StringVar(&configPath, "config", "./config.yml", "sets the config that is used")

	startCmd.Flags().IntVar(&port, "port", 4242, "API server port")

	viper.BindPFlag("server.port", startCmd.Flags().Lookup("port"))
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Trustless API",
	Run: func(cmd *cobra.Command, args []string) {
		config.LoadConfig(configPath)
		server.StartApiServer()
	},
}
