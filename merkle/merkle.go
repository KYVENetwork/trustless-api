package merkle

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/KYVENetwork/trustless-api/types"
	"github.com/KYVENetwork/trustless-api/utils"
)

// builds a merkle tree based of the hashes
// each level will be inserted in `tree`,
// where the first item are the leafs and the last element are the two leafs that make the root
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
//
// the `leafHash` has to be included in the hashes array
func GetHashesCompact(hashes *[][32]byte, leafHash *[32]byte) ([]types.MerkleNode, error) {
	var tree [][]string
	buildMerkleTree(hashes, &tree)

	leaf := hex.EncodeToString(leafHash[:])

	if len(tree) == 0 {
		// was not able to find leaf in merkle tree
		return []types.MerkleNode{}, fmt.Errorf("failed to create tree")
	}

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
		return []types.MerkleNode{}, fmt.Errorf("failed to find leaf in hashes")
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
