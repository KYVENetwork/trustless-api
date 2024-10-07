package merkle

import (
	"crypto/sha256"
	"fmt"

	"github.com/KYVENetwork/trustless-api/types"
	"github.com/KYVENetwork/trustless-api/utils"
)

// builds a merkle tree based of the hashes
// each level will be inserted in `tree`,
// where the first item are the leafs and the last element are the two leafs that make the root
func buildMerkleTree(hashes *[][32]byte, tree *[][]string) {

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

	if len(computedHashes) == 1 {
		return
	}

	buildMerkleTree(&computedHashes, tree)
}

func GetMerkleRoot(hashes [][32]byte) [32]byte {
	if len(hashes) == 0 {
		return [32]byte{}
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

	if len(computedHashes) == 1 {
		return computedHashes[0]
	}
	return GetMerkleRoot(computedHashes)
}

func GetBundleHashes(bundle *[]types.DataItem) *[][32]byte {
	var hashes [][32]byte
	for _, dataitem := range *bundle {
		hashes = append(hashes, utils.CalculateSHA256Hash(dataitem))
	}
	return &hashes
}

func GetBundleHashesHex(bundle *[]types.DataItem) []string {
	hashes := GetBundleHashes(bundle)
	return utils.BytesToHex(hashes)
}

// GetHashesCompact creates a compact merkle tree for the given leaf
// this function will construct a merkle tree based on the hashes and
// construct only the necessary hashes for building the merkle tree
func GetHashesCompact(hashes *[][32]byte, leafIndex int) ([]types.MerkleNode, error) {
	var tree [][]string
	buildMerkleTree(hashes, &tree)
	length := len(tree)
	if length == 0 {
		// failed to construct merkle tree
		return []types.MerkleNode{}, fmt.Errorf("failed to create tree")
	}
	if leafIndex < 0 || leafIndex >= len(*hashes) {
		// leafIndex not within the hashes
		return []types.MerkleNode{}, fmt.Errorf("leafIndex out of bounds")
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

	return compactHashes, nil
}
