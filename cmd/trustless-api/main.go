package main

import (
	cmd "github.com/KYVENetwork/trustless-api/cmd/trustless-api/commands"
	"github.com/KYVENetwork/trustless-api/config"
)

func main() {
	config.LoadConfig("config.yml")
	cmd.Execute()
}
