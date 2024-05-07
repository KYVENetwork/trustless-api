package db

import (
	"fmt"

	"github.com/KYVENetwork/trustless-api/files"
	"github.com/KYVENetwork/trustless-api/indexer"
	"github.com/KYVENetwork/trustless-api/types"
)

type DataItemDocument struct {
	ID       uint `gorm:"primarykey"`
	BundleID int64
	PoolID   int64
	FileType int
	FilePath string
}

type IndexDocument struct {
	ComponentID uint   `gorm:"primarykey"`
	DataItemID  uint   `gorm:"primarykey"`
	Value       string `gorm:"primarykey"`
	IndexID     int
}

type Adapter interface {
	Save(dataitem *types.Bundle) error
	Get(indexId int, keys ...string) (files.SavedFile, error)
	GetMissingBundles(lastBundle int64) []int64
	GetIndexer() indexer.Indexer
}

func GetTableNames(poolId int64) (string, string) {
	return fmt.Sprintf("data_items_pool_%v", poolId),
		fmt.Sprintf("indices_pool_%v", poolId)
}
