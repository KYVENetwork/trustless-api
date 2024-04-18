package adapters

import (
	"os"

	"github.com/KYVENetwork/trustless-rpc/db"
	"github.com/KYVENetwork/trustless-rpc/files"
	"github.com/KYVENetwork/trustless-rpc/indexer"
	"github.com/KYVENetwork/trustless-rpc/types"
	"github.com/KYVENetwork/trustless-rpc/utils"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	logger = utils.TrustlessRpcLogger("DB")
)

type SQLiteAdapter struct {
	db           *gorm.DB
	saveDataItem files.SaveDataItem
	indexer      indexer.Indexer
}

func StartSQLite(saveDataItem files.SaveDataItem, indexer indexer.Indexer) SQLiteAdapter {
	path := os.Getenv("DB")
	database, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		logger.Fatal().Err(err).Msg("Open Folder")
	}

	// Migrate the schema
	database.AutoMigrate(&db.DataItemDocument{})
	database.AutoMigrate(&db.IndexDocument{})

	return SQLiteAdapter{db: database, saveDataItem: saveDataItem, indexer: indexer}
}

func (adapter *SQLiteAdapter) Save(dataitems *[]types.TrustlessDataItem) error {
	return adapter.db.Transaction(func(tx *gorm.DB) error {
		for _, dataitem := range *dataitems {
			file, err := adapter.saveDataItem.Save(&dataitem)
			if err != nil {
				logger.Fatal().Err(err).Msg("Faild to save dataitem")
				return err
			}
			item := db.DataItemDocument{BundleID: dataitem.BundleId, PoolId: dataitem.PoolId, FileType: file.Type, FilePath: file.Path}
			err = tx.Create(&item).Error
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
				err = tx.Create(&db.IndexDocument{DataItemDocumentID: item.ID, IndexID: keyIndex, Key: key}).Error
				if err != nil {
					logger.Error().Err(err).Msg("Faild to add index")
					return err
				}
			}
		}
		return nil
	})
}

func (adapter *SQLiteAdapter) Get(dataitemKey int64, indexId int) (files.SavedFile, error) {

	query := db.IndexDocument{IndexID: indexId, Key: dataitemKey}
	result := adapter.db.Preload("DataItemDocument").First(&query)
	if result.Error != nil {
		return files.SavedFile{}, result.Error
	}
	return files.SavedFile{Path: query.DataItemDocument.FilePath, Type: query.DataItemDocument.FileType}, nil
}

func (adapter *SQLiteAdapter) Exists(bundleId int64) bool {
	query := db.DataItemDocument{BundleID: 2}
	err := adapter.db.First(&query).Error
	if err != nil {
		return false
	}
	return query.ID != 0 && query.BundleID == bundleId
}
