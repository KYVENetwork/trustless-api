package bundles

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/KYVENetwork/trustless-api/types"
	"github.com/gin-gonic/gin"
)

func GetDecompressedBundleByHeight(c *gin.Context, height int, restEndpoint string, storageRest string, poolId int64) *types.Bundle {
	compressedBundle, err := GetBundleByKey(height, restEndpoint, poolId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return nil
	}

	decompressedBundle, err :=
		GetDataFromFinalizedBundle(*compressedBundle, storageRest)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("failed to decompress bundle: %v", err.Error()),
		})
		return nil
	}

	// parse bundle
	var bundle types.Bundle

	if err := json.Unmarshal(decompressedBundle, &bundle); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("failed to unmarshal bundle data: %v", err.Error()),
		})
		return nil
	}

	return &bundle

}

func GetDecompressedBundleBySlot(c *gin.Context, slot int, restEndpoint string, storageRest string, poolId int64) *types.Bundle {
	compressedBundle, err := GetBundleBySlot(slot, restEndpoint, poolId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return nil
	}

	decompressedBundle, err := GetDataFromFinalizedBundle(*compressedBundle, storageRest)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("failed to decompress bundle: %v", err.Error()),
		})
		return nil
	}

	// parse bundle
	var bundle types.Bundle

	if err := json.Unmarshal(decompressedBundle, &bundle); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("failed to unmarshal bundle data: %v", err.Error()),
		})
		return nil
	}

	return &bundle
}

func GetDecompressedBundle(finalizedBundle types.FinalizedBundle, storageRest string) (types.Bundle, error) {

	decompressedBundle, err := GetDataFromFinalizedBundle(finalizedBundle, storageRest)
	if err != nil {
		return nil, err
	}

	var bundle types.Bundle
	if err := json.Unmarshal(decompressedBundle, &bundle); err != nil {
		return nil, err
	}

	return bundle, nil
}
