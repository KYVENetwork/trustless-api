package db

import (
	"github.com/KYVENetwork/trustless-rpc/types"
)

type Adapter interface {
	Save(dataitem types.TrustlessDataItem) error
	Get(dataitemKey string) error
}
