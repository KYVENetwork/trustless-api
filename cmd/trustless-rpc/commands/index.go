package commands

import (
	"fmt"
	"github.com/KYVENetwork/trustless-rpc/indexer"
	"github.com/KYVENetwork/trustless-rpc/utils"
	"github.com/spf13/cobra"
	"strconv"
	"strings"
)

func init() {
	indexCmc.Flags().StringVar(&dbPath, "db-path", ".trustless-rpc/key_bundle_map.db", "path to SQLite DB")

	indexCmc.Flags().StringVar(&chainId, "chain-id", utils.DefaultChainId, fmt.Sprintf("KYVE chain id [\"%s\",\"%s\", \"%s\"]", utils.ChainIdMainnet, utils.ChainIdKaon, utils.ChainIdKorellia))

	indexCmc.Flags().StringVar(&restEndpoint, "rest-endpoint", "", "KYVE API endpoint to retrieve validated bundles")

	indexCmc.Flags().StringVar(&storageRest, "storage-rest", "", "storage endpoint for requesting bundle data")

	rootCmd.AddCommand(indexCmc)
}

var indexCmc = &cobra.Command{
	Use:   "index",
	Short: "Index keys to bundle ids for certain pools",
	Run: func(cmd *cobra.Command, args []string) {
		endpoint := utils.GetChainRest(chainId, restEndpoint)
		storageRest = strings.TrimSuffix(storageRest, "/")

		i := indexer.SQLiteIndexer{
			ChainId:      chainId,
			DbPath:       dbPath,
			RestEndpoint: endpoint,
		}

		logger.Info().Msg("setting up db")

		if err := i.SetupDB(true); err != nil {
			panic(err)
		}

		logger.Info().Msg("successfully set up db")

		if err := i.CreateIndex([]int64{5}); err != nil {
			panic(err)
		}

		logger.Info().Msg("created index")

		key, err := i.GetLatestKey(7)
		if err != nil {
			panic(err)
		}

		logger.Info().Msg(strconv.Itoa(key))
		//
		//bundleId, err := i.GetBundleIdByKey(1342371, 7)
		//if err != nil {
		//	panic(err)
		//}
		//
		//logger.Info().Str("key", strconv.Itoa(1342371)).Str("bundleId", strconv.Itoa(bundleId)).Msg("Found key for bundle")
	},
}
