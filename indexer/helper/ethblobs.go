package helper

import (
	"encoding/json"
	"fmt"

	"github.com/KYVENetwork/trustless-api/merkle"
	"github.com/KYVENetwork/trustless-api/types"
	"github.com/KYVENetwork/trustless-api/utils"
)

type EthBlobsIndexer struct{}

const (
	IndexBlockHeight = 0
	IndexSlotNumber  = 1
)

func (eth *EthBlobsIndexer) GetBindings() map[string][]types.ParameterIndex {
	return map[string][]types.ParameterIndex{
		"/beacon/blob_sidecars": {
			{
				IndexId:   IndexBlockHeight,
				Parameter: []string{"block_height"},
			},
			{
				IndexId:   IndexSlotNumber,
				Parameter: []string{"slot_number"},
			},
		},
	}
}

func (*EthBlobsIndexer) getDataItemIndices(dataitem *types.DataItem) ([]types.Index, error) {
	// Create a struct to unmarshal into
	var blobData types.BlobValue

	// Unmarshal the RawMessage into the struct
	err := json.Unmarshal(dataitem.Value, &blobData)
	if err != nil {
		return nil, err
	}
	var indices []types.Index = []types.Index{
		{Index: dataitem.Key, IndexId: IndexBlockHeight},
		{Index: fmt.Sprintf("%v", blobData.SlotNumber), IndexId: IndexSlotNumber},
	}

	return indices, nil
}

func (e *EthBlobsIndexer) IndexBundle(bundle *types.Bundle) (*[]types.TrustlessDataItem, error) {
	leafs := merkle.GetBundleHashes(&bundle.DataItems)
	var trustlessItems []types.TrustlessDataItem
	for index, dataitem := range bundle.DataItems {
		leafHash := utils.CalculateSHA256Hash(dataitem)
		proof, err := merkle.GetHashesCompact(leafs, &leafHash)
		if err != nil {
			return nil, err
		}
		Indices, err := e.getDataItemIndices(&dataitem)
		if err != nil {
			return nil, err
		}

		raw, err := json.Marshal(bundle.DataItems[index])
		if err != nil {
			return nil, err
		}

		trustlessDataItem := types.TrustlessDataItem{
			Value:    raw,
			Proof:    proof,
			BundleId: bundle.BundleId,
			PoolId:   bundle.PoolId,
			ChainId:  bundle.ChainId,
			Indices:  Indices,
		}
		trustlessItems = append(trustlessItems, trustlessDataItem)
	}
	return &trustlessItems, nil
}
