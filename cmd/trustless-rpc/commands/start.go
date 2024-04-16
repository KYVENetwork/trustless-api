package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/KYVENetwork/trustless-rpc/server"
	"github.com/KYVENetwork/trustless-rpc/utils"
	"github.com/spf13/cobra"
)

func init() {
	startCmd.Flags().StringVar(&chainId, "chain-id", utils.DefaultChainId, fmt.Sprintf("KYVE chain id [\"%s\",\"%s\", \"%s\"]", utils.ChainIdMainnet, utils.ChainIdKaon, utils.ChainIdKorellia))

	startCmd.Flags().StringVar(&port, "port", "", "API server port")

	startCmd.Flags().StringVar(&restEndpoint, "rest-endpoint", "", "KYVE API endpoint to retrieve validated bundles")

	startCmd.Flags().StringVar(&storageRest, "storage-rest", "", "storage endpoint for requesting bundle data")

	startCmd.Flags().BoolVar(&noCache, "no-cache", false, "Query bundles directly on request, don't use any cache")

	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the trustless RPC",
	Run: func(cmd *cobra.Command, args []string) {
		endpoint := utils.GetChainRest(chainId, restEndpoint)
		storageRest = strings.TrimSuffix(storageRest, "/")

		if os.Getenv("PORT") != "" {
			port = os.Getenv("PORT")
		}

		if port == "" {
			port = "4242"
		}

		server.StartApiServer(chainId, endpoint, storageRest, port, noCache)
	},
}
