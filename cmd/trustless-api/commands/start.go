package commands

import (
	"strings"

	"github.com/KYVENetwork/trustless-api/config"

	"github.com/KYVENetwork/trustless-api/server"
	"github.com/KYVENetwork/trustless-api/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	startCmd.Flags().StringVar(&configPath, "config", "./config.yml", "sets the config that is used")

	startCmd.Flags().IntVar(&port, "port", 4242, "API server port")

	startCmd.Flags().StringVar(&mainnetEndpoint, "mainnet-endpoint", utils.RestEndpointMainnet, "KYVE API endpoint to retrieve validated bundles")

	startCmd.Flags().StringVar(&kaonEndpoint, "kaon-endpoint", utils.RestEndpointKaon, "KYVE Testnet API endpoint to retrieve validated bundles")

	startCmd.Flags().StringVar(&korelliaEndpoint, "korellia-endpoint", utils.RestEndpointKorellia, "KYVE Devnet API endpoint to retrieve validated bundles")

	startCmd.Flags().StringVar(&storageRest, "storage-rest", "", "storage endpoint for requesting bundle data")

	viper.BindPFlag("server.port", startCmd.Flags().Lookup("port"))
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Trustless API",
	Run: func(cmd *cobra.Command, args []string) {
		config.LoadConfig(configPath)
		endpointMap := map[string]string{
			"kyve-1":     mainnetEndpoint,
			"kaon-1":     kaonEndpoint,
			"korellia-2": korelliaEndpoint,
		}
		storageRest = strings.TrimSuffix(storageRest, "/")
		server.StartApiServer(endpointMap, storageRest)
	},
}
