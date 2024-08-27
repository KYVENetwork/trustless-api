package helper

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/KYVENetwork/trustless-api/utils"

	"github.com/KYVENetwork/trustless-api/merkle"
	"github.com/KYVENetwork/trustless-api/types"
)

type TendermintIndexer struct{}

func (t *TendermintIndexer) GetBindings() map[string][]types.ParameterIndex {
	return map[string][]types.ParameterIndex{
		"/block": {
			{
				IndexId:     utils.IndexTendermintBlock,
				Parameter:   []string{"height"},
				Description: []string{"block height"},
			},
		},
		"/block_results": {
			{
				IndexId:     utils.IndexTendermintBlockResults,
				Parameter:   []string{"height"},
				Description: []string{"block height"},
			},
		},
	}
}

func (t *TendermintIndexer) CalculateProof(dataItem *types.TendermintDataItem, leafs [][32]byte, dataItemIndex int) ([]types.MerkleNode, []types.MerkleNode, error) {
	// Create proof for API response.
	proof, err := merkle.GetHashesCompact(&leafs, dataItemIndex)
	if err != nil {
		return nil, nil, err
	}

	var tendermintHashes [][32]byte

	blockHash := utils.CalculateSHA256Hash(dataItem.Value.Block)

	if dataItem.Key == "1" {
		fmt.Println(hex.EncodeToString(blockHash[:]))
	}
	tendermintHashes = append(tendermintHashes, utils.CalculateSHA256Hash(dataItem.Value.Block))
	tendermintHashes = append(tendermintHashes, utils.CalculateSHA256Hash(dataItem.Value.BlockResults))

	blockProof, err := merkle.GetHashesCompact(&tendermintHashes, 0)
	if err != nil {
		return nil, nil, err
	}

	blockResultsProof, err := merkle.GetHashesCompact(&tendermintHashes, 1)
	if err != nil {
		return nil, nil, err
	}

	// Because we also hash the key of the original data item, we have to append an extra leaf with the key
	keyBytes := sha256.Sum256([]byte(dataItem.Key))
	keyHash := hex.EncodeToString(keyBytes[:])

	totalBlockProof := append(blockProof, types.MerkleNode{Left: false, Hash: keyHash})

	// Append the proof for the rest of the data items
	totalBlockProof = append(totalBlockProof, proof...)

	totalBlockResultsProof := append(blockResultsProof, types.MerkleNode{Left: false, Hash: keyHash})

	// Append the proof for the rest of the data items
	totalBlockResultsProof = append(totalBlockResultsProof, proof...)

	return totalBlockProof, totalBlockResultsProof, nil
}

func (t *TendermintIndexer) IndexBundle(bundle *types.Bundle, proofAttached bool) (*[]types.TrustlessDataItem, error) {
	var dataItems []types.TendermintDataItem
	var leafs [][32]byte

	for _, item := range bundle.DataItems {
		var tendermintValue types.TendermintValue
		if err := json.Unmarshal(item.Value, &tendermintValue); err != nil {
			return nil, err
		}

		tendermintItem := types.TendermintDataItem{Key: item.Key, Value: tendermintValue}

		if proofAttached {
			leafs = append(leafs, t.tendermintDataItemToSha256(&tendermintItem))
		}

		dataItems = append(dataItems, tendermintItem)
	}

	var trustlessItems []types.TrustlessDataItem
	for index, dataItem := range dataItems {

		var blockProof, blockResultsProof []types.MerkleNode
		if proofAttached {
			var err error
			blockProof, blockResultsProof, err = t.CalculateProof(&dataItem, leafs, index)
			if err != nil {
				return nil, err
			}
		}

		createTrustlessDataItem := func(value json.RawMessage, indexId int, proof []types.MerkleNode) (types.TrustlessDataItem, error) {

			encodedProof := utils.EncodeProof(bundle.PoolId, bundle.BundleId, bundle.ChainId, dataItem.Key, "result", proof)
			// if proof is not attached, we set the proof to an empty string
			if !proofAttached {
				encodedProof = ""
			}

			rpcResponse, err := utils.WrapIntoJsonRpcResponse(value)
			if err != nil {
				return types.TrustlessDataItem{}, err
			}

			return types.TrustlessDataItem{
				Value:    rpcResponse,
				Proof:    encodedProof,
				BundleId: bundle.BundleId,
				PoolId:   bundle.PoolId,
				Indices: []types.Index{
					{
						Index:   dataItem.Key,
						IndexId: indexId,
					},
				},
			}, nil
		}
		// Create and append trustless data items for block and block_results
		blockTrustlessItem, err := createTrustlessDataItem(dataItem.Value.Block, utils.IndexTendermintBlock, blockProof)
		if err != nil {
			return nil, err
		}

		blockResultsTrustlessItem, err := createTrustlessDataItem(dataItem.Value.BlockResults, utils.IndexTendermintBlockResults, blockResultsProof)
		if err != nil {
			return nil, err
		}

		trustlessItems = append(trustlessItems, blockTrustlessItem)
		trustlessItems = append(trustlessItems, blockResultsTrustlessItem)
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
