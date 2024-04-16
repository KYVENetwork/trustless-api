package adapters

import (
	"database/sql"
	"fmt"

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
	saveDataItem types.SaveDataItem
	indexer      indexer.Indexer
}

func StartSQLite(path string, saveDataItem types.SaveDataItem, indexer indexer.Indexer) (SQLiteAdapter, error) {

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		logger.Fatal().Err(err).Msg("Open Folder")
		return SQLiteAdapter{}, err
	}

	sqlStmt := `
	create table if not exists dataitems(bundleId bigint, poolId bigint, filepath varchar(255), fileType int);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		logger.Fatal().Err(err).Msg("Create Table")
		return SQLiteAdapter{}, err
	}

	for key := 0; key < indexer.GetIndexCount(); key++ {
		sqlStmt := fmt.Sprintf("create table if not exists index_%v(key varchar(255) primary key, dataitem integer);", key)
		_, err = db.Exec(sqlStmt)
		if err != nil {
			logger.Fatal().Err(err).Msg("Create Index")
			return SQLiteAdapter{}, err
		}
	}

	return SQLiteAdapter{db: db, saveDataItem: saveDataItem, indexer: indexer}, nil
}

func (adapter *SQLiteAdapter) Save(dataitem types.TrustlessDataItem) error {
	file := adapter.saveDataItem.Save(dataitem)

	tx, err := adapter.db.Begin()
	if err != nil {
		logger.Fatal().Err(err).Msg("Faild to add index")
		return nil
	}
	// First insert the dataitem
	result, err := tx.Exec("insert into dataitems(bundleId, poolId, filePath, fileType) values(?, ?, ?, ?)", dataitem.BundleId, dataitem.PoolId, file.Path, file.Type)
	if err != nil {
		logger.Fatal().Err(err).Msg("Faild to add index")
		return nil
	}

	dataitemId, err := result.LastInsertId()
	if err != nil {
		logger.Fatal().Err(err).Msg("Faild to get DataItem ID")
		return nil
	}

	keys, err := adapter.indexer.GetDataItemIndicies(&dataitem)
	if err != nil {
		logger.Fatal().Err(err).Msg("Faild to get dataitem indicies")
		return nil
	}

	for keyIndex, key := range keys {
		_, err = tx.Exec(fmt.Sprintf("insert into index_%v(key, dataitem) values(?, ?)", keyIndex), key, dataitemId)
		if err != nil {
			logger.Fatal().Err(err).Msg("Faild to add index")
			return nil
		}
	}

	err = tx.Commit()
	if err != nil {
		logger.Fatal().Err(err).Msg("Faild to commit txn")
		return nil
	}
	return nil
}

func (adapter *SQLiteAdapter) Get(dataitemKey string, index int) error {
	stmt, err := adapter.db.Prepare(fmt.Sprintf("select filePath, fileType from dataitems d, index_%v i where i.key = '?' AND i.dataitem = d.rowid", index))
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed ot find dataitem")
		return err
	}
	defer stmt.Close()
	var filePath string
	var fileType int
	err = stmt.QueryRow(dataitemKey).Scan(&filePath, &fileType)
	if err != nil {
		logger.Fatal().Err(err)
	}
	return nil
}

func (adapter *SQLiteAdapter) Exists(bundleId int64) bool {
	rows, err := adapter.db.Query("select * from dataitems where bundleId = ? LIMIT 1", bundleId)
	if err != nil {
		logger.Fatal().Err(err)
	}
	defer rows.Close()
	return rows.Next()
}
