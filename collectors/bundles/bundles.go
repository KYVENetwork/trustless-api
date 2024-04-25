package bundles

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/KYVENetwork/trustless-api/collectors/pool"
	"github.com/KYVENetwork/trustless-api/types"
	"github.com/KYVENetwork/trustless-api/utils"
)

func GetBundleByKey(key int, restEndpoint string, poolId int64) (*types.FinalizedBundle, error) {
	poolInfo, err := pool.GetPoolInfo(restEndpoint, poolId)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool info: %w", err)
	}

	// Exit if requested height is smaller than pool start height
	poolStartKey, err := strconv.Atoi(poolInfo.Pool.Data.StartKey)
	if err != nil {
		return nil, fmt.Errorf("failed to convert pool start key: %w", err)
	}
	if poolStartKey > key {
		return nil, fmt.Errorf("requested height is smaller than KYVE pool start height")
	}

	paginationKey := ""
	for {
		finalizedBundles, nextKey, err := GetFinalizedBundlesPage(restEndpoint, poolId, utils.BundlesPageLimit, paginationKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get finalized bundles page: %w", err)
		}

		lastToKey, err := strconv.Atoi(finalizedBundles[len(finalizedBundles)-1].ToKey)
		if err != nil {
			return nil, fmt.Errorf("failed to convert last toKey: %w", err)
		}

		if lastToKey < key {
			// If last found key is smaller than requested key, continue or exit if not validated yet
			if nextKey == "" {
				return nil, fmt.Errorf("could not find requested height")
			}
			paginationKey = nextKey
			continue
		} else {
			// If last found key is greater than or equal to requested key, return bundle containing requested key
		BundleFinder:
			for _, bundle := range finalizedBundles {
				toKey, err := strconv.Atoi(bundle.ToKey)
				if err != nil {
					return nil, fmt.Errorf("failed to convert toKey: %w", err)
				}
				if toKey < key {
					continue BundleFinder
				} else {
					return &bundle, nil
				}
			}
		}
	}
}

func GetBundleBySlot(slot int, restEndpoint string, poolId int64) (*types.FinalizedBundle, error) {
	// TODO: Calculate start_slot or define in registry to exit before searching in all bundles
	if slot < 800000 {
		return nil, fmt.Errorf("requested slot is smaller than KYVE pool start slot")
	}

	paginationKey := ""
	for {
		finalizedBundles, nextKey, err := GetFinalizedBundlesPage(restEndpoint, poolId, utils.BundlesPageLimit, paginationKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get finalized bundles page: %w", err)
		}

		lastBundleSummaryRaw := finalizedBundles[len(finalizedBundles)-1].BundleSummary

		var lastBundleSummary *types.EthereumBlobsBundleSummary
		if err = json.Unmarshal([]byte(lastBundleSummaryRaw), &lastBundleSummary); err != nil {
			return nil, err
		}

		if lastBundleSummary.ToSlot < slot {
			// If last found slot is smaller than requested slot, continue or exit if not validated yet
			if nextKey == "" {
				return nil, fmt.Errorf("could not find requested height")
			}
			paginationKey = nextKey
			continue
		} else {
			// If last found key is greater than or equal to requested key, return bundle containing requested key
		BundleFinder:
			for _, bundle := range finalizedBundles {
				var summary types.EthereumBlobsBundleSummary
				if err = json.Unmarshal([]byte(bundle.BundleSummary), &summary); err != nil {
					return nil, err
				}

				if summary.ToSlot < slot {
					continue BundleFinder
				} else {
					return &bundle, nil
				}
			}
		}
	}
}

func GetFinalizedBundlesPage(restEndpoint string, poolId int64, paginationLimit int64, paginationKey string) ([]types.FinalizedBundle, string, error) {
	raw, err := utils.GetFromUrlWithBackoff(fmt.Sprintf(
		"%s/kyve/v1/bundles/%d?pagination.limit=%d&pagination.key=%s",
		restEndpoint,
		poolId,
		paginationLimit,
		paginationKey,
	))
	if err != nil {
		return nil, "", err
	}

	var bundlesResponse types.FinalizedBundlesResponse

	if err := json.Unmarshal(raw, &bundlesResponse); err != nil {
		return nil, "", err
	}

	nextKey := base64.URLEncoding.EncodeToString(bundlesResponse.Pagination.NextKey)

	return bundlesResponse.FinalizedBundles, nextKey, nil
}

func GetFinalizedBundle(restEndpoint string, poolId int64, bundleId int64) (*types.FinalizedBundle, error) {
	raw, err := utils.GetFromUrlWithBackoff(fmt.Sprintf(
		"%s/kyve/v1/bundles/%d/%d",
		restEndpoint,
		poolId,
		bundleId,
	))
	if err != nil {
		return nil, err
	}

	var finalizedBundle types.FinalizedBundle

	if err := json.Unmarshal(raw, &finalizedBundle); err != nil {
		return nil, err
	}

	return &finalizedBundle, nil
}

func GetDataFromFinalizedBundle(bundle types.FinalizedBundle, storageRest string) ([]byte, error) {
	// retrieve bundle from storage provider
	data, err := RetrieveDataFromStorageProvider(bundle, storageRest)
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

func RetrieveDataFromStorageProvider(bundle types.FinalizedBundle, storageRest string) ([]byte, error) {
	id, err := strconv.ParseUint(bundle.StorageProviderId, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse uint from storage provider id: %w", err)
	}

	if storageRest != "" {
		return utils.GetFromUrlWithBackoff(fmt.Sprintf("%s/%s", storageRest, bundle.StorageId))
	}

	switch id {
	case 1:
		return utils.GetFromUrlWithBackoff(fmt.Sprintf("%v/%s", utils.RestEndpointArweave, bundle.StorageId))
	case 2:
		return utils.GetFromUrlWithBackoff(fmt.Sprintf("%v/%s", utils.RestEndpointBundlr, bundle.StorageId))
	case 3:
		return utils.GetFromUrlWithBackoff(fmt.Sprintf("%v/%s", utils.RestEndpointKYVEStorage, bundle.StorageId))
	default:
		return nil, fmt.Errorf("bundle has an invalid storage provider id %s. canceling sync", bundle.StorageProviderId)
	}
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
