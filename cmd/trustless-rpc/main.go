package main

import (
	cmd "github.com/KYVENetwork/trustless-rpc/cmd/trustless-rpc/commands"
	"github.com/joho/godotenv"
)

func main() {

	godotenv.Load()

	cmd.Execute()
}
