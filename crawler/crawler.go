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
	"github.com/KYVENetwork/trustless-api/types"
	"github.com/KYVENetwork/trustless-api/utils"
	"github.com/go-co-op/gocron"
	"github.com/spf13/viper"
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
		logger.Error().Int64("poolId", crawler.poolId).Msg("Something went wrong when retrieving the bundle...")
		return err
	}

	dataitems, err := bundles.GetDecompressedBundle(*compressedBundle)

	if err != nil {
		logger.Error().Int64("poolId", crawler.poolId).Msg("Something went wrong when retrieving the bundle...")
		return err
	}

	elapsed := time.Since(start)
	logger.Debug().Int64("poolId", crawler.poolId).Msg(fmt.Sprintf("Downloading bundle took: %v", elapsed))

	bundle := types.Bundle{
		DataItems: dataitems,
		PoolId:    crawler.poolId,
		BundleId:  bundleId,
		ChainId:   crawler.chainId,
	}
	start = time.Now()
	err = crawler.adapter.Save(&bundle)
	if err != nil {
		return err
	}
	elapsed = time.Since(start)
	logger.Debug().Int64("poolId", crawler.poolId).Msg(fmt.Sprintf("Inserting data items took: %v", elapsed))

	return nil
}

// Crawls the latest bundles and processes them.
// Gets the latest pool info of the selected pool and calls `insertBundleDataItems` for each bundle that has not been processed yet.
// Will return as soon as some insertion fails
//
// NOTE: this function is thread safe, meaning it will return instanly if the crawler has not finished crawling the bundles yet
func (crawler *ChildCrawler) CrawlBundles() {

	if !crawler.crawling.TryLock() {
		logger.Info().Int64("poolId", crawler.poolId).Msg("Still crawling bundles!")
		return
	}

	// create new error group with context
	// because we want to stop the crawling processes as soon as one request fails and start over again
	group, ctx := errgroup.WithContext(context.Background())
	group.SetLimit(viper.GetInt("crawler.threads"))

	poolInfo, err := pool.GetPoolInfo(crawler.chainId, crawler.poolId)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get latest bundle")
		return
	}

	lastBundle := poolInfo.Pool.Data.TotalBundles - 1
	missingBundles := crawler.adapter.GetMissingBundles(lastBundle)

	for _, i := range missingBundles {
		logger.Info().Int64("poolId", crawler.poolId).Msg(fmt.Sprintf("Inserting data items: %v/%v", i, lastBundle))
		localIndex := i
		group.Go(func() error {
			return crawler.insertBundleDataItems(localIndex)
		})

		// if the context was cancled we don't we return
		select {
		case <-ctx.Done():
			err := group.Wait() // get the error
			logger.Error().Int64("poolId", crawler.poolId).Err(err).Msg("Failed to process bundle...")
			return
		default:
		}
	}

	// wait until all bundles are uploaded
	if err := group.Wait(); err != nil {
		logger.Error().Err(err).Msg("Failed to process bundle...")
	}

	logger.Info().Int64("bundleId", lastBundle).Int64("poolId", crawler.poolId).Msg("Finished crawling to bundle.")
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
