package adapters

import (
	"database/sql"
	"fmt"

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
	getUniqueKey types.UniqueDataItemKey
}

func StartSQLite(dataDir string, saveDataItem types.SaveDataItem, getUniqueKey types.UniqueDataItemKey) (SQLiteAdapter, error) {

	db, err := sql.Open("sqlite3", fmt.Sprintf("%v/database.db", dataDir))
	if err != nil {
		logger.Fatal().Err(err)
		return SQLiteAdapter{}, err
	}

	sqlStmt := `
	create table if not exists dataitems(key varchar(255) not null primary key, bundleId bigint, poolId bigint, filepath varchar(255), fileType int);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		logger.Fatal().Err(err)
		return SQLiteAdapter{}, err
	}

	return SQLiteAdapter{db: db, saveDataItem: saveDataItem, getUniqueKey: getUniqueKey}, nil
}

func (adapter *SQLiteAdapter) Save(dataitem types.TrustlessDataItem) error {
	file := adapter.saveDataItem.Save(dataitem)

	tx, err := adapter.db.Begin()
	if err != nil {
		logger.Fatal().Err(err)
	}
	stmt, err := tx.Prepare("insert into dataitems(key, bundleId, poolId, filePath, fileType) values(?, ?, ?, ?, ?)")
	if err != nil {
		logger.Fatal().Err(err)
	}
	defer stmt.Close()
	key := adapter.getUniqueKey.GetUniqueKey(dataitem)
	_, err = stmt.Exec(key, dataitem.BundleId, dataitem.PoolId, file.Path, file.Type)
	if err != nil {
		logger.Fatal().Err(err)
	}
	err = tx.Commit()
	if err != nil {
		logger.Fatal().Err(err)
	}
	return nil
}

func (adapter *SQLiteAdapter) Get(dataitemKey string) error {
	stmt, err := adapter.db.Prepare("select filePath, fileType from dataitems where uniqueKey = '?'")
	if err != nil {
		logger.Fatal().Err(err)
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
