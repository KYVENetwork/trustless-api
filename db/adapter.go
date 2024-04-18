package db

import (
	"github.com/KYVENetwork/trustless-rpc/files"
	"github.com/KYVENetwork/trustless-rpc/types"
	"gorm.io/gorm"
)

type DataItemDocument struct {
	gorm.Model
	BundleID int64
	PoolId   int64
	FileType int
	FilePath string
}

type IndexDocument struct {
	gorm.Model
	Key                int64 `gorm:"index:idx"`
	IndexID            int   `gorm:"index:idx"`
	DataItemDocumentID uint
	DataItemDocument   DataItemDocument
}

type Adapter interface {
	Save(dataitem *[]types.TrustlessDataItem) error
	Get(dataitemKey int64, indexId int) (files.SavedFile, error)
	Exists(bundle int64) bool
}
