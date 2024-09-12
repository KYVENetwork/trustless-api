package indexer

import (
	"github.com/KYVENetwork/trustless-api/indexer/helper"
	"github.com/KYVENetwork/trustless-api/types"
)

type Indexer interface {

	// IndexBundle indexes a bundle and returns an array of trustless data items.
	// One trustless data item contains the actual data item and all necessary information to verify it:
	// - proof
	// - chainId
	// - bundleId
	//
	// Also, each trustless data item has an array indices that will be stored in the database and associated with the response.
	//
	// NOTE: 	If you want to create compound indices, you have to separate them with dashes '-' e. g.: '<blockHeight>-<namespace>'
	// 			the order has to be identical to the order defined in `GetBindings`
	IndexBundle(bundle *types.Bundle) (*[]types.TrustlessDataItem, error)

	// GetErrorResponse returns a wrapped error response
	GetErrorResponse(message string, data any) any

	// GetBindings returns a map of endpoints. Each Endpoint contains information about the url and necessary query parameter. The bindings also map QueryParamter to database indices.
	//
	// This way it is possible for an Indexer to bind to multiple urls, having an arbitrary amount of query parameter for each url.
	//
	// E.g. we want to bind to two different urls, each of them having multiple parameter and pointing to different indices
	// - "/block?block_height=1" // 1 as this is the block height index
	// - "/block?slot_number=2" // 2 as this is the slot number index
	//
	// - "/block_results?block_height=3&slot_number=3" // 3 as we want another index, but this time compound
	// NOTE: compound keys will be constructed into a single string that is joined with dashes '-' here that'd be: '<block_height>-<slot_number>'
	//
	// The corresponding map would look like this:
	// return map[string]types.Endpoint{
	// 	"/beacon/blob_sidecars": {
	// 		QueryParameter: []types.ParameterIndex{
	// 			{
	// 				IndexId:     utils.IndexBlockHeight,
	// 				Parameter:   []string{"block_height"},
	// 				Description: []string{"your query parameter description"},
	// 			},
	// 			{
	// 				IndexId:     utils.IndexSlotNumber,
	// 				Parameter:   []string{"slot_number"},
	// 				Description: []string{"your query parameter description"},
	// 			},
	// 		},
	// 		Schema: "DataItem",
	// 	},
	//  "/GetSharesByNamespace": {
	// 	 	QueryParameter: []types.ParameterIndex{
	// 	 		{
	// 	 			IndexId:     utils.IndexBlockHeightSlotNumber,
	//	 	 		Parameter:   []string{"block_height", "slot_number"},
	// 		 		Description: []string{"parameter 1 desc.", "parameter 2 desc."},
	// 	 		},
	// 		},
	// 		Schema: "DataItem",
	// 	},
	// }
	GetBindings() map[string]types.Endpoint
}

var (
	EthBlobIndexer    = helper.EthBlobsIndexer{}
	HeightIndexer     = helper.HeightIndexer{}
	CelestiaIndexer   = helper.CelestiaIndexer{}
	TendermintIndexer = helper.TendermintIndexer{}
)
