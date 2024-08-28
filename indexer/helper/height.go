package helper

import (
	"encoding/json"

	"github.com/KYVENetwork/trustless-api/utils"

	"github.com/KYVENetwork/trustless-api/merkle"
	"github.com/KYVENetwork/trustless-api/types"
)

type HeightIndexer struct{}

func (eth *HeightIndexer) GetBindings() map[string]types.Endpoint {
	return map[string]types.Endpoint{
		"/value": {
			QueryParameter: []types.ParameterIndex{
				{
					IndexId:     utils.IndexBlockHeight,
					Parameter:   []string{"height"},
					Description: []string{"height"},
				},
			},
			Schema: "DataItem",
		},
	}
}

// TODO: Handle proofAttached = false
func (*HeightIndexer) IndexBundle(bundle *types.Bundle, proofAttached bool) (*[]types.TrustlessDataItem, error) {
	leafs := merkle.GetBundleHashes(&bundle.DataItems)
	var trustlessItems []types.TrustlessDataItem
	for index, dataitem := range bundle.DataItems {
		proof, err := merkle.GetHashesCompact(leafs, index)
		if err != nil {
			return nil, err
		}
		raw, err := json.Marshal(dataitem)
		if err != nil {
			return nil, err
		}

		var encodedProof string
		if proofAttached {
			encodedProof = utils.EncodeProof(bundle.PoolId, bundle.BundleId, bundle.ChainId, dataitem.Key, "value", proof)
		} else {
			encodedProof = ""
		}

		trustlessDataItem := types.TrustlessDataItem{
			Value:    raw,
			Proof:    encodedProof,
			BundleId: bundle.BundleId,
			PoolId:   bundle.PoolId,
			Indices: []types.Index{
				{Index: dataitem.Key, IndexId: utils.IndexBlockHeight},
			},
		}
		trustlessItems = append(trustlessItems, trustlessDataItem)
	}
	return &trustlessItems, nil
}
