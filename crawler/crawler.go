package crawler

import (
	"context"
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
	"golang.org/x/sync/errgroup"
)

var (
	logger = utils.TrustlessApiLogger("crawler")
)

// Crawler is a master of mulitple child crawler
// One child crawler is responsible for crawling a specifc pool
type Crawler struct {
	children []*ChildCrawler
}

type ChildCrawler struct {
	chainId  string
	adapter  db.Adapter
	poolId   int64
	crawling sync.Mutex
}

// this is a helper function that will be called from multiple go routines
//
// downloads the bundle, creates the inclusion proof (merkle tree)
// and inserts the bundle into the database
func (crawler *ChildCrawler) insertBundleDataItems(bundleId int64) error {
	start := time.Now()

	compressedBundle, err := bundles.GetFinalizedBundle(crawler.chainId, crawler.poolId, bundleId)
	if err != nil {
		logger.Error().Msg("Something went wrong when retrieving the bundle...")
		return err
	}

	bundle, err := bundles.GetDecompressedBundle(*compressedBundle)

	if err != nil {
		logger.Error().Msg("Something went wrong when retrieving the bundle...")
		return err
	}

	elapsed := time.Since(start)
	logger.Debug().Msg(fmt.Sprintf("Downloading bundle took: %v", elapsed))

	leafs := merkle.GetBundleHashes(&bundle)

	var trustlessDataItems []types.TrustlessDataItem
	for _, dataitem := range bundle {
		proof, err := merkle.GetHashesCompact(leafs, &dataitem)
		if err != nil {
			return err
		}
		trustlessDataItem := types.TrustlessDataItem{Value: dataitem, Proof: proof, BundleId: bundleId, PoolId: crawler.poolId, ChainId: crawler.chainId}
		trustlessDataItems = append(trustlessDataItems, trustlessDataItem)
	}
	start = time.Now()
	err = crawler.adapter.Save(&trustlessDataItems)
	if err != nil {
		return err
	}
	elapsed = time.Since(start)
	logger.Debug().Int("size", len(trustlessDataItems)).Msg(fmt.Sprintf("Inserting data items took: %v", elapsed))

	return nil
}

// Crawls the latest bundles and processes them.
// Gets the latest pool info of the selected pool and calls `insertBundleDataItems` for each bundle that has not been processed yet.
// Will return as soon as some insertion fails
//
// NOTE: this function is thread safe, meaning it will return instanly if the crawler has not finished crawling the bundles yet
func (crawler *ChildCrawler) CrawlBundles() {

	if !crawler.crawling.TryLock() {
		logger.Info().Msg("Still crawling bundles!")
		return
	}

	// create new error group with context
	// because we want to stop the crawling processes as soon as one request fails and start over again
	group, ctx := errgroup.WithContext(context.Background())
	group.SetLimit(4)

	poolInfo, err := pool.GetPoolInfo(crawler.chainId, crawler.poolId)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get latest bundle")
		return
	}

	lastBundle := poolInfo.Pool.Data.TotalBundles

	for i := int64(0); i < lastBundle; i++ {
		if crawler.adapter.Exists(i) {
			continue
		}
		logger.Info().Msg(fmt.Sprintf("Inserting data items: %v/%v", i, lastBundle-1))
		localIndex := i

		group.Go(func() error {
			return crawler.insertBundleDataItems(localIndex)
		})

		// if the context was cancled we don't we return
		select {
		case <-ctx.Done():
			err := group.Wait() // get the error
			logger.Error().Err(err).Msg("Failed to process bundle...")
			return
		default:
		}
	}

	// wait until all bundles are uploaded
	if err := group.Wait(); err != nil {
		logger.Error().Err(err).Msg("Failed to process bundle...")
	}

	logger.Info().Int64("bundleId", lastBundle-1).Msg("Finished crawling to bundle.")
}

// starts the crawling processes
// creates a scheduler that invokes CrawlBundles
// this function is blocking
func (crawler *ChildCrawler) Start() {
	scheduler := gocron.NewScheduler(time.UTC)
	scheduler.Every(30).Seconds().Do(crawler.CrawlBundles)
	scheduler.StartBlocking()
}

func CreateBundleCrawler(adapter db.Adapter, chainId string, poolId int64) ChildCrawler {
	return ChildCrawler{adapter: adapter, poolId: poolId, chainId: chainId}
}

// Creates a crawler based on the config file
func Create() Crawler {
	var bundleCrawler []*ChildCrawler
	for _, bc := range config.GetPoolsConfig() {
		adapter := bc.GetDatabaseAdapter()
		newCrawler := CreateBundleCrawler(adapter, bc.ChainId, bc.PoolId)
		bundleCrawler = append(bundleCrawler, &newCrawler)
	}

	return Crawler{
		children: bundleCrawler,
	}
}

// starts the crawling process for each child crawler
// this function is blocking
func (c *Crawler) Start() {
	var wg sync.WaitGroup
	for _, bc := range c.children {
		current := bc
		wg.Add(1)
		go func() {
			current.Start()
			wg.Done()
		}()
	}
	wg.Wait()
}
