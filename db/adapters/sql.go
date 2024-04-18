package adapters

import (
	"github.com/KYVENetwork/trustless-rpc/db"
	"github.com/KYVENetwork/trustless-rpc/files"
	"github.com/KYVENetwork/trustless-rpc/indexer"
	"github.com/KYVENetwork/trustless-rpc/types"
	"github.com/KYVENetwork/trustless-rpc/utils"
	"github.com/spf13/viper"
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

func (adapter *SQLAdapter) Save(dataitems *[]types.TrustlessDataItem) error {
	return adapter.db.Transaction(func(tx *gorm.DB) error {
		for _, dataitem := range *dataitems {
			file, err := adapter.saveDataItem.Save(&dataitem)
			if err != nil {
				logger.Fatal().Err(err).Msg("Faild to save dataitem")
				return err
			}
			item := db.DataItemDocument{BundleID: dataitem.BundleId, PoolId: dataitem.PoolId, FileType: file.Type, FilePath: file.Path}
			err = tx.Table(adapter.dataItemTable).Create(&item).Error
			if err != nil {
				logger.Fatal().Err(err).Msg("Faild to save dataitem")
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
		}
		return nil
	})
}

func (adapter *SQLAdapter) Get(dataitemKey int64, indexId int) (files.SavedFile, error) {

	query := db.IndexDocument{IndexID: indexId, Key: dataitemKey}
	err := adapter.db.Table(adapter.indexTable).Model(&db.IndexDocument{}).Find(&query).Error
	if err != nil {
		return files.SavedFile{}, err
	}
	result := db.DataItemDocument{}
	err = adapter.db.Table(adapter.dataItemTable).Model(&db.DataItemDocument{}).Find(&result, query.DataItemID).Error
	if err != nil {
		return files.SavedFile{}, err
	}
	return files.SavedFile{Path: result.FilePath, Type: result.FileType}, nil
}

func (adapter *SQLAdapter) Exists(bundleId int64) bool {
	query := db.DataItemDocument{BundleID: bundleId}
	var count int64
	err := adapter.db.Table(adapter.dataItemTable).Model(&db.DataItemDocument{}).Where(&query).Count(&count).Error
	if err != nil {
		return false
	}
	return count > 0
}
