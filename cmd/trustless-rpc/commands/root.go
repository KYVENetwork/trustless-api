package commands

import (
	"fmt"
	"github.com/KYVENetwork/trustless-rpc/utils"
	"github.com/spf13/cobra"
)

var (
	logger = utils.TrustlessRpcLogger("commands")
)

var (
	chainId      string
	ecosystem    string
	port         string
	restEndpoint string
	storageRest  string
)

// RootCmd is the root command for trustless-rpc.
var rootCmd = &cobra.Command{
	Use:   "trustless-rpc",
	Short: "The first trustless RPC, providing validated data through KYVE.",
}

func Execute() {
	versionCmd.Flags().SortFlags = false

	if err := rootCmd.Execute(); err != nil {
		panic(fmt.Errorf("failed to execute root command: %w", err))
	}
}
