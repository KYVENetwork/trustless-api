package server

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/KYVENetwork/trustless-rpc/collectors/bundles"
	"github.com/KYVENetwork/trustless-rpc/types"
	"github.com/KYVENetwork/trustless-rpc/utils"
	"github.com/gin-gonic/gin"
	"go.eigsys.de/gin-cachecontrol/v2"
	"net/http"
	"strconv"
	"time"
)

var (
	logger = utils.TrustlessRpcLogger("server")
)

type ApiServer struct {
	chainId      string
	restEndpoint string
	storageRest  string
}

// TODO: Replace with Source-Registry integration
var (
	MainnetPoolMap  = make(map[string]int64)
	KaonPoolMap     = make(map[string]int64)
	KorelliaPoolMap = make(map[string]int64)
)

func StartApiServer(chainId, restEndpoint, storageRest string, port string) *ApiServer {
	apiServer := &ApiServer{
		chainId:      chainId,
		restEndpoint: restEndpoint,
		storageRest:  storageRest,
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Define index route
	r.LoadHTMLGlob("templates/*")
	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.tmpl", gin.H{})
	})

	// Enable caching
	r.Use(cachecontrol.New(cachecontrol.Config{
		MaxAge: cachecontrol.Duration(30 * 24 * time.Hour),
	}))

	r.GET("/celestia/GetSharesByNamespace", apiServer.GetSharesByNamespace)
	r.GET("/beacon/blob_sidecars", apiServer.BlobSidecars)

	if err := r.Run(fmt.Sprintf(":%v", port)); err != nil {
		logger.Error().Str("err", err.Error()).Msg("failed to run api server")
	}

	return apiServer
}

func (apiServer *ApiServer) GetSharesByNamespace(c *gin.Context) {
	heightStr := c.Query("height")
	namespace := c.Query("namespace")

	// TODO: Replace with Source-Registry integration
	korelliaPoolMap := map[string]int64{
		"AAAAAAAAAAAAAAAAAAAAAAAAAIZiad33fbxA7Z0=": 93,
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAACAgICAgICAg=": 93,
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAABYTLU4hLOUU=": 93,
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAADBuw7+PjGs8=": 93,
	}

	var poolId int64

	switch apiServer.chainId {
	case utils.ChainIdMainnet:
		id, exists := MainnetPoolMap[namespace]
		if exists {
			poolId = id
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "namespace is not supported yet; please contact the KYVE team",
			})
			return
		}
	case utils.ChainIdKaon:
		id, exists := KaonPoolMap[namespace]
		if exists {
			poolId = id
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "namespace is not supported yet; please contact the KYVE team",
			})
			return
		}
	case utils.ChainIdKorellia:
		id, exists := korelliaPoolMap[namespace]
		if exists {
			poolId = id
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "namespace is not supported yet; please contact the KYVE team",
			})
			return
		}
	}

	height, err := strconv.Atoi(heightStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	compressedBundle, err := bundles.GetBundleByKey(height, apiServer.restEndpoint, poolId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	decompressedBundle, err := bundles.GetDataFromFinalizedBundle(*compressedBundle, apiServer.storageRest)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("failed to decompress bundle: %v", err.Error()),
		})
		return
	}

	// parse bundle
	var bundle types.Bundle

	if err := json.Unmarshal(decompressedBundle, &bundle); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("failed to unmarshal bundle data: %v", err.Error()),
		})
		return
	}

	for _, dataItem := range bundle {
		itemHeight, err := strconv.Atoi(dataItem.Key)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("failed to parse block height from key: %v", err.Error()),
			})
			return
		}

		// skip blocks until we reach start height
		if itemHeight < height {
			continue
		} else if itemHeight == height {
			var shares types.Shares

			if err := json.Unmarshal(dataItem.Value, &shares); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("failed to unmarshal value for key %v: %v", itemHeight, err.Error()),
				})
				return
			}

			for _, share := range shares.SharesByNamespace {
				for key, value := range share {
					if key == namespace {
						c.JSON(http.StatusOK, value)
						return
					}
				}
			}
		}
	}
	c.JSON(http.StatusBadRequest, gin.H{
		"error": fmt.Sprintf("failed to find data item in bundle"),
	})
	return
}

func (apiServer *ApiServer) BlobSidecars(c *gin.Context) {
	heightStr := c.Query("block_height")
	slotStr := c.Query("slot_number")
	chainId := c.Query("l2")

	// TODO: Replace with Source-Registry integration
	KorelliaPoolMap["blobs"] = 95

	// For backwards compatibility; will be removed soon
	if chainId == "arbitrum" {
		KorelliaPoolMap["blobs"] = 86
	}

	var poolId int64

	switch apiServer.chainId {
	case utils.ChainIdMainnet:
		poolId = MainnetPoolMap["blobs"]
	case utils.ChainIdKaon:
		poolId = KaonPoolMap["blobs"]
	case utils.ChainIdKorellia:
		poolId = KorelliaPoolMap["blobs"]
	}

	if poolId == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "query is not supported yet; please contact the KYVE team",
		})
		return
	}

	if heightStr != "" && slotStr != "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "it's not allowed to specify block_height and slot_number",
		})
		return
	}

	if heightStr != "" {
		var bundle *types.Bundle

		height, err := strconv.Atoi(heightStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		bundle = bundles.GetDecompressedBundleByHeight(c, height, apiServer.restEndpoint, apiServer.storageRest, poolId)
		if bundle == nil {
			return
		}

		for _, dataItem := range *bundle {
			itemHeight, err := strconv.Atoi(dataItem.Key)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("failed to parse block height from key: %v", err.Error()),
				})
				return
			}

			// skip blocks until we reach start height
			if itemHeight < height {
				continue
			} else if itemHeight == height {
				c.JSON(http.StatusOK, dataItem.Value)
				return
			}
		}
	} else if slotStr != "" {
		var bundle *types.Bundle

		slot, err := strconv.Atoi(slotStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		bundle = bundles.GetDecompressedBundleBySlot(c, slot, apiServer.restEndpoint, apiServer.storageRest, poolId)
		if bundle == nil {
			return
		}

		for _, dataItem := range *bundle {
			// Parse JSON into RawMessage
			var rawMsg json.RawMessage
			err := json.Unmarshal(dataItem.Value, &rawMsg)
			if err != nil {
				fmt.Println("Error:", err)
				return
			}

			// Create a struct to unmarshal into
			var blobData types.BlobValue

			// Unmarshal the RawMessage into the struct
			err = json.Unmarshal(rawMsg, &blobData)
			if err != nil {
				fmt.Println("Error:", err)
				return
			}

			// skip blocks until we reach start height
			if blobData.SlotNumber < slot {
				continue
			} else if blobData.SlotNumber == slot {
				c.JSON(http.StatusOK, blobData)
				return
			}
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("required to specify block_height or slot_number"),
		})
	}

	c.JSON(http.StatusBadRequest, gin.H{
		"error": fmt.Sprintf("failed to find data item in bundle"),
	})
	return
}
