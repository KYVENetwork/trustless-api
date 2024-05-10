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

func GetSQLite(saveDataItem files.SaveDataItem, indexer indexer.Indexer, poolId int64) SQLAdapter {

	database, err := gorm.Open(sqlite.Open(viper.GetString("database.dbname")), &gorm.Config{})
	if err != nil {
		logger.Fatal().Err(err).Msg("Cannot open database.")
	}

	dataItemTable, indexTable := db.GetTableNames(poolId)

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

func GetPostgres(saveDataItem files.SaveDataItem, indexer indexer.Indexer, poolId int64) SQLAdapter {
	dsn := fmt.Sprintf(
		"host=%v user=%v password=%v dbname=%v port=%v",
		viper.GetString("database.host"),
		viper.GetString("database.user"),
		viper.GetString("database.password"),
		viper.GetString("database.dbname"),
		viper.GetString("database.port"),
	)

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Fatal().Err(err).Msg("Cannot open datase.")
	}

	dataItemTable, indexTable := db.GetTableNames(poolId)

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

// inserts the dataitems provided into the database.
// the entire array is inserted as one transaction ensuring we don't have incomplete data
//
// NOTE: this function is thread safe
func (adapter *SQLAdapter) Save(bundle *types.Bundle) error {

	dataitems, err := adapter.indexer.IndexBundle(bundle)
	if err != nil {
		return err
	}

	type Result struct {
		item *types.TrustlessDataItem
		file files.SavedFile
	}

	var result []Result
	var m sync.Mutex
	var g errgroup.Group
	g.SetLimit(viper.GetInt("storage.threads"))
	for index := range *dataitems {
		localIndex := index
		g.Go(func() error {
			localDataitem := &(*dataitems)[localIndex]
			file, err := adapter.saveDataItem.Save(localDataitem)
			if err != nil {
				logger.Error().
					Err(err).
					Int64("bundleId", localDataitem.BundleId).
					Int64("poolId", localDataitem.PoolId).
					Msg("failed to save data item")
				return err
			}
			m.Lock()
			defer m.Unlock()
			result = append(result, Result{
				file: file,
				item: localDataitem,
			})
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	// lock the entire module as we might have multiple data base adapter instances at the same time
	mutex.Lock()
	defer mutex.Unlock()

	return adapter.db.Transaction(func(tx *gorm.DB) error {
		for _, r := range result {
			file := r.file
			dataitem := r.item
			item := db.DataItemDocument{
				BundleID: dataitem.BundleId,
				PoolID:   dataitem.PoolId,
				FileType: file.Type,
				FilePath: file.Path,
			}
			err := tx.Table(adapter.dataItemTable).Create(&item).Error
			if err != nil {
				logger.Error().
					Err(err).
					Int64("bundleId", dataitem.BundleId).
					Int64("poolId", dataitem.PoolId).
					Msg("Failed to insert dataitem into db")
				return err
			}

			for _, index := range dataitem.Indices {
				index := db.IndexDocument{
					DataItemID: item.ID,
					Value:      index.Index,
					IndexID:    index.IndexId,
				}
				err = tx.Table(adapter.indexTable).Create(&index).Error
				if err != nil {
					logger.Error().
						Err(err).
						Int64("bundleId", dataitem.BundleId).
						Int64("poolId", dataitem.PoolId).
						Msg("Failed to insert index into db")
					return err
				}
			}
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

func (adapter *SQLAdapter) GetMissingBundles(lastBundle int64) []int64 {
	template := `WITH recursive ids AS
	(
		   SELECT 1 AS id
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
	query := fmt.Sprintf(template, lastBundle, adapter.dataItemTable, lastBundle)
	var ids []int64
	adapter.db.Raw(query).Scan(&ids)
	return ids
}

func (adapter *SQLAdapter) GetIndexer() indexer.Indexer {
	return adapter.indexer
}
