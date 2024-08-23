package files

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/KYVENetwork/trustless-api/types"
	"github.com/KYVENetwork/trustless-api/utils"
	"github.com/spf13/viper"
)

type SaveLocalFileInterface struct{}

func (saveFile *SaveLocalFileInterface) Save(dataItem *types.TrustlessDataItem, proofAttached bool) (SavedFile, error) {
	var b []byte
	var err error

	// unmarshal item
	if proofAttached {
		b, err = json.Marshal(dataItem)

		if err != nil {
			return SavedFile{}, err
		}
	} else {
		b, err = json.Marshal(dataItem.ValueWithoutProof)

		if err != nil {
			return SavedFile{}, err
		}
	}

	path := viper.GetString("storage.path")
	dir := fmt.Sprintf("%v/%v/%v", path, dataItem.PoolId, dataItem.BundleId)

	// create directories if we need them
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return SavedFile{}, err
	}
	var filePath string
	filename := utils.GetUniqueDataitemName(dataItem)
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
		filePath = fmt.Sprintf("%v/%v.gz", dir, filename)
	default:
		filePath = fmt.Sprintf("%v/%v.json", dir, filename)
	}

	// create the file
	file, err := os.Create(filePath)
	if err != nil {
		return SavedFile{}, err
	}

	_, err = file.Write(b)
	if err != nil {
		return SavedFile{}, err
	}

	return SavedFile{Type: LocalFile, Path: filePath}, nil
}

// LoadLocalFile loads a trustless data item from the local device storage.
// If the file ends with .gz it will automatically be decompressed.
func LoadLocalFile(link string) (json.RawMessage, error) {

	file, err := os.ReadFile(link)
	if err != nil {
		return json.RawMessage{}, err
	}

	switch filepath.Ext(link) {
	case ".gz":
		var out bytes.Buffer
		r, err := gzip.NewReader(bytes.NewBuffer(file))
		if err != nil {
			return json.RawMessage{}, err
		}
		defer r.Close()

		if _, err := io.Copy(&out, r); err != nil {
			return json.RawMessage{}, err
		}

		file = out.Bytes()
	}
	return file, nil
}
