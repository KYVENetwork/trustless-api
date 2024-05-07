package helper

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/KYVENetwork/trustless-api/merkle"
	"github.com/KYVENetwork/trustless-api/types"
	"github.com/KYVENetwork/trustless-api/utils"
)

type CelestiaIndexer struct{}

const (
	IndexSharesByNamespace = 3
)

func (*CelestiaIndexer) GetBindings() map[string]map[string]int64 {
	return map[string]map[string]int64{
		"/GetSharesByNamespace": {
			"height":    IndexSharesByNamespace,
			"namespace": IndexSharesByNamespace,
		},
		"/GetShares": {
			"height": IndexBlockHeight,
		},
	}
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

func (c *CelestiaIndexer) IndexBundle(bundle *types.Bundle) (*[]types.TrustlessDataItem, error) {

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
		leafHash := c.celestiaDataItemToSha256(&dataitem)
		proof, err := merkle.GetHashesCompact(&leafs, &leafHash)
		if err != nil {
			return nil, err
		}
		raw, err := json.Marshal(bundle.DataItems[index])
		if err != nil {
			return nil, err
		}

		// first we insert the entire bundle for the block height key
		trustlessDataItem := types.TrustlessDataItem{
			Value:    raw,
			Proof:    proof,
			BundleId: bundle.BundleId,
			PoolId:   bundle.PoolId,
			ChainId:  bundle.ChainId,
			Keys:     []string{dataitem.Key},
			IndexId:  IndexBlockHeight,
		}
		trustlessItems = append(trustlessItems, trustlessDataItem)

		// then we go through every namespace and create another item just for the namespace as the key and the block height

		// first we have to construct the leafs of all the namespaces
		var namespaceLeafs [][32]byte
		for _, namespacedShares := range dataitem.Value.SharesByNamespace {
			namespaceLeafs = append(namespaceLeafs, utils.CalculateSHA256Hash(namespacedShares))
		}

		for index, namespace := range dataitem.Value.SharesByNamespace {
			leafHash = utils.CalculateSHA256Hash(namespace)
			namespaceProof, err := merkle.GetHashesCompact(&namespaceLeafs, &leafHash)
			if err != nil {
				return nil, err
			}

			// NOTE: becase we also hash the key of the original data item, we have to append an extra node of with the key
			keyBytes := sha256.Sum256([]byte(dataitem.Key))
			keyHash := hex.EncodeToString(keyBytes[:])
			totalProof := append(namespaceProof, types.MerkleNode{Left: false, Hash: keyHash})

			// finally append the proof for the rest of the data items
			totalProof = append(totalProof, proof...)

			raw, err := json.Marshal(dataitem.Value.SharesByNamespace[index])
			if err != nil {
				return nil, err
			}

			trustlessDataItem := types.TrustlessDataItem{
				Value:    raw,
				Proof:    totalProof,
				BundleId: bundle.BundleId,
				PoolId:   bundle.PoolId,
				ChainId:  bundle.ChainId,
				Keys:     []string{dataitem.Key, namespace.NamespaceId},
				IndexId:  IndexSharesByNamespace,
			}
			trustlessItems = append(trustlessItems, trustlessDataItem)
		}
	}

	return &trustlessItems, nil
}
