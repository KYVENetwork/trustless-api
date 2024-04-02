package merkle

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/KYVENetwork/trustless-rpc/collectors/bundles"
	"github.com/KYVENetwork/trustless-rpc/types"
	"github.com/KYVENetwork/trustless-rpc/utils"
)

var (
	logger = utils.TrustlessRpcLogger("merkle")
)

func GetMerkleRoot(hashes [][32]byte) [32]byte {
	if len(hashes) == 1 {
		return hashes[0]
	}
	var computedHashes = [][32]byte{}

	for i := 0; i < len(hashes); i += 2 {
		left := hashes[i]
		if i+1 == len(hashes) {
			combined := append(left[:], left[:]...)
			parentHash := sha256.Sum256(combined)
			computedHashes = append(computedHashes, parentHash)
			continue
		}
		right := hashes[i+1]
		combined := append(left[:], right[:]...)
		parentHash := sha256.Sum256(combined)
		computedHashes = append(computedHashes, parentHash)
	}

	return GetMerkleRoot(computedHashes)
}

func IsBundleValid(bundleId int64, poolId int64, restEndpoint string, storageRest string) bool {
	compressedBundle, err := bundles.GetFinalizedBundle(restEndpoint, poolId, bundleId)
	if err != nil {
		fmt.Println(err)
		return false
	}

	decompressedBundle, err :=
		bundles.GetDataFromFinalizedBundle(*compressedBundle, storageRest)
	if err != nil {
		logger.Fatal().Msg(fmt.Sprintf("failed to decompress bundle: %v\n", err.Error()))
		return false
	}

	// parse bundle
	var bundle types.Bundle

	if err := json.Unmarshal(decompressedBundle, &bundle); err != nil {
		logger.Fatal().Msg(fmt.Sprintf("failed to unmarshal bundle data: %v\n", err.Error()))
		return false
	}
	var hashes [][32]byte

	for _, dataitem := range bundle {
		hashes = append(hashes, utils.CalculateSHA256Hash(dataitem))
	}

	var summary types.BundleSummary

	if err := json.Unmarshal([]byte(compressedBundle.BundleSummary), &summary); err != nil {
		logger.Fatal().Msg(fmt.Sprintf("failed to unmarshal bundle summary: %v\n", err.Error()))
		return false
	}

	rootHash := GetMerkleRoot(hashes)
	hexHash := hex.EncodeToString(rootHash[:])
	if hexHash != summary.MerkleRoot {
		logger.Fatal().Str("expected", summary.MerkleRoot).Str("got", hexHash).Msg("bundle is not valid: bundle summary hash is not equal to calculated hash")
		return false
	}
	logger.Info().Msg("Bundle valid!")
	return false
}
