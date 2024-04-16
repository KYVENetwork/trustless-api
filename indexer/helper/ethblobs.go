package helper

import (
	"encoding/json"
	"fmt"

	"github.com/KYVENetwork/trustless-rpc/types"
)

type EthBlobIndexer struct{}

func (eth *EthBlobIndexer) GetIndexCount() int {
	return 2
}

func (eth *EthBlobIndexer) GetDataItemIndicies(dataitem *types.TrustlessDataItem) ([]string, error) {
	// Create a struct to unmarshal into
	var blobData types.BlobValue

	// Unmarshal the RawMessage into the struct
	err := json.Unmarshal(dataitem.Value.Value, &blobData)
	if err != nil {
		return nil, err
	}

	var indicies []string = []string{
		dataitem.Value.Key,
		fmt.Sprintf("%v", blobData.SlotNumber),
	}

	return indicies, nil
}
