package db

import (
	"fmt"

	"github.com/KYVENetwork/trustless-rpc/types"
	"github.com/KYVENetwork/trustless-rpc/utils"
)

type KeyAsPrimaryInterface struct{}

func (key *KeyAsPrimaryInterface) GetUniqueKey(dataitem types.TrustlessDataItem) string {
	return fmt.Sprintf(dataitem.Value.Key)
}

type SaveLocalFileInterface struct{}

func (fiel *SaveLocalFileInterface) Save(dataitem types.TrustlessDataItem) types.SavedFile {
	return types.SavedFile{Type: utils.LocalFile, Path: "TODO"}
}

var (
	KeyAsPrimary  = KeyAsPrimaryInterface{}
	SaveLocalFile = SaveLocalFileInterface{}
)
