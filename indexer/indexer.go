package indexer

import (
	"github.com/KYVENetwork/trustless-rpc/indexer/helper"
	"github.com/KYVENetwork/trustless-rpc/types"
)

type Indexer interface {
	GetIndexCount() int
	GetDataItemIndicies(dataitem *types.TrustlessDataItem) ([]string, error)
}

var (
	EthBlobIndexer     = helper.EthBlobIndexer{}
	EthBlobIndexHeight = 0
	EthBlobIndexSlot   = 1
)
