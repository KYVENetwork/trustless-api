package commands

import (
	"fmt"
	"os"

	"github.com/KYVENetwork/trustless-api/config"

	"github.com/KYVENetwork/trustless-api/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	home, _ := os.UserHomeDir()
	defaultPath := fmt.Sprintf("%v/.trustless-api/config.yml", home)
	startCmd.Flags().StringVar(&configPath, "config", defaultPath, "sets the config that is used")

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
