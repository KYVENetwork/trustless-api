package adapters

import (
	"fmt"
	"sync"

	"github.com/KYVENetwork/trustless-rpc/db"
	"github.com/KYVENetwork/trustless-rpc/files"
	"github.com/KYVENetwork/trustless-rpc/indexer"
	"github.com/KYVENetwork/trustless-rpc/types"
	"github.com/KYVENetwork/trustless-rpc/utils"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	logger = utils.TrustlessRpcLogger("DB")
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
		logger.Fatal().Err(err).Msg("Cannot open datase.")
	}

	dataItemTable, indexTable := db.GetTableNames(poolId)

	// Migrate the schema
	database.Table(dataItemTable).AutoMigrate(&db.DataItemDocument{})
	database.Table(indexTable).AutoMigrate(&db.IndexDocument{})

	return SQLAdapter{db: database, saveDataItem: saveDataItem, indexer: indexer, dataItemTable: dataItemTable, indexTable: indexTable}
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

	return SQLAdapter{db: database, saveDataItem: saveDataItem, indexer: indexer, dataItemTable: dataItemTable, indexTable: indexTable}
}

func (adapter *SQLAdapter) Save(dataitems *[]types.TrustlessDataItem) error {
	return adapter.db.Transaction(func(tx *gorm.DB) error {
		for _, dataitem := range *dataitems {
			var mutex sync.Mutex
			var g errgroup.Group

			g.Go(func() error {
				file, err := adapter.saveDataItem.Save(&dataitem)

				mutex.Lock()
				defer mutex.Unlock()

				if err != nil {
					logger.Error().Err(err).Msg("Faild to save dataitem")
					return err
				}
				item := db.DataItemDocument{BundleID: dataitem.BundleId, PoolId: dataitem.PoolId, FileType: file.Type, FilePath: file.Path}
				err = tx.Table(adapter.dataItemTable).Create(&item).Error
				if err != nil {
					logger.Error().Err(err).Msg("Faild to save dataitem")
					return err
				}

				keys, err := adapter.indexer.GetDataItemIndicies(&dataitem)
				if err != nil {
					logger.Error().Err(err).Msg("Faild to get dataitem indicies")
					return err
				}

				for keyIndex, key := range keys {
					err = tx.Table(adapter.indexTable).Create(&db.IndexDocument{DataItemID: item.ID, IndexID: keyIndex, Key: key}).Error
					if err != nil {
						logger.Error().Err(err).Msg("Faild to add index")
						return err
					}
				}
				return nil
			})
			if err := g.Wait(); err != nil {
				return err
			}
		}
		return nil
	})
}

func (adapter *SQLAdapter) Get(dataitemKey int64, indexId int) (files.SavedFile, error) {

	result := db.DataItemDocument{}
	query := db.IndexDocument{IndexID: indexId, Key: dataitemKey}
	joinString := fmt.Sprintf("join %v on %v.id = %v.data_item_id", adapter.dataItemTable, adapter.dataItemTable, adapter.indexTable)
	rows := adapter.db.Table(adapter.indexTable).Joins(joinString).Where(&query).Scan(&result)
	if rows.Error != nil {
		return files.SavedFile{}, rows.Error
	}
	if rows.RowsAffected == 0 {
		return files.SavedFile{}, fmt.Errorf("data item not found")
	}
	return files.SavedFile{Path: result.FilePath, Type: result.FileType}, nil
}

func (adapter *SQLAdapter) Exists(bundleId int64) bool {
	query := db.DataItemDocument{BundleID: bundleId}
	var count int64
	err := adapter.db.Table(adapter.dataItemTable).Where(&query).Count(&count).Error
	if err != nil {
		return false
	}
	return count > 0
}
