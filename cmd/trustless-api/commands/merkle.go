package commands

import (
	"fmt"

	"github.com/KYVENetwork/trustless-api/merkle"
	"github.com/KYVENetwork/trustless-api/utils"
	"github.com/spf13/cobra"
)

func init() {
	merkleCmd.Flags().StringVar(&chainId, "chain-id", utils.DefaultChainId, fmt.Sprintf("KYVE chain id [\"%s\",\"%s\", \"%s\"]", utils.ChainIdMainnet, utils.ChainIdKaon, utils.ChainIdKorellia))

	merkleCmd.Flags().Int64Var(&bundleId, "bundle-id", 0, "Bundle ID to check")
	if err := merkleCmd.MarkFlagRequired("bundle-id"); err != nil {
		panic(fmt.Errorf("flag 'bundle-id' should be required: %w", err))
	}

	merkleCmd.Flags().Int64Var(&poolId, "pool-id", 0, "Pool ID from the Bundle")
	if err := merkleCmd.MarkFlagRequired("pool-id"); err != nil {
		panic(fmt.Errorf("flag 'pool-id' should be required: %w", err))
	}
	rootCmd.AddCommand(merkleCmd)
}

var merkleCmd = &cobra.Command{
	Use:   "merkle",
	Short: "Construct and verify the merkle tree of the given bundle",
	Run: func(cmd *cobra.Command, args []string) {
		merkle.IsBundleValid(bundleId, poolId, chainId)
	},
}
