#!/usr/bin/make -f

celestia-kyve-rpc:
	go build -mod=readonly -o ./build/trustless-rpc ./cmd/trustless-rpc/main.go