package db

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/KYVENetwork/trustless-rpc/types"
	"github.com/KYVENetwork/trustless-rpc/utils"
)

type SaveLocalFileInterface struct {
	DataDir string
}

func (saveFile *SaveLocalFileInterface) Save(dataitem types.TrustlessDataItem) types.SavedFile {

	json, err := json.Marshal(dataitem)

	if err != nil {
		fmt.Println("Something big went wrong....")
	}

	filepath := fmt.Sprintf("%v/%v.json", saveFile.DataDir, dataitem.Value.Key)

	file, err := os.Create(filepath)
	if err != nil {
		fmt.Println("Something big went wrong....")
	}
	file.Write(json)

	return types.SavedFile{Type: utils.LocalFile, Path: filepath}
}
