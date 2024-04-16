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
	crawlerCmd.Flags().StringVar(&chainId, "chain-id", utils.DefaultChainId, fmt.Sprintf("KYVE chain id [\"%s\",\"%s\", \"%s\"]", utils.ChainIdMainnet, utils.ChainIdKaon, utils.ChainIdKorellia))
	crawlerCmd.Flags().StringVar(&dbPath, "db-path", "./database.db", "the path where the db will be located")

	rootCmd.AddCommand(crawlerCmd)
}

var crawlerCmd = &cobra.Command{
	Use:   "crawler",
	Short: "Indexes all bundles and saves them",
	Run: func(cmd *cobra.Command, args []string) {
		endpoint := utils.GetChainRest(chainId, restEndpoint)
		storageRest = strings.TrimSuffix(storageRest, "/")

		dataDir := "./bundles"
		sqliteAdapter, err := adapters.StartSQLite(dbPath, &db.SaveLocalFileInterface{DataDir: dataDir}, &indexer.EthBlobIndexer)
		if err != nil {
			logger.Fatal().Err(err)
			return
		}

		crawler := crawler.Create(endpoint, storageRest, &sqliteAdapter, 21)
		crawler.Start()
	},
}
