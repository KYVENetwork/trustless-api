package helper

import (
	"encoding/json"
	"fmt"

	"github.com/KYVENetwork/trustless-api/utils"

	"github.com/KYVENetwork/trustless-api/merkle"
	"github.com/KYVENetwork/trustless-api/types"
)

type EthBlobsIndexer struct {
	DefaultIndexer
}

func (eth *EthBlobsIndexer) GetBindings() map[string]types.Endpoint {
	return map[string]types.Endpoint{
		"/beacon/blob_sidecars": {
			QueryParameter: []types.ParameterIndex{
				{
					IndexId:     utils.IndexBlockHeight,
					Parameter:   []string{"block_height"},
					Description: []string{"Ethereum block height, starting from 19426587"},
				},
				{
					IndexId:     utils.IndexSlotNumber,
					Parameter:   []string{"slot_number"},
					Description: []string{"Ethereum slot number, starting from 8626178"},
				},
			},
			Schema: "DataItem",
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
		{Index: dataitem.Key, IndexId: utils.IndexBlockHeight},
		{Index: fmt.Sprintf("%v", blobData.SlotNumber), IndexId: utils.IndexSlotNumber},
	}

	return indices, nil
}

func (e *EthBlobsIndexer) IndexBundle(bundle *types.Bundle) (*[]types.TrustlessDataItem, error) {
	leafs := merkle.GetBundleHashes(&bundle.DataItems)
	var trustlessItems []types.TrustlessDataItem
	for index, dataitem := range bundle.DataItems {
		proof, err := merkle.GetHashesCompact(leafs, index)
		if err != nil {
			return nil, err
		}
		indices, err := e.getDataItemIndices(&dataitem)
		if err != nil {
			return nil, err
		}

		encodedProof := utils.EncodeProof(bundle.PoolId, bundle.BundleId, bundle.ChainId, dataitem.Key, "value", proof)

		bytes, err := json.Marshal(dataitem)
		if err != nil {
			return nil, err
		}

		trustlessDataItem := types.TrustlessDataItem{
			Value:    bytes,
			Proof:    encodedProof,
			BundleId: bundle.BundleId,
			PoolId:   bundle.PoolId,
			ChainId:  bundle.ChainId,
			Indices:  indices,
		}
		trustlessItems = append(trustlessItems, trustlessDataItem)
	}
	return &trustlessItems, nil
}
