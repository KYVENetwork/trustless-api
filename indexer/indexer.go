package indexer

import (
	"github.com/KYVENetwork/trustless-api/indexer/helper"
	"github.com/KYVENetwork/trustless-api/types"
)

type Indexer interface {

	// TODO: right now this is super stupid as we save the same response twice if we have two indices on the same file
	//
	// this is the whole purpose of the indexer to reduce redundancy
	// it is probably the best if the database receives a form of saved file, that includes all relevant information
	// including all indicies on that file
	IndexBundle(bundle *types.Bundle) (*[]types.TrustlessDataItem, error)

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
	HeightIndexer      = helper.HeightIndexer{}
	EthBlobIndexHeight = helper.IndexBlockHeight
	EthBlobIndexSlot   = helper.IndexSlotNumber
	CelestiaIndexer    = helper.CelestiaIndexer{}
)
