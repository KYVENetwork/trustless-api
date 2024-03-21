package indexer

import (
	"database/sql"
	"fmt"
	"github.com/KYVENetwork/trustless-rpc/collectors/bundles"
	"github.com/KYVENetwork/trustless-rpc/utils"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

var (
	mu     sync.Mutex
	logger = utils.TrustlessRpcLogger("indexer")
)

type SQLiteIndexer struct {
	ChainId      string
	DbPath       string
	RestEndpoint string
}

func (i *SQLiteIndexer) SetupDB(indexEnabled bool) error {
	if err := EnsureDBPathExists(i.DbPath); err != nil {
		return err
	}

	// Set up DB
	db, err := sql.Open("sqlite3", i.DbPath)
	if err != nil {
		return err
	}

	// Create the required table if it doesn't exist
	mu.Lock()
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS keyBundleMapping (
			data_item_key INTEGER,
			pool_id INTEGER NOT NULL,
			bundle_id INTEGER NOT NULL,
			chain_id TEXT NOT NULL,
			PRIMARY KEY (data_item_key, pool_id, chain_id)
		);
    `)
	mu.Unlock()
	if err != nil {
		logger.Error().Str("err", err.Error()).Msg("failed to create table")
		return err
	}

	if indexEnabled {
		mu.Lock()
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS key_pool_index ON keyBundleMapping(data_item_key, pool_id)")
		mu.Unlock()
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *SQLiteIndexer) CreateIndex(poolIds []int64) error {
	for _, poolId := range poolIds {
		latestKey, err := i.GetLatestKey(poolId)
		if err != nil {
			return err
		}

		if err := CreateKeyBundleMapping(i.RestEndpoint, poolId, i.DbPath, i.ChainId, latestKey); err != nil {
			return err
		}
	}

	return nil
}

func (i *SQLiteIndexer) GetBundleIdByKey(key int, poolId int) (int, error) {
	db, err := sql.Open("sqlite3", i.DbPath)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	query := "SELECT bundle_id FROM keyBundleMapping WHERE data_item_key = ? AND pool_id = ? AND chain_id"
	mu.Lock()
	rows, err := db.Query(query, key, poolId, i.ChainId)

	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var results []int

	for rows.Next() {
		var bundleId int
		err = rows.Scan(&bundleId)
		if err != nil {
			return 0, fmt.Errorf("failed to scan rows: %v", err.Error())
		}

		results = append(results, bundleId)
	}

	if len(results) > 1 {
		return 0, fmt.Errorf("internal error (received more than one bundle_id)")
	} else if len(results) < 1 {
		return 0, fmt.Errorf("couldn't find bundle_id for key %v in pool %v", key, poolId)
	}

	return results[0], nil
}

func (i *SQLiteIndexer) GetLatestKey(poolId int64) (int, error) {
	db, err := sql.Open("sqlite3", i.DbPath)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	query := "SELECT MAX(data_item_key) FROM keyBundleMapping WHERE pool_id = ? AND chain_id = ?;"
	mu.Lock()
	rows, err := db.Query(query, poolId, i.ChainId)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var latestKey sql.NullInt64

	for rows.Next() {
		err = rows.Scan(&latestKey)
		if err != nil {
			return 0, fmt.Errorf("failed to scan rows: %v", err.Error())
		}
	}

	if latestKey.Valid {
		logger.Info().Int64("key", latestKey.Int64).Msg("got latest key")
		return int(latestKey.Int64), nil
	} else {
		return 0, nil // No latest key found
	}
}
func EnsureDBPathExists(dbPath string) error {
	// Get the directory of the database path
	dir := filepath.Dir(dbPath)

	// Check if the directory exists
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		// Directory does not exist, create it
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}
	} else if err != nil {
		// An error occurred while checking the directory
		return fmt.Errorf("failed to check directory: %v", err)
	}

	// Create the file if it doesn't exist
	_, err = os.Stat(dbPath)
	if os.IsNotExist(err) {
		// File does not exist, create it
		file, err := os.Create(dbPath)
		if err != nil {
			return fmt.Errorf("failed to create file: %v", err)
		}
		defer file.Close()
	} else if err != nil {
		// An error occurred while checking the file
		return fmt.Errorf("failed to check file: %v", err)
	}

	return nil
}

func CreateKeyBundleMapping(restEndpoint string, poolId int64, dbPath string, chainId string, latestKey int) error {
	// Set up DB
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	// Prepare insert
	insert, err := db.Prepare("INSERT INTO keyBundleMapping(data_item_key, pool_id, bundle_id, chain_id) values(?,?,?,?)")
	if err != nil {
		return fmt.Errorf("failed to prepare insert: %v", err)
	}

	paginationKey := ""
FinalizedBundleCollector:
	for {
		finalizedBundles, nextKey, err := bundles.GetFinalizedBundlesPage(restEndpoint, poolId, utils.BundlesPageLimit, paginationKey)
		if err != nil {
			return fmt.Errorf("failed to get finalized bundles page: %w", err)
		}

		lastToKey, err := strconv.Atoi(finalizedBundles[len(finalizedBundles)-1].ToKey)
		if err != nil {
			return fmt.Errorf("failed to convert lastToKey: %v", err)
		}

		if lastToKey < latestKey {
			continue FinalizedBundleCollector
		}

	BundleCollector:
		for _, bundle := range finalizedBundles {
			from, err := strconv.Atoi(bundle.FromKey)
			if err != nil {
				return fmt.Errorf("failed to convert fromKey: %w", err)
			}
			to, err := strconv.Atoi(bundle.ToKey)
			if err != nil {
				return fmt.Errorf("failed to convert toKey: %w", err)
			}

			logger.Info().Int("to", to).Str("id", bundle.Id).Int("latestKey", latestKey).Msg("Bundle ID")

			if to < latestKey {
				logger.Info().Msg("continue BundleCollector")
				continue BundleCollector
			}

			for i := from; i <= to; i++ {
				logger.Info().Int("key", i).Int64("poolId", poolId).Msg("inserting...")
				mu.Lock()
				result, err := insert.Exec(i, poolId, bundle.Id, chainId)
				if err != nil {
					return fmt.Errorf("failed during query execution: %v", err)
				}

				logger.Info().Int("key", i).Int64("poolId", poolId).Msg("inserted")

				// Check the number of rows affected to determine if the insert was successful
				rowsAffected, _ := result.RowsAffected()
				if rowsAffected == 0 {
					return fmt.Errorf("no rows were affected, insert may have failed")
				}
				mu.Unlock()
			}
		}

		if nextKey == "" {
			return nil
		}
		paginationKey = nextKey
		continue
	}
}
