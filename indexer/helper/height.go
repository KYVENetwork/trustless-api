package helper

import (
	"strconv"

	"github.com/KYVENetwork/trustless-rpc/types"
)

type HeightIndexer struct{}

func (*HeightIndexer) GetDataItemIndicies(dataitem *types.TrustlessDataItem) ([]int64, error) {
	height, err := strconv.Atoi(dataitem.Value.Key)
	if err != nil {
		return []int64{}, err
	}
	var indicies []int64 = []int64{
		int64(height),
	}
	return indicies, nil
}
