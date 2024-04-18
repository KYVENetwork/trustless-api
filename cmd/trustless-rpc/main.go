package main

import (
	cmd "github.com/KYVENetwork/trustless-rpc/cmd/trustless-rpc/commands"
	"github.com/KYVENetwork/trustless-rpc/config"
)

func main() {
	config.LoadConfig()
	cmd.Execute()
}
