package db

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/KYVENetwork/trustless-rpc/types"
	"github.com/KYVENetwork/trustless-rpc/utils"
)

type SaveLocalFileInterface struct {
}

func (saveFile *SaveLocalFileInterface) Save(dataitem types.TrustlessDataItem) (types.SavedFile, error) {

	json, err := json.Marshal(dataitem)

	if err != nil {
		return types.SavedFile{}, err
	}
	path := os.Getenv("DATA_DIR")
	filepath := fmt.Sprintf("%v/%v.json", path, dataitem.Value.Key)

	file, err := os.Create(filepath)
	if err != nil {
		return types.SavedFile{}, err
	}
	file.Write(json)

	return types.SavedFile{Type: utils.LocalFile, Path: filepath}, nil
}

func (saveFile *SaveLocalFileInterface) Load(link string) (types.TrustlessDataItem, error) {

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
