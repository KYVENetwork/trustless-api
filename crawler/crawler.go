package crawler

import (
	"fmt"
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
)

var (
	logger = utils.TrustlessApiLogger("crawler")
)

type Crawler struct {
	children []*BundleCrawler
}

type BundleCrawler struct {
	restEndpoint string
	storageRest  string
	adapter      db.Adapter
	poolId       int64
	crawling     sync.Mutex
	buffer       chan int64
	err          chan error
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
		fmt.Println(bundleId)
		return err
	}
	elapsed = time.Since(start)
	logger.Debug().Msg(fmt.Sprintf("Inserting data items took: %v", elapsed))

	return nil
}

func (crawler *BundleCrawler) bundleWorker() {
	for {
		bundleId := <-crawler.buffer
		err := crawler.insertBundleDataItems(bundleId)
		if err != nil {
			crawler.err <- err
		}
	}
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
		if crawler.adapter.Exists(i) {
			continue
		}
		logger.Info().Msg(fmt.Sprintf("Inserting data items: %v/%v", i, lastBundle))
		crawler.buffer <- i
		select {
		case <-crawler.err:
			logger.Error().Err(err).Msg("Failed to process bundle...")
			return
		default:
		}

	}
	logger.Info().Int64("bundleId", lastBundle).Msg("Finished crawling to bundle.")
}

func (crawler *BundleCrawler) Start() {
	scheduler := gocron.NewScheduler(time.UTC)
	scheduler.Every(3).Minutes().Do(crawler.crawlBundles)

	for i := 0; i < cap(crawler.buffer); i++ {
		go crawler.bundleWorker()
	}

	scheduler.StartBlocking()
}

func CreateBundleCrawler(restEndpoint string, storageRest string, adapter db.Adapter, poolId int64) BundleCrawler {
	buffer := make(chan int64, 8)
	err := make(chan error)
	return BundleCrawler{restEndpoint: restEndpoint, storageRest: storageRest, adapter: adapter, poolId: poolId, buffer: buffer, err: err}
}

func Create() Crawler {
	var bundleCrawler []*BundleCrawler
	for _, bc := range config.GetCrawlerConfig() {
		adapter := bc.GetDatabaseAdapter()
		newCrawler := CreateBundleCrawler(bc.ChainRest, bc.StorageRest, adapter, bc.PoolId)
		bundleCrawler = append(bundleCrawler, &newCrawler)
	}

	return Crawler{
		children: bundleCrawler,
	}
}

func (c *Crawler) Start() {
	var wg sync.WaitGroup
	for _, bc := range c.children {
		wg.Add(1)
		go bc.Start()
	}
	wg.Wait()
}
