package db

import (
	"github.com/KYVENetwork/trustless-rpc/files"
	"github.com/KYVENetwork/trustless-rpc/types"
)

type Adapter interface {
	Save(dataitem *[]types.TrustlessDataItem) error
	Get(dataitemKey string, index int) (files.SavedFile, error)
	Exists(bundle int64) bool
}
