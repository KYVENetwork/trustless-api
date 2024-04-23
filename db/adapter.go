package db

import (
	"fmt"

	"github.com/KYVENetwork/trustless-rpc/files"
	"github.com/KYVENetwork/trustless-rpc/types"
)

type DataItemDocument struct {
	ID       uint  `gorm:"primarykey"`
	BundleID int64 `gorm:"index:bundleId"`
	PoolId   int64
	FileType int
	FilePath string
}

type IndexDocument struct {
	Key        int64 `gorm:"index:idx;primarykey"`
	IndexID    int   `gorm:"index:idx;primarykey"`
	DataItemID uint
}

type Adapter interface {
	Save(dataitem *[]types.TrustlessDataItem) error
	Get(dataitemKey int64, indexId int) (files.SavedFile, error)
	Exists(bundle int64) bool
}

func GetTableNames(poolId int64) (string, string) {
	return fmt.Sprintf("data_items_pool_%v", poolId), fmt.Sprintf("indexes_pool_%v", poolId)
}
