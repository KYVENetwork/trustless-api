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

func (eth *EthBlobsIndexer) GetBindings() map[string]map[string]int64 {
	return map[string]map[string]int64{
		"/beacon/blob_sidecars": {
			"block_height": IndexBlockHeight,
			"slot_number":  IndexSlotNumber,
		},
	}
}

func (*EthBlobsIndexer) getDataItemKeys(dataitem *types.DataItem) ([]string, error) {
	// Create a struct to unmarshal into
	var blobData types.BlobValue

	// Unmarshal the RawMessage into the struct
	err := json.Unmarshal(dataitem.Value, &blobData)
	if err != nil {
		return nil, err
	}
	var indices []string = []string{
		dataitem.Key,
		fmt.Sprintf("%v", blobData.SlotNumber),
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
		keys, err := e.getDataItemKeys(&dataitem)
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
			Keys:     []string{keys[0]},
			IndexId:  IndexBlockHeight,
		}
		trustlessItems = append(trustlessItems, trustlessDataItem)

		trustlessDataItem = types.TrustlessDataItem{
			Value:    raw,
			Proof:    proof,
			BundleId: bundle.BundleId,
			PoolId:   bundle.PoolId,
			ChainId:  bundle.ChainId,
			Keys:     []string{keys[1]},
			IndexId:  IndexSlotNumber,
		}
		trustlessItems = append(trustlessItems, trustlessDataItem)
	}
	return &trustlessItems, nil
}
