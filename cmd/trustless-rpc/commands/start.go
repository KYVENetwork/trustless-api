package commands

import (
	"fmt"
	"strings"

	"github.com/KYVENetwork/trustless-rpc/server"
	"github.com/KYVENetwork/trustless-rpc/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	startCmd.Flags().StringVar(&chainId, "chain-id", utils.DefaultChainId, fmt.Sprintf("KYVE chain id [\"%s\",\"%s\", \"%s\"]", utils.ChainIdMainnet, utils.ChainIdKaon, utils.ChainIdKorellia))

	startCmd.Flags().IntVar(&port, "port", 4242, "API server port")

	startCmd.Flags().StringVar(&restEndpoint, "rest-endpoint", "", "KYVE API endpoint to retrieve validated bundles")

	startCmd.Flags().StringVar(&storageRest, "storage-rest", "", "storage endpoint for requesting bundle data")

	startCmd.Flags().BoolVar(&noCache, "no-cache", false, "Query bundles directly on request, don't use any cache")

	viper.BindPFlag("server.no-cache", startCmd.Flags().Lookup("no-cache"))
	viper.BindPFlag("chain-id", startCmd.Flags().Lookup("chain-id"))
	viper.BindPFlag("server.port", startCmd.Flags().Lookup("port"))
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the trustless RPC",
	Run: func(cmd *cobra.Command, args []string) {
		chainId := viper.GetString("chain-id")
		endpoint := utils.GetChainRest(chainId, restEndpoint)
		storageRest = strings.TrimSuffix(storageRest, "/")

		server.StartApiServer(chainId, endpoint, storageRest, viper.GetInt("port"), viper.GetBool("no-cache"))
	},
}
