package crawler

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/KYVENetwork/trustless-api/collectors/bundles"
	"github.com/KYVENetwork/trustless-api/collectors/pool"
	"github.com/KYVENetwork/trustless-api/config"
	"github.com/KYVENetwork/trustless-api/db"
	"github.com/KYVENetwork/trustless-api/merkle"
	"github.com/KYVENetwork/trustless-api/types"
	"github.com/KYVENetwork/trustless-api/utils"
	"github.com/go-co-op/gocron"
	"github.com/spf13/viper"
)

var (
	logger = utils.TrustlessApiLogger("crawler")
)

type Crawler struct {
	bundleCrawler []*BundleCrawler
}

type BundleCrawler struct {
	restEndpoint string
	storageRest  string
	adapter      db.Adapter
	poolId       int64
	crawling     sync.Mutex
}

func (crawler *BundleCrawler) insertBundleDataItems(bundleId int64) error {
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

func (crawler *BundleCrawler) crawlBundles() {
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
			logger.Error().Err(err).Msg("Failed to insert bundle data items, retrying...")
			i--
		}
	}

	logger.Info().Int64("bundleId", lastBundle).Msg("Finished crawling to bundle.")
}

func (crawler *BundleCrawler) Start() {
	scheduler := gocron.NewScheduler(time.UTC)
	scheduler.Every(3).Minutes().Do(crawler.crawlBundles)
	scheduler.StartBlocking()
}

func CreateBundleCrawler(restEndpoint string, storageRest string, adapter db.Adapter, poolId int64) BundleCrawler {
	return BundleCrawler{restEndpoint: restEndpoint, storageRest: storageRest, adapter: adapter, poolId: poolId}
}

func Create() Crawler {
	var bundleCrawler []*BundleCrawler
	chainId := viper.GetString("chain-id")
	endpoint := utils.GetChainRest(chainId, viper.GetString("chain-rest"))
	storageRest := strings.TrimSuffix(viper.GetString("storage-rest"), "/")
	for _, bc := range config.GetCrawlerConfig() {
		adapter := bc.GetDatabaseAdapter()
		newCrawler := CreateBundleCrawler(endpoint, storageRest, adapter, bc.PoolId)
		bundleCrawler = append(bundleCrawler, &newCrawler)
	}

	return Crawler{
		bundleCrawler: bundleCrawler,
	}
}

func (c *Crawler) Start() {
	for _, bc := range c.bundleCrawler {
		bc.Start()
	}
}
