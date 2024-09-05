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
	"golang.org/x/sync/semaphore"
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
	adapter       db.Adapter
	bundleStartId int64
	chainId       string
	crawling      sync.Mutex
	poolId        int64
	excludeProof  bool
	semaphore     *semaphore.Weighted
}

// This is a helper function that will be called from multiple go routines.
//
// Downloads the bundle, creates the inclusion proof
// and inserts the bundle into the database.
func (crawler *ChildCrawler) insertBundleDataItems(bundleId int64) error {
	start := time.Now()

	compressedBundle, err := bundles.GetFinalizedBundle(crawler.chainId, crawler.poolId, bundleId)
	if err != nil {
		logger.Error().Int64("poolId", crawler.poolId).Msg("Something went wrong when retrieving the bundle...")
		return err
	}

	dataItems, err := bundles.GetDecompressedBundle(*compressedBundle)

	if err != nil {
		logger.Error().Int64("poolId", crawler.poolId).Msg("Something went wrong when retrieving the bundle...")
		return err
	}

	elapsed := time.Since(start)
	logger.Debug().Int64("poolId", crawler.poolId).Msg(fmt.Sprintf("Downloading bundle took: %v", elapsed))

	bundle := types.Bundle{
		DataItems: dataItems,
		PoolId:    crawler.poolId,
		BundleId:  bundleId,
		ChainId:   crawler.chainId,
	}
	start = time.Now()

	err = crawler.adapter.Save(&bundle, crawler.excludeProof)
	if err != nil {
		return err
	}
	elapsed = time.Since(start)
	logger.Debug().Int64("poolId", crawler.poolId).Msg(fmt.Sprintf("Inserting data items took: %v", elapsed))

	return nil
}

// CrawlBundles crawls the latest bundles and processes them.
// Gets the latest pool info of the selected pool and calls `insertBundleDataItems` for each bundle that has not been processed yet.
// Will return as soon as some insertion fails.
//
// NOTE: This function is thread safe, meaning it will return instanly if the crawler has not finished crawling the bundles yet.
func (crawler *ChildCrawler) CrawlBundles() {

	if !crawler.crawling.TryLock() {
		logger.Info().Int64("poolId", crawler.poolId).Msg("Still crawling bundles!")
		return
	}

	defer crawler.crawling.Unlock()

	// create new error group with context
	// because we want to stop the crawling processes as soon as one request fails and start over again
	group, ctx := errgroup.WithContext(context.Background())

	poolInfo, err := pool.GetPoolInfo(crawler.chainId, crawler.poolId)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get latest bundle")
		return
	}

	lastBundle := poolInfo.Pool.Data.TotalBundles - 1
	missingBundles := crawler.adapter.GetMissingBundles(crawler.bundleStartId, lastBundle)

	for _, i := range missingBundles {
		crawler.semaphore.Acquire(ctx, 1)

		logger.Info().Int64("poolId", crawler.poolId).Int64("bundle-id", i).Msg(fmt.Sprintf("Inserting data items: %v/%v", i+1-crawler.bundleStartId, lastBundle+1-crawler.bundleStartId))
		localIndex := i
		group.Go(func() error {
			defer crawler.semaphore.Release(1)
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

// Start starts the crawling processes and
// creates a scheduler that invokes CrawlBundles.
// NOTE: This function is blocking.
func (crawler *ChildCrawler) Start() {
	scheduler := gocron.NewScheduler(time.UTC)
	scheduler.Every(30).Seconds().Do(crawler.CrawlBundles)
	scheduler.StartBlocking()
}

func CreateBundleCrawler(
	adapter db.Adapter,
	chainId string,
	poolId int64,
	bundleStartId int64,
	excludeProof bool,
	semaphore *semaphore.Weighted,
) ChildCrawler {
	return ChildCrawler{
		adapter:       adapter,
		bundleStartId: bundleStartId,
		chainId:       chainId,
		poolId:        poolId,
		excludeProof:  excludeProof,
		semaphore:     semaphore,
	}
}

// Create creates a crawler based on the config file.
func Create() Crawler {
	var bundleCrawler []*ChildCrawler

	semaphore := semaphore.NewWeighted(viper.GetInt64("crawler.threads"))

	for _, bc := range config.GetPoolsConfig() {
		adapter := bc.GetDatabaseAdapter()
		newCrawler := CreateBundleCrawler(adapter, bc.ChainId, bc.PoolId, bc.BundleStartId, bc.ExcludeProof, semaphore)
		bundleCrawler = append(bundleCrawler, &newCrawler)
	}

	return Crawler{
		children: bundleCrawler,
	}
}

// Start starts the crawling process for each child crawler.
// NOTE: This function is blocking.
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
