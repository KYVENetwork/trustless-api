package helper

import (
	"encoding/json"
	"strconv"

	"github.com/KYVENetwork/trustless-api/types"
)

type EthBlobsIndexer struct{}

const (
	IndexBlockHeight = 0
	IndexSlotNumber  = 1
)

func (eth *EthBlobsIndexer) GetBindings() map[string]map[string]int64 {
	return map[string]map[string]int64{
		"/beacon/blob_sidecars": {
			"block_height": IndexBlockHeight,
			"slot_number":  IndexSlotNumber,
		},
	}
}

func (eth *EthBlobsIndexer) GetDataItemIndices(dataitem *types.TrustlessDataItem) ([]int64, error) {
	// Create a struct to unmarshal into
	var blobData types.BlobValue

	// Unmarshal the RawMessage into the struct
	err := json.Unmarshal(dataitem.Value.Value, &blobData)
	if err != nil {
		return nil, err
	}
	height, err := strconv.Atoi(dataitem.Value.Key)
	if err != nil {
		return nil, err
	}
	var indices []int64 = []int64{
		int64(height),
		int64(blobData.SlotNumber),
	}

	return indices, nil
}
