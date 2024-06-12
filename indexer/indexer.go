package indexer

import (
	"github.com/KYVENetwork/trustless-api/indexer/helper"
	"github.com/KYVENetwork/trustless-api/types"
)

type Indexer interface {

	// indexes a bundle and returns an array of trustless data items
	// one trustless data item contains the actual data item and all necessary information to verify it:
	// - proof
	// - chainId
	// - bundleId
	//
	// Also each trustless data item has an array indices that will be stored in the data base and associated with the response
	//
	// NOTE: 	if you want to create compound indices, you have to seperate them with dashes '-' e. g.: '<blockHeight>-<namespace>'
	// 			the order has to be identical to the order defined in `GetBindings`
	IndexBundle(bundle *types.Bundle) (*[]types.TrustlessDataItem, error)

	// GetBindings returns a map of urls of query params
	//
	// This way it is possible for an Indexer to bind to multiple urls, having an arbitrary amount of query parameter for each url
	//
	// e. g. we want to bind to two different urls, each of them having multiple parameter and pointing to different indices
	// - "/block?block_height=1" // 1 as this is the block height index
	// - "/block?slot_number=2" // 2 as this is the slot number index
	//
	// - "/block_results?block_height=3&slot_number=3" // 3 as we want another index, but this time compound
	// NOTE: compound keys will be constructed into a single string that is joined with dashes '-' here that'd be: '<block_height>-<slot_number>'
	//
	// the corresponding map would look like this:
	// return map[string][]types.ParameterIndex{
	// 	"/block": {
	// 		{
	// 			IndexId:   1,
	// 			Parameter: []string{"block_height"},
	// 		},
	// 		{
	// 			IndexId:   2,
	// 			Parameter: []string{"slot_number"},
	// 		},
	// 	},
	// 	"/block_results": {
	// 		{
	// 			IndexId:   3,
	// 			Parameter: []string{"block_height", "slot_number"},
	// 		},
	// 	},
	// }
	// }
	GetBindings() map[string][]types.ParameterIndex
}

var (
	EthBlobIndexer    = helper.EthBlobsIndexer{}
	HeightIndexer     = helper.HeightIndexer{}
	CelestiaIndexer   = helper.CelestiaIndexer{}
	TendermintIndexer = helper.TendermintIndexer{}
)
