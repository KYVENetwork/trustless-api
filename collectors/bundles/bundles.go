package bundles

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/KYVENetwork/trustless-api/config"
	"github.com/KYVENetwork/trustless-api/types"
	"github.com/KYVENetwork/trustless-api/utils"
)

func GetFinalizedBundle(chainId string, poolId int64, bundleId int64) (*types.FinalizedBundle, error) {
	restEndpoint := config.Endpoints.Chains[chainId]

	var raw []byte
	var err error
	for _, r := range restEndpoint {
		raw, err = utils.GetFromUrlWithBackoff(fmt.Sprintf(
			"%s/kyve/v1/bundles/%d/%d",
			r,
			poolId,
			bundleId,
		))
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}

	var finalizedBundle types.FinalizedBundle

	if err := json.Unmarshal(raw, &finalizedBundle); err != nil {
		return nil, err
	}

	return &finalizedBundle, nil
}

func GetDataFromFinalizedBundle(bundle types.FinalizedBundle) ([]byte, error) {
	// retrieve bundle from storage provider
	data, err := RetrieveDataFromStorageProvider(bundle)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve data from storage provider with storage id %s: %w", bundle.StorageId, err)
	}

	// validate bundle with sha256 checksum
	if utils.CreateSha256Checksum(data) != bundle.DataHash {
		return nil, fmt.Errorf("found different sha256 checksum on bundle with storage id %s: expected = %s found = %s", bundle.StorageId, utils.CreateSha256Checksum(data), bundle.DataHash)
	}

	// decompress bundle
	deflated, err := DecompressBundleFromStorageProvider(bundle, data)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress bundle: %w", err)
	}

	return deflated, nil
}

func RetrieveDataFromStorageProvider(bundle types.FinalizedBundle) ([]byte, error) {
	id, err := strconv.ParseUint(bundle.StorageProviderId, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse uint from storage provider id: %w", err)
	}

	storageRest, ok := config.Endpoints.Storage[int(id)]
	if !ok {
		return nil, fmt.Errorf("bundle has an invalid storage provider id %s", bundle.StorageProviderId)
	}
	for _, s := range storageRest {
		bytes, err := utils.GetFromUrlWithBackoff(fmt.Sprintf("%v/%s", s, bundle.StorageId))
		if err == nil {
			return bytes, nil
		}
	}
	return nil, fmt.Errorf("failed to fetch bundle: %v", bundle.Id)
}

func DecompressBundleFromStorageProvider(bundle types.FinalizedBundle, data []byte) ([]byte, error) {
	id, err := strconv.ParseUint(bundle.CompressionId, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse uint from compression id: %w", err)
	}

	switch id {
	case 1:
		return utils.DecompressGzip(data)
	default:
		return nil, fmt.Errorf("bundle has an invalid compression id %s. canceling sync", bundle.CompressionId)
	}
}
