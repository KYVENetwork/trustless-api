package crawler

import (
	"os"
	"testing"

	"github.com/KYVENetwork/trustless-api/collectors/pool"
	"github.com/KYVENetwork/trustless-api/config"
	"github.com/KYVENetwork/trustless-api/db/adapters"
	"github.com/KYVENetwork/trustless-api/files"
	"github.com/KYVENetwork/trustless-api/indexer"
	"github.com/spf13/viper"
	"golang.org/x/sync/semaphore"
)

func TestOsmosisLast50Bundles(t *testing.T) {

	save := files.LocalFileAdapter

	config.LoadDefaults()

	// remove old db
	os.Remove("db_test.db")

	viper.Set("database.dbname", "db_test.db")
	viper.Set("crawler.threads", 4)

	// get latests api info from osmosis pool
	p, e := pool.GetPoolInfo("kyve-1", 1)
	if e != nil {
		panic(e)
	}

	adapter := adapters.GetSQLite(&save, &indexer.TendermintIndexer, 1, "kyve-1")

	c := CreateBundleCrawler(&adapter, "kyve-1", 1, p.Pool.Data.TotalBundles-50, semaphore.NewWeighted(16))

	c.CrawlBundles()
}
