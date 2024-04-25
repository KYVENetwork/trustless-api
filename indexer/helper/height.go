package helper

import (
	"strconv"

	"github.com/KYVENetwork/trustless-api/types"
)

type HeightIndexer struct{}

func (*HeightIndexer) GetDataItemIndices(dataitem *types.TrustlessDataItem) ([]int64, error) {
	height, err := strconv.Atoi(dataitem.Value.Key)
	if err != nil {
		return []int64{}, err
	}
	var indices []int64 = []int64{
		int64(height),
	}
	return indices, nil
}
