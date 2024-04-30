package bundles

import (
	"encoding/json"

	"github.com/KYVENetwork/trustless-api/types"
)

func GetDecompressedBundle(finalizedBundle types.FinalizedBundle) (types.Bundle, error) {

	decompressedBundle, err := GetDataFromFinalizedBundle(finalizedBundle)
	if err != nil {
		return nil, err
	}

	var bundle types.Bundle
	if err := json.Unmarshal(decompressedBundle, &bundle); err != nil {
		return nil, err
	}

	return bundle, nil
}
