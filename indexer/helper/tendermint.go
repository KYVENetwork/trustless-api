package helper

import (
	"crypto/sha256"
	"encoding/hex"
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
		var tendermintValue types.TendermintValue
		if err := json.Unmarshal(item.Value, &tendermintValue); err != nil {
			return nil, err
		}

		tendermintItem := types.TendermintDataItem{Key: item.Key, Value: tendermintValue}

		leafs = append(leafs, t.tendermintDataItemToSha256(&tendermintItem))

		dataItems = append(dataItems, tendermintItem)
	}

	var trustlessItems []types.TrustlessDataItem
	for index, dataItem := range dataItems {
		// Create proof for API response.
		proof, err := merkle.GetHashesCompact(&leafs, index)
		if err != nil {
			return nil, err
		}

		var tendermintHashes [][32]byte

		tendermintHashes = append(tendermintHashes, utils.CalculateSHA256Hash(dataItem.Value.Block))
		tendermintHashes = append(tendermintHashes, utils.CalculateSHA256Hash(dataItem.Value.BlockResults))

		blockProof, err := merkle.GetHashesCompact(&tendermintHashes, 0)
		if err != nil {
			return nil, err
		}

		blockResultsProof, err := merkle.GetHashesCompact(&tendermintHashes, 1)
		if err != nil {
			return nil, err
		}

		// Because we also hash the key of the original data item, we have to append an extra leaf with the key
		keyBytes := sha256.Sum256([]byte(dataItem.Key))
		keyHash := hex.EncodeToString(keyBytes[:])

		// Extract block and block_results from data item
		blockValue, err := json.Marshal(dataItem.Value.Block)
		if err != nil {
			return nil, err
		}

		blockResultsValue, err := json.Marshal(dataItem.Value.BlockResults)
		if err != nil {
			return nil, err
		}

		createTrustlessDataItem := func(value []byte, indexId int, tendermintProof []types.MerkleNode) types.TrustlessDataItem {
			totalProof := append(tendermintProof, types.MerkleNode{Left: false, Hash: keyHash})

			// Append the proof for the rest of the data items
			totalProof = append(totalProof, proof...)

			return types.TrustlessDataItem{
				Value:    value,
				Proof:    totalProof,
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
		trustlessItems = append(trustlessItems, createTrustlessDataItem(blockValue, utils.IndexTendermintBlock, blockProof))
		trustlessItems = append(trustlessItems, createTrustlessDataItem(blockResultsValue, utils.IndexTendermintBlockResults, blockResultsProof))
	}
	return &trustlessItems, nil
}

func (*TendermintIndexer) tendermintDataItemToSha256(dataItem *types.TendermintDataItem) [32]byte {
	merkleRoot := createHashesForTendermintValue(&dataItem.Value)

	keyBytes := sha256.Sum256([]byte(dataItem.Key))

	combined := append(keyBytes[:], merkleRoot[:]...)

	return sha256.Sum256(combined)
}

func createHashesForTendermintValue(value *types.TendermintValue) [32]byte {
	var hashes [][32]byte

	hashes = append(hashes, utils.CalculateSHA256Hash(value.Block))
	hashes = append(hashes, utils.CalculateSHA256Hash(value.BlockResults))

	return merkle.GetMerkleRoot(hashes)
}
