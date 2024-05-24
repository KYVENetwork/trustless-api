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
	// returns a SavedFile
	// saves an dataitem on some form of FileStorage, this can be any storage like: S3, local, etc.
	Save(dataitem *types.TrustlessDataItem) (SavedFile, error)
}
