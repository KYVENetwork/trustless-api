package crawler

import (
	"fmt"
	"sync"
	"time"

	"github.com/KYVENetwork/trustless-rpc/collectors/bundles"
	"github.com/KYVENetwork/trustless-rpc/collectors/pool"
	"github.com/KYVENetwork/trustless-rpc/db"
	"github.com/KYVENetwork/trustless-rpc/merkle"
	"github.com/KYVENetwork/trustless-rpc/types"
	"github.com/KYVENetwork/trustless-rpc/utils"
	"github.com/go-co-op/gocron"
)

var (
	logger = utils.TrustlessRpcLogger("crawler")
)

type Crawler struct {
	restEndpoint string
	storageRest  string
	adapter      db.Adapter
	poolId       int64
	crawling     sync.Mutex
}

func (crawler *Crawler) insertBundleDataItems(bundleId int64) error {
	start := time.Now()

	compressedBundle, err := bundles.GetFinalizedBundle(crawler.restEndpoint, crawler.poolId, bundleId)
	if err != nil {
		logger.Error().Msg("Something went wrong when retrieving the bundle...")
		return err
	}

	bundle, err := bundles.GetDecompressedBundle(*compressedBundle, crawler.storageRest)

	if err != nil {
		logger.Error().Msg("Something went wrong when retrieving the bundle...")
		return err
	}

	elapsed := time.Since(start)
	logger.Debug().Msg(fmt.Sprintf("Downloading bundle took: %v", elapsed))

	leafs := merkle.GetBundleHashes(&bundle)
	if false {
		fmt.Println(leafs)
	}

	start = time.Now()
	var trustlessDataItems []types.TrustlessDataItem
	for _, dataitem := range bundle {
		proof := merkle.GetHashesCompact(leafs, &dataitem)
		trustlessDataItem := types.TrustlessDataItem{Value: dataitem, Proof: proof, BundleId: bundleId, PoolId: crawler.poolId}
		trustlessDataItems = append(trustlessDataItems, trustlessDataItem)
	}
	err = crawler.adapter.Save(&trustlessDataItems)
	if err != nil {
		return err
	}
	elapsed = time.Since(start)
	logger.Debug().Msg(fmt.Sprintf("Inserting data items took: %v", elapsed))

	return nil
}

func (crawler *Crawler) crawlBundles() {
	if !crawler.crawling.TryLock() {
		logger.Info().Msg("Still crawling bundles!")
		return
	}

	defer crawler.crawling.Unlock()

	poolInfo, err := pool.GetPoolInfo(crawler.restEndpoint, crawler.poolId)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get latest bundle")
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

	logger.Info().Int64("bundleId", lastBundle).Msg("Finished crawling to bundle.")
}

func (crawler *Crawler) Start() {
	scheduler := gocron.NewScheduler(time.UTC)
	scheduler.Every(3).Minutes().Do(crawler.crawlBundles)
	scheduler.StartBlocking()
}

func Create(restEndpoint string, storageRest string, adapter db.Adapter, poolId int64) Crawler {
	return Crawler{restEndpoint: restEndpoint, storageRest: storageRest, adapter: adapter, poolId: poolId}
}
