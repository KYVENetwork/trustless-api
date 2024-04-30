package indexer

import (
	"github.com/KYVENetwork/trustless-api/indexer/helper"
	"github.com/KYVENetwork/trustless-api/types"
)

type Indexer interface {

	// returns an array of indices for the given data item
	GetDataItemIndices(dataitem *types.TrustlessDataItem) ([]int64, error)

	// GetBindings returns a map of urls of query params to index
	// NOTE: that the index should point to the correct index that is returned in GetDataItemIndices()
	//
	// This way it is possible for an Indexer to bind to multiple urls, having an arbitrary amount of query parameter for each url
	//
	// e. g. we want to bind to two different urls, each of them having multiple parameter
	// - "/block?block_height=1"
	// - "/block?slot_number=1"
	//
	// - "/block_results?block_height=1"
	// - "/block_results?slot_number=1"
	//
	// the corresponding map would look like this:
	// return map[string]map[string]int64{
	// 	"/block": {
	// 		"block_height": IndexBlockHeight,
	// 		"slot_number":  IndexSlotNumber,
	// 	},
	// 	"/block_results": {
	// 		"block_height": IndexBlockHeight,
	// 		"slot_number":  IndexSlotNumber,
	// 	},
	// }
	GetBindings() map[string]map[string]int64
}

var (
	EthBlobIndexer     = helper.EthBlobsIndexer{}
	EthBlobIndexHeight = helper.IndexBlockHeight
	EthBlobIndexSlot   = helper.IndexSlotNumber
	HeightIndexer      = helper.HeightIndexer{}
	HeightIndexHeight  = helper.IndexBlockHeight
)
