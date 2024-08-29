package helper

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/KYVENetwork/trustless-api/merkle"
	"github.com/KYVENetwork/trustless-api/types"
	"github.com/KYVENetwork/trustless-api/utils"
)

type CelestiaIndexer struct{}

func (*CelestiaIndexer) GetBindings() map[string]types.Endpoint {
	return map[string]types.Endpoint{
		"/GetSharesByNamespace": {
			QueryParameter: []types.ParameterIndex{
				{
					IndexId:     utils.IndexSharesByNamespace,
					Parameter:   []string{"height", "namespace"},
					Description: []string{"celestia block height", "celestia namespace, available namespaces: AAAAAAAAAAAAAAAAAAAAAAAAAIZiad33fbxA7Z0=,AAAAAAAAAAAAAAAAAAAAAAAAAAAACAgICAgICAg=,AAAAAAAAAAAAAAAAAAAAAAAAAAAABYTLU4hLOUU=,AAAAAAAAAAAAAAAAAAAAAAAAAAAADBuw7+PjGs8="},
				},
			},
			Schema: "DataItem",
		},
	}
}

func (c *CelestiaIndexer) IndexBundle(bundle *types.Bundle, excludeProof bool) (*[]types.TrustlessDataItem, error) {

	// convert data items to celestia data items
	// we can also construct the high level leafs at this point
	var dataItems []types.CelestiaDataItem
	var leafs [][32]byte

	for _, item := range bundle.DataItems {
		var celestiaValue types.CelestiaValue

		if err := json.Unmarshal(item.Value, &celestiaValue); err != nil {
			return nil, err
		}
		celestiaItem := types.CelestiaDataItem{Key: item.Key, Value: celestiaValue}
		leafs = append(leafs, c.celestiaDataItemToSha256(&celestiaItem))
		dataItems = append(dataItems, celestiaItem)
	}

	var trustlessItems []types.TrustlessDataItem

	// now we can process all the data items inside of the bundle
	// we want to create an index for each data item
	// but we also want to create an index for each namespace of each data item
	for index, dataitem := range dataItems {
		// this will be the roof of our proof
		proof, err := merkle.GetHashesCompact(&leafs, index)
		if err != nil {
			return nil, err
		}

		// first we have to construct the leafs of all the namespaces
		var namespaceLeafs [][32]byte
		for _, namespacedShares := range dataitem.Value.SharesByNamespace {
			namespaceLeafs = append(namespaceLeafs, utils.CalculateSHA256Hash(namespacedShares))
		}

		for index, namespace := range dataitem.Value.SharesByNamespace {
			namespaceProof, err := merkle.GetHashesCompact(&namespaceLeafs, index)
			if err != nil {
				return nil, err
			}

			// Because we also hash the key of the original data item, we have to append an extra leaf with the key
			keyBytes := sha256.Sum256([]byte(dataitem.Key))
			keyHash := hex.EncodeToString(keyBytes[:])
			totalProof := append(namespaceProof, types.MerkleNode{Left: false, Hash: keyHash})

			// finally append the proof for the rest of the data items
			totalProof = append(totalProof, proof...)

			rpcResponse, err := utils.WrapIntoJsonRpcResponse(dataitem.Value.SharesByNamespace[index])
			if err != nil {
				return nil, err
			}

			index := fmt.Sprintf("%v-%v", dataitem.Key, namespace.NamespaceId)

			var encodedProof string
			// if proof is not attached, we set the proof to an empty string
			if excludeProof {
				encodedProof = ""
			} else {
				encodedProof = utils.EncodeProof(bundle.PoolId, bundle.BundleId, bundle.ChainId, "", "result", totalProof)
			}

			trustlessDataItem := types.TrustlessDataItem{
				Value:    rpcResponse,
				Proof:    encodedProof,
				BundleId: bundle.BundleId,
				PoolId:   bundle.PoolId,
				Indices: []types.Index{
					{Index: index, IndexId: utils.IndexSharesByNamespace},
				},
			}
			trustlessItems = append(trustlessItems, trustlessDataItem)
		}
	}

	return &trustlessItems, nil
}

func (*CelestiaIndexer) celestiaDataItemToSha256(dataItem *types.CelestiaDataItem) [32]byte {

	var shareHashes [][32]byte
	for _, namespacedShares := range dataItem.Value.SharesByNamespace {
		shareHashes = append(shareHashes, utils.CalculateSHA256Hash(namespacedShares))
	}

	merkleRoot := merkle.GetMerkleRoot(shareHashes)
	keyBytes := sha256.Sum256([]byte(dataItem.Key))
	combined := append(keyBytes[:], merkleRoot[:]...)

	return sha256.Sum256(combined)
}
