package files

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/KYVENetwork/trustless-rpc/types"
)

const (
	LocalFile = iota
	AWSFile   = iota
)

type SavedFile struct {
	Type int
	Path string
}

type SaveDataItem interface {
	Save(dataitem *types.TrustlessDataItem) (SavedFile, error)
	Load(link string) (types.TrustlessDataItem, error)
}

type SaveLocalFileInterface struct {
}

func (saveFile *SaveLocalFileInterface) Save(dataitem *types.TrustlessDataItem) (SavedFile, error) {

	json, err := json.Marshal(dataitem)

	if err != nil {
		return SavedFile{}, err
	}
	path := os.Getenv("DATA_DIR")
	filepath := fmt.Sprintf("%v/%v.json", path, dataitem.Value.Key)

	file, err := os.Create(filepath)
	if err != nil {
		return SavedFile{}, err
	}
	file.Write(json)

	return SavedFile{Type: LocalFile, Path: filepath}, nil
}

func (saveFile *SaveLocalFileInterface) Load(link string) (types.TrustlessDataItem, error) {
	return LoadLocalFile(link)
}

func LoadLocalFile(link string) (types.TrustlessDataItem, error) {
	file, err := os.ReadFile(link)

	if err != nil {
		return types.TrustlessDataItem{}, err
	}

	var dataItem types.TrustlessDataItem

	err = json.Unmarshal(file, &dataItem)
	if err != nil {
		return types.TrustlessDataItem{}, err
	}

	return dataItem, nil
}
