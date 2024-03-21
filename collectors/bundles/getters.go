package bundles

import (
	"encoding/json"
	"fmt"
	"github.com/KYVENetwork/trustless-rpc/types"
	"github.com/gin-gonic/gin"
	"net/http"
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
