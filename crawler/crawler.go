package crawler

import (
	"fmt"

	"github.com/KYVENetwork/trustless-rpc/collectors/bundles"
	"github.com/KYVENetwork/trustless-rpc/collectors/pool"
	"github.com/KYVENetwork/trustless-rpc/db"
	"github.com/KYVENetwork/trustless-rpc/merkle"
	"github.com/KYVENetwork/trustless-rpc/types"
	"github.com/KYVENetwork/trustless-rpc/utils"
)

var (
	logger = utils.TrustlessRpcLogger("crawler")
)

type Crawler struct {
	restEndpoint string
	storageRest  string
	adapter      db.Adapter
	poolId       int64
}

func (crawler *Crawler) insertBundleDataItems(bundleId int64) error {
	compressedBundle, err := bundles.GetFinalizedBundle(crawler.restEndpoint, crawler.poolId, bundleId)
	if err != nil {
		logger.Fatal().Msg("Something went wrong when retrieving the bundle...")
		return err
	}

	bundle, err := bundles.GetDecompressedBundle(*compressedBundle, crawler.storageRest)

	if err != nil {
		logger.Fatal().Msg("Something went wrong when retrieving the bundle...")
		return err
	}

	leafs := merkle.GetBundleHashes(&bundle)

	for _, dataitem := range bundle {
		proof := merkle.GetHashesCompact(leafs, dataitem)
		trustlessDataItem := types.TrustlessDataItem{Value: dataitem, Proof: proof, BundleId: bundleId, PoolId: crawler.poolId}
		crawler.adapter.Save(trustlessDataItem)
	}

	return nil
}

func (crawler *Crawler) Start() {
	poolInfo, err := pool.GetPoolInfo(crawler.restEndpoint, crawler.poolId)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to get latest bundle")
		return
	}

	lastBundle := poolInfo.Pool.Data.TotalBundles

	for i := int64(0); i < lastBundle; i++ {
		logger.Info().Msg(fmt.Sprintf("Inserting data items: %v/%v", i, lastBundle))
		if crawler.adapter.Exists(i) {
			logger.Info().Int64("bundleId", i).Msg("Bundle already exists, skipping...")
			continue
		}

		err := crawler.insertBundleDataItems(i)
		if err != nil {
			i--
		}
	}
}

func Create(restEndpoint string, storageRest string, adapter db.Adapter, poolId int64) Crawler {
	return Crawler{restEndpoint: restEndpoint, storageRest: storageRest, adapter: adapter, poolId: poolId}
}
