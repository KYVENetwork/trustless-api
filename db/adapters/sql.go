package adapters

import (
	"fmt"
	"sync"
	"time"

	"github.com/KYVENetwork/trustless-api/db"
	"github.com/KYVENetwork/trustless-api/files"
	"github.com/KYVENetwork/trustless-api/indexer"
	"github.com/KYVENetwork/trustless-api/types"
	"github.com/KYVENetwork/trustless-api/utils"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

var (
	logger = utils.TrustlessApiLogger("DB")
	mutex  sync.Mutex
)

type SQLAdapter struct {
	db            *gorm.DB
	saveDataItem  files.SaveDataItem
	indexer       indexer.Indexer
	dataItemTable string
	indexTable    string
}

func GetSQLite(saveDataItem files.SaveDataItem, indexer indexer.Indexer, poolId int64, chainId string) SQLAdapter {

	dns := viper.GetString("database.dbname")
	database, err := gorm.Open(sqlite.Open(dns), &gorm.Config{
		SkipDefaultTransaction: false,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("Cannot open database.")
	}

	dataItemTable, indexTable := db.GetTableNames(poolId, chainId)

	// Migrate the schema
	database.Table(dataItemTable).AutoMigrate(&db.DataItemDocument{})
	database.Table(indexTable).AutoMigrate(&db.IndexDocument{})

	return SQLAdapter{
		db:            database,
		saveDataItem:  saveDataItem,
		indexer:       indexer,
		dataItemTable: dataItemTable,
		indexTable:    indexTable,
	}
}

func GetPostgres(saveDataItem files.SaveDataItem, indexer indexer.Indexer, poolId int64, chainId string) SQLAdapter {
	dsn := fmt.Sprintf(
		"host=%v user=%v password=%v dbname=%v port=%v",
		viper.GetString("database.host"),
		viper.GetString("database.user"),
		viper.GetString("database.password"),
		viper.GetString("database.dbname"),
		viper.GetString("database.port"),
	)

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Error),
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("Cannot open database.")
	}

	dataItemTable, indexTable := db.GetTableNames(poolId, chainId)

	// Migrate the schema
	database.Table(dataItemTable).AutoMigrate(&db.DataItemDocument{})
	database.Table(indexTable).AutoMigrate(&db.IndexDocument{})

	return SQLAdapter{
		db:            database,
		saveDataItem:  saveDataItem,
		indexer:       indexer,
		dataItemTable: dataItemTable,
		indexTable:    indexTable,
	}
}

// Save inserts the data items provided into the database.
// The entire array is inserted as one transaction ensuring we don't have incomplete data.
//
// NOTE: This function is thread safe.
func (adapter *SQLAdapter) Save(bundle *types.Bundle) error {
	start := time.Now()
	dataItems, err := adapter.indexer.IndexBundle(bundle)
	if err != nil {
		return err
	}

	logger.Debug().
		Int64("bundleId", bundle.BundleId).
		Int64("poolId", bundle.PoolId).
		Msg(fmt.Sprintf("indexed %v data items in %v", len(*dataItems), time.Since(start)))

	start = time.Now()

	type Result struct {
		item *types.TrustlessDataItem
		file files.SavedFile
	}

	var result []Result
	var m sync.Mutex
	var g errgroup.Group
	g.SetLimit(viper.GetInt("storage.threads"))
	for index := range *dataItems {
		localIndex := index
		g.Go(func() error {
			localDataItem := &(*dataItems)[localIndex]
			file, err := adapter.saveDataItem.Save(localDataItem)
			if err != nil {
				logger.Error().
					Err(err).
					Int64("bundleId", localDataItem.BundleId).
					Int64("poolId", localDataItem.PoolId).
					Msg("failed to save data item")
				return err
			}
			m.Lock()
			defer m.Unlock()
			result = append(result, Result{
				file: file,
				item: localDataItem,
			})
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	elapsed := time.Since(start)
	logger.Debug().
		Int64("bundleId", bundle.BundleId).
		Int64("poolId", bundle.PoolId).
		Msg(fmt.Sprintf("saving data items took: %v", elapsed))

	start = time.Now()
	// lock the entire module as we might have multiple database adapter instances at the same time
	mutex.Lock()
	defer mutex.Unlock()
	logger.Debug().
		Int64("bundleId", bundle.BundleId).
		Int64("poolId", bundle.PoolId).
		Msg(fmt.Sprintf("locked database in %v", time.Since(start)))

	items := make([]db.DataItemDocument, 0)
	indices := make([]db.IndexDocument, 0)

	for _, r := range result {
		file := r.file
		dataItem := r.item
		item := db.DataItemDocument{
			BundleID: dataItem.BundleId,
			FileType: file.Type,
			FilePath: file.Path,
		}
		items = append(items, item)
	}

	return adapter.db.Transaction(func(tx *gorm.DB) error {
		// first insert the data items, the ID will be written into the array
		err := tx.Table(adapter.dataItemTable).CreateInBatches(items, 50).Error
		if err != nil {
			logger.Error().
				Err(err).
				Int64("bundleId", bundle.BundleId).
				Int64("poolId", bundle.PoolId).
				Msg("Failed to insert dataitem into db")
			return err
		}

		// then set the data item ID for each index document
		for i, item := range items {
			for _, index := range result[i].item.Indices {
				index := db.IndexDocument{
					DataItemID: item.ID,
					Value:      index.Index,
					IndexID:    index.IndexId,
				}
				indices = append(indices, index)
			}
		}

		err = tx.Table(adapter.indexTable).CreateInBatches(indices, 50).Error
		if err != nil {
			logger.Error().
				Err(err).
				Int64("bundleId", bundle.BundleId).
				Int64("poolId", bundle.PoolId).
				Msg("Failed to insert index into db")
			return err
		}

		return nil
	})
}

func (adapter *SQLAdapter) Get(indexId int, key string) (files.SavedFile, error) {
	start := time.Now()

	result := db.DataItemDocument{}
	query := db.IndexDocument{IndexID: indexId, Value: key}

	// because we are using custom table names we can't leverage gorms preloading
	// therefore we have to write our own join query
	joinString := fmt.Sprintf("join %v on %v.id = %v.data_item_id", adapter.dataItemTable, adapter.dataItemTable, adapter.indexTable)
	rows := adapter.db.Table(adapter.indexTable).Joins(joinString).Where(&query).Scan(&result)
	elapsed := time.Since(start)
	logger.Debug().Msg(fmt.Sprintf("data item lookup took: %v", elapsed))

	if rows.Error != nil {
		return files.SavedFile{}, rows.Error
	}

	// data item is not found
	if rows.RowsAffected == 0 {
		return files.SavedFile{}, fmt.Errorf("data item not found")
	}

	return files.SavedFile{Path: result.FilePath, Type: result.FileType}, nil
}

func (adapter *SQLAdapter) GetMissingBundles(bundleStartId, lastBundle int64) []int64 {
	template := `WITH recursive ids AS
	(
		   SELECT %v AS id
		   UNION ALL
		   SELECT id + 1
		   FROM   ids
		   WHERE  id < %v )
	SELECT id
	FROM   ids
	WHERE  id NOT IN
		   (
					SELECT   bundle_id AS id
					FROM     %v
					WHERE    bundle_id <= %v
					GROUP BY bundle_id )`
	query := fmt.Sprintf(template, bundleStartId, lastBundle, adapter.dataItemTable, lastBundle)
	var ids []int64
	adapter.db.Raw(query).Scan(&ids)
	return ids
}

func (adapter *SQLAdapter) GetIndexer() indexer.Indexer {
	return adapter.indexer
}
