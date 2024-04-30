package db

import (
	"fmt"

	"github.com/KYVENetwork/trustless-api/files"
	"github.com/KYVENetwork/trustless-api/indexer"
	"github.com/KYVENetwork/trustless-api/types"
)

type DataItemDocument struct {
	ID       uint  `gorm:"primarykey"`
	BundleID int64 `gorm:"bundleId"`
	PoolID   int64
	FileType int
	FilePath string
}

type IndexDocument struct {
	Key        int64 `gorm:"primarykey"`
	IndexID    int   `gorm:"primarykey"`
	DataItemID uint
}

type Adapter interface {
	Save(dataitem *[]types.TrustlessDataItem) error
	Get(dataitemKey int64, indexId int) (files.SavedFile, error)
	Exists(bundle int64) bool
	GetIndexer() indexer.Indexer
}

func GetTableNames(poolId int64) (string, string) {
	return fmt.Sprintf("data_items_pool_%v", poolId), fmt.Sprintf("indices_pool_%v", poolId)
}
