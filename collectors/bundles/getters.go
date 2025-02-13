package bundles

import (
	jsoniter "github.com/json-iterator/go"

	"github.com/KYVENetwork/trustless-api/types"
	"github.com/KYVENetwork/trustless-api/utils"
)

func GetDecompressedBundle(finalizedBundle types.FinalizedBundle, labels []string) ([]types.DataItem, error) {

	decompressedBundle, err := GetDataFromFinalizedBundle(finalizedBundle)
	if err != nil {
		return nil, err
	}

	utils.PrometheusBundleSize.WithLabelValues(labels...).Set(float64(len(decompressedBundle)))

	var bundle []types.DataItem
	if err := jsoniter.Unmarshal(decompressedBundle, &bundle); err != nil {
		return nil, err
	}

	return bundle, nil
}
