package files

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/KYVENetwork/trustless-api/types"
	"github.com/spf13/viper"
)

type SaveLocalFileInterface struct{}

func (saveFile *SaveLocalFileInterface) Save(dataitem *types.TrustlessDataItem) (SavedFile, error) {

	// unmarshal item
	b, err := json.Marshal(dataitem)

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
	var filepath string
	filename := strings.Join(dataitem.Keys, "-")
	switch viper.GetString("storage.compression") {
	case "gzip":
		var compressed bytes.Buffer
		gz := gzip.NewWriter(&compressed)
		if _, err := gz.Write(b); err != nil {
			return SavedFile{}, err
		}
		if err := gz.Close(); err != nil {
			return SavedFile{}, err
		}
		b = compressed.Bytes()
		filepath = fmt.Sprintf("%v/%v.gz", dir, filename)
	default:
		filepath = fmt.Sprintf("%v/%v.json", dir, filename)
	}

	// create the file
	file, err := os.Create(filepath)
	if err != nil {
		return SavedFile{}, err
	}

	_, err = file.Write(b)
	if err != nil {
		return SavedFile{}, err
	}

	return SavedFile{Type: LocalFile, Path: filepath}, nil
}

// loads a trustless dataitem from the local device storage
// if the file ends with .gz it will automatically be decompressed
func LoadLocalFile(link string) (types.TrustlessDataItem, error) {

	file, err := os.ReadFile(link)
	if err != nil {
		return types.TrustlessDataItem{}, err
	}

	switch filepath.Ext(link) {
	case ".gz":
		var out bytes.Buffer
		r, err := gzip.NewReader(bytes.NewBuffer(file))
		if err != nil {
			return types.TrustlessDataItem{}, err
		}
		defer r.Close()

		if _, err := io.Copy(&out, r); err != nil {
			return types.TrustlessDataItem{}, err
		}

		file = out.Bytes()
	}

	var dataItem types.TrustlessDataItem

	err = json.Unmarshal(file, &dataItem)
	if err != nil {
		return types.TrustlessDataItem{}, err
	}

	return dataItem, nil
}
