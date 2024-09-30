package files

import (
	"fmt"

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
	// Save returns a SavedFile.
	// It saves a data item on some form of FileStorage, this can be any storage like: S3, local, etc.
	Save(dataItem *types.TrustlessDataItem) (SavedFile, error)
}

type Get func(indexId int, key string) (SavedFile, error)

func (file *SavedFile) Resolve() ([]byte, error) {

	var rawFile []byte

	switch file.Type {
	case LocalFile:
		return LoadLocalFile(file.Path)
	case S3File:
		return LoadS3File(file.Path)
	}

	return rawFile, fmt.Errorf("unkown file type %v", file.Type)
}
