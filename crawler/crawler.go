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

func (crawler *Crawler) insertBundleDataItems(bundleId int64) {
	compressedBundle, err := bundles.GetFinalizedBundle(crawler.restEndpoint, crawler.poolId, bundleId)
	if err != nil {
		logger.Fatal().Msg("Something went wrong when retrieving the bundle...")
		return
	}

	bundle, err := bundles.GetDecompressedBundle(*compressedBundle, crawler.storageRest)

	if err != nil {
		logger.Fatal().Msg("Something went wrong when retrieving the bundle...")
		return
	}

	leafs := merkle.GetBundleHashes(&bundle)

	for _, dataitem := range bundle {
		proof := merkle.GetHashesCompact(leafs, dataitem)
		trustlessDataItem := types.TrustlessDataItem{Value: dataitem, Proof: proof, BundleId: bundleId, PoolId: crawler.poolId}
		// TODO: save the trustless data item
		crawler.adapter.Save(trustlessDataItem)
	}
}

func (crawler *Crawler) Start() {
	poolInfo, err := pool.GetPoolInfo(crawler.restEndpoint, crawler.poolId)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to get latest bundle")
		return
	}

	lastBundle := poolInfo.Pool.Data.TotalBundles

	for i := lastBundle - 1; i >= 0; i-- {
		logger.Info().Msg(fmt.Sprintf("Inserting data items: %v/%v", lastBundle-i, lastBundle))
		if crawler.adapter.Exists(i) {
			logger.Info().Int64("bundleId", i).Msg("Bundle already exists, skipping...")
			continue
		}
		crawler.insertBundleDataItems(i)
	}
}

func Create(restEndpoint string, storageRest string, adapter db.Adapter, poolId int64) Crawler {
	return Crawler{restEndpoint: restEndpoint, storageRest: storageRest, adapter: adapter, poolId: poolId}
}
