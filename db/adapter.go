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
	Value      string `gorm:"primarykey"`
	IndexID    int    `gorm:"primarykey"`
	DataItemID uint
}

type Adapter interface {
	Save(dataitem *types.Bundle) error
	Get(indexId int, key string) (files.SavedFile, error)
	GetMissingBundles(lastBundle int64) []int64
	GetIndexer() indexer.Indexer
}

func GetTableNames(poolId int64) (string, string) {
	return fmt.Sprintf("data_items_pool_%v", poolId),
		fmt.Sprintf("indices_pool_%v", poolId)
}
