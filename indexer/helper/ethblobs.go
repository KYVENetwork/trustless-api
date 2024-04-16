package helper

import (
	"encoding/json"
	"strconv"

	"github.com/KYVENetwork/trustless-rpc/types"
)

type EthBlobIndexer struct{}

func (eth *EthBlobIndexer) GetIndexCount() int {
	return 2
}

func (eth *EthBlobIndexer) GetDataItemIndicies(dataitem *types.TrustlessDataItem) ([]int, error) {
	// Create a struct to unmarshal into
	var blobData types.BlobValue

	// Unmarshal the RawMessage into the struct
	err := json.Unmarshal(dataitem.Value.Value, &blobData)
	if err != nil {
		return nil, err
	}
	height, _ := strconv.Atoi(dataitem.Value.Key)
	var indicies []int = []int{
		height,
		height,
	}

	return indicies, nil
}
