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

func (t *TendermintIndexer) GetBindings() map[string]types.Endpoint {
	return map[string]types.Endpoint{
		"/block": {
			QueryParameter: []types.ParameterIndex{
				{
					IndexId:     utils.IndexTendermintBlock,
					Parameter:   []string{"height"},
					Description: []string{"block height"},
				},
			},
			Schema: "TendermintBlock",
		},
		"/block_results": {
			QueryParameter: []types.ParameterIndex{
				{
					IndexId:     utils.IndexTendermintBlockResults,
					Parameter:   []string{"height"},
					Description: []string{"block height"},
				},
			},
			Schema: "TendermintBlockResults",
		},
		"/block_by_hash": {
			QueryParameter: []types.ParameterIndex{
				{
					IndexId:     utils.IndexTendermintBlockByHash,
					Parameter:   []string{"hash"},
					Description: []string{"block hash"},
				},
			},
			Schema: "TendermintBlock",
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

func (t *TendermintIndexer) getBlockHash(dataItem *types.TendermintDataItem) (string, error) {
	var blockHash types.TendermintBlock
	err := json.Unmarshal(dataItem.Value.Block, &blockHash)
	if err != nil {
		return "", err
	}
	return blockHash.BlockId.Hash, nil
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

	// we have 2 turstless items per normal data item
	trustlessItems := make([]types.TrustlessDataItem, len(dataItems)*2)
	for index, dataItem := range dataItems {

		blockProof, blockResultsProof, err := t.CalculateProof(&dataItem, leafs, index)
		if err != nil {
			return nil, err
		}

		insertTurstlessDataItem := func(value *json.RawMessage, proof []types.MerkleNode, indices []types.Index, dest *types.TrustlessDataItem) error {

			encodedProof := utils.EncodeProof(bundle.PoolId, bundle.BundleId, bundle.ChainId, "", "result", proof)

			rpcResponse, err := utils.WrapIntoJsonRpcResponse(*value)
			if err != nil {
				return err
			}

			*dest = types.TrustlessDataItem{
				Value:    rpcResponse,
				Proof:    encodedProof,
				BundleId: bundle.BundleId,
				PoolId:   bundle.PoolId,
				ChainId:  bundle.ChainId,
				Indices:  indices,
			}
			return nil
		}

		blockHash, err := t.getBlockHash(&dataItem)
		if err != nil {
			return nil, err
		}

		// Create and append trustless data items for block and block_results
		err = insertTurstlessDataItem(&dataItem.Value.Block, blockProof, []types.Index{
			{
				Index:   dataItem.Key,
				IndexId: utils.IndexTendermintBlock,
			},
			{
				Index:   blockHash,
				IndexId: utils.IndexTendermintBlockByHash,
			},
		}, &trustlessItems[index])

		if err != nil {
			return nil, err
		}

		err = insertTurstlessDataItem(&dataItem.Value.BlockResults, blockResultsProof, []types.Index{
			{
				Index:   dataItem.Key,
				IndexId: utils.IndexTendermintBlockResults,
			},
		}, &trustlessItems[index+len(dataItems)])

		if err != nil {
			return nil, err
		}
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

func (t *TendermintIndexer) GetErrorResponse(message string, data any) any {
	return utils.WrapIntoJsonRpcErrorResponse(message, data)
}
