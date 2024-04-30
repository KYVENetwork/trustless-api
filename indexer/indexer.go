package indexer

import (
	"github.com/KYVENetwork/trustless-api/indexer/helper"
	"github.com/KYVENetwork/trustless-api/types"
)

type Indexer interface {
	GetDataItemIndices(dataitem *types.TrustlessDataItem) ([]int64, error)
	GetBindings() map[string]map[string]int64
}

var (
	EthBlobIndexer     = helper.EthBlobsIndexer{}
	EthBlobIndexHeight = 0
	EthBlobIndexSlot   = 1
	HeightIndexer      = helper.HeightIndexer{}
	HeightIndexHeight  = 0
)
