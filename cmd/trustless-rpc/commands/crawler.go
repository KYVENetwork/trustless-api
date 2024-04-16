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

	rootCmd.AddCommand(crawlerCmd)
}

var crawlerCmd = &cobra.Command{
	Use:   "crawler",
	Short: "Indexes all bundles and saves them",
	Run: func(cmd *cobra.Command, args []string) {
		endpoint := utils.GetChainRest(chainId, restEndpoint)
		storageRest = strings.TrimSuffix(storageRest, "/")

		sqliteAdapter := adapters.StartSQLite(&db.SaveLocalFileInterface{}, &indexer.EthBlobIndexer)

		crawler := crawler.Create(endpoint, storageRest, &sqliteAdapter, 21)
		crawler.Start()
	},
}
