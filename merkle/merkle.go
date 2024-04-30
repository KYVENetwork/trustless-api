package merkle

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/KYVENetwork/trustless-api/collectors/bundles"
	"github.com/KYVENetwork/trustless-api/types"
	"github.com/KYVENetwork/trustless-api/utils"
)

var (
	logger = utils.TrustlessApiLogger("merkle")
)

func buildMerkleTree(hashes *[][32]byte, tree *[][]string) {
	if len(*hashes) == 1 {
		return
	}

	// make sure we have an even number of hashes
	if len(*hashes)%2 == 1 {
		*hashes = append(*hashes, (*hashes)[len(*hashes)-1])
	}

	hexHashes := utils.BytesToHex(hashes)
	*tree = append(*tree, hexHashes)

	var computedHashes = [][32]byte{}

	for i := 0; i < len(*hashes); i += 2 {
		left := (*hashes)[i]
		right := (*hashes)[i+1]
		combined := append(left[:], right[:]...)
		parentHash := sha256.Sum256(combined)
		computedHashes = append(computedHashes, parentHash)
	}

	buildMerkleTree(&computedHashes, tree)
}

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

func GetBundleHashes(bundle *types.Bundle) *[][32]byte {
	var hashes [][32]byte
	for _, dataitem := range *bundle {
		hashes = append(hashes, utils.CalculateSHA256Hash(dataitem))
	}
	return &hashes
}

func GetBundleHashesHex(bundle *types.Bundle) []string {
	hashes := GetBundleHashes(bundle)
	return utils.BytesToHex(hashes)
}

func GetHashesCompact(hashes *[][32]byte, leafObj *types.DataItem) []types.MerkleNode {
	var tree [][]string
	buildMerkleTree(hashes, &tree)

	leafHash := utils.CalculateSHA256Hash(*leafObj)
	leaf := hex.EncodeToString(leafHash[:])

	// first find the leaf index
	var leafIndex int = -1
	for index, currentLeaf := range tree[0] {
		if leaf == currentLeaf {
			leafIndex = index
			break
		}
	}

	if leafIndex == -1 {
		// was not able to find leaf in merkle tree
		return []types.MerkleNode{}
	}

	var compactHashes []types.MerkleNode
	var level = 0 // we start at level 0
	var currentIndex = leafIndex

	for level < len(tree) {
		// even means the leaf is on the left side
		if currentIndex%2 == 0 {
			node := types.MerkleNode{Left: true, Hash: tree[level][currentIndex+1]}
			compactHashes = append(compactHashes, node)
			currentIndex /= 2
			level++
		} else {
			node := types.MerkleNode{Left: false, Hash: tree[level][currentIndex-1]}
			compactHashes = append(compactHashes, node)
			currentIndex /= 2
			level++
		}
	}

	return compactHashes
}

func IsBundleValid(bundleId int64, poolId int64, chainId string) bool {
	compressedBundle, err := bundles.GetFinalizedBundle(chainId, poolId, bundleId)
	if err != nil {
		fmt.Println(err)
		return false
	}

	decompressedBundle, err :=
		bundles.GetDataFromFinalizedBundle(*compressedBundle)
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

	var summary types.BundleSummary

	if err := json.Unmarshal([]byte(compressedBundle.BundleSummary), &summary); err != nil {
		logger.Fatal().Msg(fmt.Sprintf("failed to unmarshal bundle summary: %v\n", err.Error()))
		return false
	}

	hashes := GetBundleHashes(&bundle)
	rootHash := GetMerkleRoot(*hashes)
	hexHash := hex.EncodeToString(rootHash[:])
	if hexHash != summary.MerkleRoot {
		logger.Fatal().Str("expected", summary.MerkleRoot).Str("got", hexHash).Msg("bundle is not valid: bundle summary hash is not equal to calculated hash")
		return false
	}
	logger.Info().Str("hash", summary.MerkleRoot).Msg("Bundle valid!")

	return false
}
