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
)

var (
	logger = utils.TrustlessApiLogger("crawler")
)

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

	start = time.Now()
	var trustlessDataItems []types.TrustlessDataItem
	for _, dataitem := range bundle {
		proof, err := merkle.GetHashesCompact(leafs, &dataitem)
		if err != nil {
			return err
		}
		trustlessDataItem := types.TrustlessDataItem{Value: dataitem, Proof: proof, BundleId: bundleId, PoolId: crawler.poolId, ChainId: crawler.chainId}
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

// the bundle worker is responsible for receiving bundleIds & process them
// only receives bundleIds from `buffer` & only sends errors to the `errChannel`
// if the context was cancled, the worker will close
func (crawler *ChildCrawler) bundleWorker(buffer <-chan int64, errChannel chan<- error, ctx context.Context) {
	for {
		bundleId, ok := <-buffer
		if !ok {
			return
		}
		err := crawler.insertBundleDataItems(bundleId)
		// we have to check if the context was cancled, because if so, the error channel is closed
		select {
		case <-ctx.Done():
			return
		default:
			if err != nil {
				errChannel <- err
			}
		}
	}
}

func (crawler *ChildCrawler) CrawlBundles() {

	if !crawler.crawling.TryLock() {
		logger.Info().Msg("Still crawling bundles!")
		return
	}

	// create channel for bundleIds, we only create 8 as we only want to process 8 bundles simultaneously
	buffer := make(chan int64, 8)
	errChannel := make(chan error, 8)
	// create context to know when to cancle each go-routine
	ctx, cancle := context.WithCancel(context.Background())

	cleanUp := func() {
		cancle()
		close(buffer)
		close(errChannel)
		crawler.crawling.Unlock()
	}

	var wg sync.WaitGroup
	// create bundle worker according to the buffers size
	for i := 0; i < cap(buffer); i++ {
		wg.Add(1)
		go func() {
			crawler.bundleWorker(buffer, errChannel, ctx)
			wg.Done()
		}()
	}

	poolInfo, err := pool.GetPoolInfo(crawler.chainId, crawler.poolId)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get latest bundle")
		cleanUp()
		return
	}

	lastBundle := poolInfo.Pool.Data.TotalBundles

	for i := int64(0); i < 20; i++ {
		if crawler.adapter.Exists(i) {
			continue
		}
		logger.Info().Msg(fmt.Sprintf("Inserting data items: %v/%v", i, lastBundle-1))
		buffer <- i
		select {
		case <-errChannel:
			logger.Error().Err(err).Msg("Failed to process bundle...")
			cleanUp()
			return
		default:
		}
	}

	close(buffer)
	// wait until every worker is finished
	wg.Wait()
	select {
	case <-errChannel:
		logger.Error().Err(err).Msg("Failed to process bundle...")
		cancle()
		close(errChannel)
		return
	default:
	}
	logger.Info().Int64("bundleId", lastBundle-1).Msg("Finished crawling to bundle.")
}

func (crawler *ChildCrawler) Start() {
	scheduler := gocron.NewScheduler(time.UTC)
	scheduler.Every(30).Seconds().Do(crawler.CrawlBundles)
	scheduler.StartBlocking()
}

func CreateBundleCrawler(adapter db.Adapter, chainId string, poolId int64) ChildCrawler {
	return ChildCrawler{adapter: adapter, poolId: poolId, chainId: chainId}
}

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
