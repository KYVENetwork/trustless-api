package server

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"io"
	"net/http"
	"strings"

	"github.com/KYVENetwork/trustless-api/files"
	"github.com/KYVENetwork/trustless-api/types"

	"github.com/KYVENetwork/trustless-api/config"
	"github.com/KYVENetwork/trustless-api/db"
	"github.com/KYVENetwork/trustless-api/indexer"
	"github.com/KYVENetwork/trustless-api/utils"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

var (
	logger = utils.TrustlessApiLogger("server")
)

//go:embed index.tmpl
var embeddedHTML string

//go:embed openapi.yml
var openapi string

type ApiServer struct {
	blobsAdapter db.Adapter
	lineaAdapter db.Adapter
}

type ServePool struct {
	Slug         string
	Adapter      db.Adapter
	Indexer      indexer.Indexer
	ExcludeProof bool
}

func StartApiServer() *ApiServer {
	var blobsAdapter, lineaAdapter db.Adapter
	port := viper.GetInt("server.port")

	var pools []ServePool
	for _, p := range config.GetPoolsConfig() {
		adapter := p.GetDatabaseAdapter()
		indexer := adapter.GetIndexer()

		serverPool := ServePool{
			Indexer:      indexer,
			Adapter:      adapter,
			Slug:         p.Slug,
			ExcludeProof: p.ExcludeProof,
		}
		pools = append(pools, serverPool)
	}

	openapiPaths, err := generateOpenApi(pools)
	if err != nil {
		logger.Error().Str("err", err.Error()).Msg("failed to generate openapi")
	}

	apiServer := &ApiServer{
		blobsAdapter: blobsAdapter,
		lineaAdapter: lineaAdapter,
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// generate the openapi file
	t, _ := template.New("").Parse(openapi)
	var templateBytes bytes.Buffer
	type OpenApi struct {
		Paths   string
		Version string
	}

	t.Execute(&templateBytes, OpenApi{string(openapiPaths), utils.GetVersion()})
	openapi = templateBytes.String()

	// Replace HTML entity for single quote with actual single quote
	openapi = html.UnescapeString(openapi)

	r.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html", []byte(embeddedHTML))
	})

	// serve the openapi file, this is used by swagger ui to display the api
	r.GET("/openapi.yml", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/yaml", []byte(openapi))
	})

	// Enable caching for successful responses only
	r.Use(func(c *gin.Context) {
		c.Next()
		if c.Writer.Status() == http.StatusOK {
			c.Header("Cache-Control", "max-age=86400") // 24 hours in seconds
		}
	})

	for _, pool := range pools {
		localPool := pool
		for p, endpoint := range localPool.Indexer.GetBindings() {
			path := fmt.Sprintf("%v%v", localPool.Slug, p)
			localEndpoint := endpoint
			r.GET(path, func(ctx *gin.Context) {
				indexString, indexId, err := apiServer.findSelectedParameter(ctx, &localEndpoint.QueryParameter)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, localPool.Indexer.GetErrorResponse("Invalid params", nil))
					return
				}
				apiServer.getIndex(ctx, localPool, indexString, indexId)
			})
		}
	}

	if err := r.Run(fmt.Sprintf(":%v", port)); err != nil {
		logger.Error().Str("err", err.Error()).Msg("failed to run api server")
	}

	return apiServer
}

func (apiServer *ApiServer) findSelectedParameter(c *gin.Context, params *[]types.ParameterIndex) (string, int, error) {
	// iterate over all params
	// select the one where all params have a value set and return the build string from the parameter
	for _, param := range *params {

		var query []string
		for _, parameterName := range param.Parameter {
			if c.Query(parameterName) != "" {
				query = append(query, c.Query(parameterName))
			}
		}

		if len(query) == len(param.Parameter) {
			return strings.Join(query, "-"), param.IndexId, nil
		}
	}

	// no fitting parameter
	return "", 0, fmt.Errorf("invalid params")
}

// getIndex will search the database for the given query and serve the correct data item if one is found
// if the desired data item does not exist it serves an error
//
// `index` - is the name of the index that will be used e. g. block_height
// `indexId` - is the corresponding Id for the key e. g. block_height -> 0
func (apiServer *ApiServer) getIndex(c *gin.Context, pool ServePool, index string, indexId int) {
	file, err := pool.Adapter.Get(indexId, index)
	if err != nil {
		c.JSON(http.StatusNotFound, pool.Indexer.GetErrorResponse("Internal error", err.Error()))
		return
	}
	apiServer.resolveFile(c, file)
}

// resolveFile serves the content of a SavedFile
func (apiServer *ApiServer) resolveFile(c *gin.Context, file files.SavedFile) {

	var rawFile []byte

	switch file.Type {
	case files.LocalFile:
		var err error
		rawFile, err = files.LoadLocalFile(file.Path)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
	case files.S3File:
		url := viper.GetString("storage.cdn")
		res, err := http.Get(fmt.Sprintf("%v%v", url, file.Path))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		defer res.Body.Close()
		rawFile, err = io.ReadAll(res.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to read response body: " + err.Error(),
			})
			return
		}
	}

	apiServer.serveFile(c, rawFile)
}

func (apiServer *ApiServer) serveFile(c *gin.Context, file []byte) {
	var trustlessDataItem types.TrustlessDataItem
	err := json.Unmarshal(file, &trustlessDataItem)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// only send the proof if it is attached
	proofValue, proofParamExists := c.GetQuery("proof")

	if trustlessDataItem.Proof != "" {
		if proofParamExists {
			if proofValue != "false" {
				c.Header("x-kyve-proof", trustlessDataItem.Proof)
			}
		} else {
			c.Header("x-kyve-proof", trustlessDataItem.Proof)
		}
	}
	c.JSON(http.StatusOK, trustlessDataItem.Value)
}
