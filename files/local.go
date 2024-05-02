package files

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/KYVENetwork/trustless-api/types"
	"github.com/spf13/viper"
)

type SaveLocalFileInterface struct{}

func (saveFile *SaveLocalFileInterface) Save(dataitem *types.TrustlessDataItem) (SavedFile, error) {

	// unmarshal item
	json, err := json.Marshal(dataitem)

	if err != nil {
		return SavedFile{}, err
	}

	path := viper.GetString("storage.path")
	dir := fmt.Sprintf("%v/%v/%v", path, dataitem.PoolId, dataitem.BundleId)

	// create directories if we need them
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return SavedFile{}, err
	}

	filepath := fmt.Sprintf("%v/%v.json", dir, dataitem.Value.Key)
	// create the file
	file, err := os.Create(filepath)
	if err != nil {
		return SavedFile{}, err
	}

	_, err = file.Write(json)
	if err != nil {
		return SavedFile{}, err
	}

	return SavedFile{Type: LocalFile, Path: filepath}, nil
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
