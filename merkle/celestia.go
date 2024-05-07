package merkle

import (
	"crypto/sha256"
	"encoding/json"
	"github.com/KYVENetwork/trustless-api/types"
	"github.com/KYVENetwork/trustless-api/utils"
)

// celestiaDataItemToSha256 computes the hash of a Celestia data item.
// Therefore, all namespacedShares are hashed in order to generate a Merkle root of it.
// Combined with the data item key, the root is used to compute the hash.
func celestiaDataItemToSha256(dataItem types.DataItem) [32]byte {
	var celestiaValue types.CelestiaValue

	if err := json.Unmarshal(dataItem.Value, &celestiaValue); err != nil {
		logger.Error().Str("err", err.Error()).Msg("Failed to unmarshal Celestia data item value")
		// TODO: Improve error handling
		panic(err)
	}

	var shareHashes [][32]byte
	for _, namespacedShares := range celestiaValue.SharesByNamespace {
		shareHashes = append(shareHashes, utils.CalculateSHA256Hash(namespacedShares))
	}

	merkleRoot := GetMerkleRoot(shareHashes)

	keyBytes := sha256.Sum256([]byte(dataItem.Key))

	combined := append(keyBytes[:], merkleRoot[:]...)

	return sha256.Sum256(combined)
}

func GetCelestiaDataItemHashesCompact() ([]types.MerkleNode, error) {
	// TODO
}
