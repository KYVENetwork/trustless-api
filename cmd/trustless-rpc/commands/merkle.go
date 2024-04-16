package commands

import (
	"fmt"
	"strings"

	"github.com/KYVENetwork/trustless-rpc/crawler"
	"github.com/KYVENetwork/trustless-rpc/db"
	"github.com/KYVENetwork/trustless-rpc/db/adapters"
	"github.com/KYVENetwork/trustless-rpc/indexer"
	"github.com/KYVENetwork/trustless-rpc/utils"
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
		endpoint := utils.GetChainRest(chainId, restEndpoint)
		storageRest = strings.TrimSuffix(storageRest, "/")
		// merkle.IsBundleValid(bundleId, poolId, endpoint, storageRest)
		dataDir := "./bundles"
		sqliteAdapter, err := adapters.StartSQLite(dataDir, &db.SaveLocalFileInterface{DataDir: dataDir}, &indexer.EthBlobIndexer)
		if err != nil {
			logger.Fatal().Err(err)
			return
		}

		crawler := crawler.Create(endpoint, storageRest, &sqliteAdapter, poolId)
		crawler.Start()
	},
}
