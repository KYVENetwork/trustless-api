package commands

import (
	"fmt"

	"github.com/KYVENetwork/trustless-rpc/crawler"
	"github.com/KYVENetwork/trustless-rpc/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	crawlerCmd.Flags().StringVar(&chainId, "chain-id", utils.DefaultChainId, fmt.Sprintf("KYVE chain id [\"%s\",\"%s\", \"%s\"]", utils.ChainIdMainnet, utils.ChainIdKaon, utils.ChainIdKorellia))

	viper.BindPFlags(crawlerCmd.Flags())
	rootCmd.AddCommand(crawlerCmd)
}

var crawlerCmd = &cobra.Command{
	Use:   "crawler",
	Short: "Indexes all bundles and saves them",
	Run: func(cmd *cobra.Command, args []string) {
		crawler := crawler.Create()
		crawler.Start()
	},
}
