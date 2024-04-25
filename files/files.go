package files

import (
	"github.com/KYVENetwork/trustless-api/types"
)

type SavedFile struct {
	Type int
	Path string
}

const (
	LocalFile = iota
	S3File    = iota
)

var (
	LocalFileAdapter = SaveLocalFileInterface{}
	S3FileAdapter    = S3FileInterface{}
)

type SaveDataItem interface {
	Save(dataitem *types.TrustlessDataItem) (SavedFile, error)
}
