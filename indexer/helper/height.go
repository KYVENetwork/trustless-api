package helper

import (
	"encoding/json"

	"github.com/KYVENetwork/trustless-api/utils"

	"github.com/KYVENetwork/trustless-api/merkle"
	"github.com/KYVENetwork/trustless-api/types"
)

type HeightIndexer struct{}

func (eth *HeightIndexer) GetBindings() map[string][]types.ParameterIndex {
	return map[string][]types.ParameterIndex{
		"/value": {
			{
				IndexId:     utils.IndexBlockHeight,
				Parameter:   []string{"height"},
				Description: []string{"blockheight"},
			},
		},
	}
}

// TODO: Handle proofAttached = false
func (*HeightIndexer) IndexBundle(bundle *types.Bundle, _ bool) (*[]types.TrustlessDataItem, error) {
	leafs := merkle.GetBundleHashes(&bundle.DataItems)
	var trustlessItems []types.TrustlessDataItem
	for index, dataitem := range bundle.DataItems {
		proof, err := merkle.GetHashesCompact(leafs, index)
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
			Indices: []types.Index{
				{Index: dataitem.Key, IndexId: utils.IndexBlockHeight},
			},
		}
		trustlessItems = append(trustlessItems, trustlessDataItem)
	}
	return &trustlessItems, nil
}
