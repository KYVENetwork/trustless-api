package server

import (
	_ "embed"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/KYVENetwork/trustless-api/config"
	"github.com/KYVENetwork/trustless-api/db"
	"github.com/KYVENetwork/trustless-api/files"
	"github.com/KYVENetwork/trustless-api/indexer"
	"github.com/KYVENetwork/trustless-api/utils"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	cachecontrol "go.eigsys.de/gin-cachecontrol/v2"
)

var (
	logger = utils.TrustlessApiLogger("server")
)

//go:embed index.tmpl
var embeddedHTML []byte

type ApiServer struct {
	blobsAdapter db.Adapter
	lineaAdapter db.Adapter
	redirect     bool
}

func StartApiServer() *ApiServer {
	var blobsAdapter, lineaAdapter db.Adapter
	port := viper.GetInt("server.port")
	redirect := viper.GetBool("server.redirect")

	blobsAdapter = config.GetDatabaseAdapter(nil, &indexer.EthBlobIndexer, 21)
	lineaAdapter = config.GetDatabaseAdapter(nil, &indexer.HeightIndexer, 108)

	apiServer := &ApiServer{
		blobsAdapter: blobsAdapter,
		lineaAdapter: lineaAdapter,
		redirect:     redirect,
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html", embeddedHTML)
	})

	// Enable caching
	r.Use(cachecontrol.New(cachecontrol.Config{
		MaxAge: cachecontrol.Duration(30 * 24 * time.Hour),
	}))

	r.GET("/celestia/GetSharesByNamespace", apiServer.GetSharesByNamespace)
	r.GET("/beacon/blob_sidecars", apiServer.BlobSidecars)
	r.GET("/linea", apiServer.LineaHeight)

	if err := r.Run(fmt.Sprintf(":%v", port)); err != nil {
		logger.Error().Str("err", err.Error()).Msg("failed to run api server")
	}

	return apiServer
}

func (apiServer *ApiServer) GetSharesByNamespace(c *gin.Context) {
	heightStr := c.Query("height")

	height, err := strconv.Atoi(heightStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	file, err := apiServer.lineaAdapter.Get(int64(height), indexer.HeightIndexHeight)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	apiServer.resolveFile(c, file)
}

func (apiServer *ApiServer) LineaHeight(c *gin.Context) {
	heightStr := c.Query("block_height")
	height, err := strconv.Atoi(heightStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	file, err := apiServer.lineaAdapter.Get(int64(height), indexer.HeightIndexHeight)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	apiServer.resolveFile(c, file)
}

func (apiServer *ApiServer) BlobSidecars(c *gin.Context) {
	heightStr := c.Query("block_height")
	slotStr := c.Query("slot_number")

	if heightStr != "" && slotStr != "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "it's not allowed to specify block_height and slot_number",
		})
		return
	}

	if heightStr != "" {
		height, err := strconv.Atoi(heightStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		file, err := apiServer.blobsAdapter.Get(int64(height), indexer.EthBlobIndexHeight)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		apiServer.resolveFile(c, file)
		return

	} else if slotStr != "" {
		slot, err := strconv.Atoi(slotStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		file, err := apiServer.blobsAdapter.Get(int64(slot), indexer.EthBlobIndexSlot)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		apiServer.resolveFile(c, file)
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{
		"error": "required to specify block_height or slot_number",
	})
}

func (apiServer *ApiServer) resolveFile(c *gin.Context, file files.SavedFile) {
	switch file.Type {
	case files.LocalFile:
		file, err := files.LoadLocalFile(file.Path)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, file)
	case files.S3File:
		url := viper.GetString("storage.cdn")
		if apiServer.redirect {
			c.Redirect(301, fmt.Sprintf("%v%v", url, file.Path))
		} else {
			res, err := http.Get(fmt.Sprintf("%v%v", url, file.Path))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}
			defer res.Body.Close()
			c.DataFromReader(200, res.ContentLength, "application/json", res.Body, nil)
		}
	}
}
