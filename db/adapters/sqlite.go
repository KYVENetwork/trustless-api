package adapters

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/KYVENetwork/trustless-rpc/files"
	"github.com/KYVENetwork/trustless-rpc/indexer"
	"github.com/KYVENetwork/trustless-rpc/types"
	"github.com/KYVENetwork/trustless-rpc/utils"
	_ "github.com/mattn/go-sqlite3"
)

var (
	logger = utils.TrustlessRpcLogger("DB")
)

type SQLiteAdapter struct {
	db           *sql.DB
	saveDataItem files.SaveDataItem
	indexer      indexer.Indexer
}

func StartSQLite(saveDataItem files.SaveDataItem, indexer indexer.Indexer) SQLiteAdapter {

	path := os.Getenv("DB")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		logger.Fatal().Err(err).Msg("Open Folder")
	}

	sqlStmt := `
	create table if not exists dataitems(bundleId bigint, poolId bigint, filepath varchar(255), fileType int);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		logger.Fatal().Err(err).Msg("Create Table")
	}

	for key := 0; key < indexer.GetIndexCount(); key++ {
		sqlStmt := fmt.Sprintf("create table if not exists index_%v(key integer primary key, dataitem integer);", key)
		_, err = db.Exec(sqlStmt)
		if err != nil {
			logger.Fatal().Err(err).Msg("Create Index")
		}
	}

	return SQLiteAdapter{db: db, saveDataItem: saveDataItem, indexer: indexer}
}

func (adapter *SQLiteAdapter) Save(dataitems *[]types.TrustlessDataItem) error {
	tx, err := adapter.db.Begin()
	if err != nil {
		logger.Error().Err(err).Msg("Faild to create txn")
		return err
	}

	logger.Debug().Int("n", len(*dataitems)).Msg("Adding data items")

	for _, dataitem := range *dataitems {
		file, err := adapter.saveDataItem.Save(&dataitem)
		if err != nil {
			logger.Fatal().Err(err).Msg("Faild to save dataitem")
			return err
		}

		// First insert the dataitem
		result, err := tx.Exec("insert into dataitems(bundleId, poolId, filePath, fileType) values(?, ?, ?, ?)", dataitem.BundleId, dataitem.PoolId, file.Path, file.Type)
		if err != nil {
			logger.Error().Err(err).Msg("Faild to add dataitem")
			return err
		}

		dataitemId, err := result.LastInsertId()
		if err != nil {
			logger.Error().Err(err).Msg("Faild to get DataItem ID")
			return err
		}

		keys, err := adapter.indexer.GetDataItemIndicies(&dataitem)
		if err != nil {
			logger.Error().Err(err).Msg("Faild to get dataitem indicies")
			return err
		}

		for keyIndex, key := range keys {
			_, err = tx.Exec(fmt.Sprintf("insert into index_%v(key, dataitem) values(?, ?)", keyIndex), key, dataitemId)
			if err != nil {
				logger.Error().Err(err).Msg("Faild to add index")
				return err
			}
		}
	}
	err = tx.Commit()
	if err != nil {
		logger.Error().Err(err).Msg("Faild to commit txn")
		return err
	}
	return nil
}

func (adapter *SQLiteAdapter) Get(dataitemKey string, index int) (files.SavedFile, error) {
	stmt, err := adapter.db.Prepare(fmt.Sprintf("select filePath, fileType from dataitems d, index_%v i where i.key = ? AND i.dataitem = d.rowid", index))
	if err != nil {
		logger.Error().Err(err).Msg("Failed ot find dataitem")
		return files.SavedFile{}, err
	}
	defer stmt.Close()
	var filePath string
	var fileType int
	err = stmt.QueryRow(dataitemKey).Scan(&filePath, &fileType)
	if err != nil {
		logger.Info().Str("key", dataitemKey).Msg("DataItem not found!")
		return files.SavedFile{}, err
	}

	return files.SavedFile{Path: filePath, Type: fileType}, nil
}

func (adapter *SQLiteAdapter) Exists(bundleId int64) bool {
	rows, err := adapter.db.Query("select * from dataitems where bundleId = ? LIMIT 1", bundleId)
	if err != nil {
		logger.Fatal().Err(err)
	}
	defer rows.Close()
	return rows.Next()
}
