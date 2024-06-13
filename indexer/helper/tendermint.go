package helper

import (
	"crypto/sha256"
	"encoding/json"

	"github.com/KYVENetwork/trustless-api/utils"

	"github.com/KYVENetwork/trustless-api/merkle"
	"github.com/KYVENetwork/trustless-api/types"
)

type TendermintIndexer struct{}

func (t *TendermintIndexer) GetBindings() map[string][]types.ParameterIndex {
	return map[string][]types.ParameterIndex{
		"/block": {
			{
				IndexId:   utils.IndexTendermintBlock,
				Parameter: []string{"height"},
			},
		},
		"/block_results": {
			{
				IndexId:   utils.IndexTendermintBlockResults,
				Parameter: []string{"height"},
			},
		},
	}
}

func (t *TendermintIndexer) IndexBundle(bundle *types.Bundle) (*[]types.TrustlessDataItem, error) {
	var dataItems []types.TendermintDataItem
	var leafs [][32]byte

	for _, item := range bundle.DataItems {
		leafs = append(leafs, t.tendermintDataItemToSha256(&item))

		var tendermintValue types.TendermintValue

		if err := json.Unmarshal(item.Value, &tendermintValue); err != nil {
			return nil, err
		}

		tendermintItem := types.TendermintDataItem{Key: item.Key, Value: tendermintValue}

		dataItems = append(dataItems, tendermintItem)
	}

	var trustlessItems []types.TrustlessDataItem
	for index, dataItem := range dataItems {
		// Create proof for API response.
		proof, err := merkle.GetHashesCompact(&leafs, index)
		if err != nil {
			return nil, err
		}

		// Extract block and block_results from data item
		blockValue, err := json.Marshal(dataItem.Value.Block)
		if err != nil {
			return nil, err
		}

		blockResultsValue, err := json.Marshal(dataItem.Value.BlockResults)
		if err != nil {
			return nil, err
		}

		createTrustlessDataItem := func(value []byte, indexId int) types.TrustlessDataItem {
			return types.TrustlessDataItem{
				Value:    value,
				Proof:    proof,
				BundleId: bundle.BundleId,
				PoolId:   bundle.PoolId,
				ChainId:  bundle.ChainId,
				Indices: []types.Index{
					{
						Index:   dataItem.Key,
						IndexId: indexId,
					},
				},
			}
		}

		// Create and append trustless data items for block and block_results
		trustlessItems = append(trustlessItems, createTrustlessDataItem(blockValue, utils.IndexTendermintBlock))
		trustlessItems = append(trustlessItems, createTrustlessDataItem(blockResultsValue, utils.IndexTendermintBlockResults))
	}
	return &trustlessItems, nil
}

func (*TendermintIndexer) tendermintDataItemToSha256(dataItem *types.DataItem) [32]byte {

	merkleRoot := createHashesForTendermintValue(dataItem)

	keyBytes := sha256.Sum256([]byte(dataItem.Key))

	combined := append(keyBytes[:], merkleRoot[:]...)

	return sha256.Sum256(combined)
}

func createHashesForTendermintValue(dataItem *types.DataItem) [32]byte {
	var tendermintValue types.TendermintValue

	if err := json.Unmarshal(dataItem.Value, &tendermintValue); err != nil {
		panic(err)
	}

	var hashes [][32]byte

	hashes = append(hashes, utils.CalculateSHA256Hash(tendermintValue.Block))
	hashes = append(hashes, utils.CalculateSHA256Hash(tendermintValue.BlockResults))

	return merkle.GetMerkleRoot(hashes)
}
