package db

import (
	"fmt"
	"strings"

	"github.com/KYVENetwork/trustless-api/files"
	"github.com/KYVENetwork/trustless-api/indexer"
	"github.com/KYVENetwork/trustless-api/types"
)

type DataItemDocument struct {
	ID       uint `gorm:"primarykey"`
	BundleID int64
	FileType int
	FilePath string
}

type IndexDocument struct {
	Value      string `gorm:"primarykey"`
	IndexID    int    `gorm:"primarykey"`
	DataItemID uint
}

type Adapter interface {
	Save(bundle *types.Bundle) error
	Get(indexId int, key string) (files.SavedFile, error)
	GetMissingBundles(bundleStartId, lastBundleId int64) []int64
	GetIndexer() indexer.Indexer
}

func GetTableNames(poolId int64, chainId string) (string, string) {

	chainId = strings.ReplaceAll(chainId, "-", "_")

	return fmt.Sprintf("data_items_pool_%v_%v", chainId, poolId),
		fmt.Sprintf("indices_pool_%v_%v", chainId, poolId)
}
