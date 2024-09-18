package bundles

import (
	"encoding/json"

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
	if err := json.Unmarshal(decompressedBundle, &bundle); err != nil {
		return nil, err
	}

	return bundle, nil
}
