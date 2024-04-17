package db

import (
	"github.com/KYVENetwork/trustless-rpc/files"
	"github.com/KYVENetwork/trustless-rpc/types"
)

type DataItemDocument struct {
	ID       int64 `bun:",pk,autoincrement"`
	BundleID int64
	PoolId   int64
	FileType int
	FilePath string
}

type IndexDocument struct {
	Key        int64
	DataItemID int64
}

type Adapter interface {
	Save(dataitem *[]types.TrustlessDataItem) error
	Get(dataitemKey string, index int) (files.SavedFile, error)
	Exists(bundle int64) bool
}
