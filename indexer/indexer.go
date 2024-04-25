package indexer

import (
	"github.com/KYVENetwork/trustless-rpc/indexer/helper"
	"github.com/KYVENetwork/trustless-rpc/types"
)

type Indexer interface {
	GetDataItemIndices(dataitem *types.TrustlessDataItem) ([]int64, error)
}

var (
	EthBlobIndexer     = helper.EthBlobIndexer{}
	EthBlobIndexHeight = 0
	EthBlobIndexSlot   = 1
	HeightIndexer      = helper.HeightIndexer{}
	HeightIndexHeight  = 0
)
