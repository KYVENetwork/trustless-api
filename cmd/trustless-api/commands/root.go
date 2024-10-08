package commands

import (
	"fmt"

	"github.com/KYVENetwork/trustless-api/utils"
	"github.com/spf13/cobra"
)

var (
	logger = utils.TrustlessApiLogger("commands")
)

var (
	configPath string
	port       int
)

// RootCmd is the root command for trustless-api.
var rootCmd = &cobra.Command{
	Use:   "trustless-api",
	Short: "The first Trustless API, providing validated data through KYVE.",
}

func Execute() {
	versionCmd.Flags().SortFlags = false

	if err := rootCmd.Execute(); err != nil {
		panic(fmt.Errorf("failed to execute root command: %w", err))
	}
}
