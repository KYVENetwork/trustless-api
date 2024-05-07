package helper

import (
	"encoding/json"

	"github.com/KYVENetwork/trustless-api/merkle"
	"github.com/KYVENetwork/trustless-api/types"
	"github.com/KYVENetwork/trustless-api/utils"
)

type HeightIndexer struct{}

func (eth *HeightIndexer) GetBindings() map[string]map[string]int64 {
	return map[string]map[string]int64{
		"/value": {
			"block_height": IndexBlockHeight,
		},
	}
}

func (*HeightIndexer) IndexBundle(bundle *types.Bundle) (*[]types.TrustlessDataItem, error) {
	leafs := merkle.GetBundleHashes(&bundle.DataItems)
	var trustlessItems []types.TrustlessDataItem
	for index, dataitem := range bundle.DataItems {
		leafHash := utils.CalculateSHA256Hash(dataitem)
		proof, err := merkle.GetHashesCompact(leafs, &leafHash)
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
			Keys:     []string{dataitem.Key},
		}
		trustlessItems = append(trustlessItems, trustlessDataItem)
	}
	return &trustlessItems, nil
}
