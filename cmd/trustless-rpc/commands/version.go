package commands

import (
	"fmt"
	"github.com/KYVENetwork/trustless-rpc/utils"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of the Celestia-KYVE-RPC",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(utils.GetVersion())
	},
}
