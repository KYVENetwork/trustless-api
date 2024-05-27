package server

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/KYVENetwork/trustless-api/config"
	"github.com/KYVENetwork/trustless-api/db"
	"github.com/KYVENetwork/trustless-api/files"
	"github.com/KYVENetwork/trustless-api/indexer"
	"github.com/KYVENetwork/trustless-api/types"
	"github.com/KYVENetwork/trustless-api/utils"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	cachecontrol "go.eigsys.de/gin-cachecontrol/v2"
)

var (
	logger = utils.TrustlessApiLogger("server")
)

//go:embed index.tmpl
var embeddedHTML string

type ApiServer struct {
	blobsAdapter db.Adapter
	lineaAdapter db.Adapter
	redirect     bool
}

type ServePool struct {
	Slug      string
	Adapter   db.Adapter
	Indexer   indexer.Indexer
	Parameter []string
}

func StartApiServer() *ApiServer {
	var blobsAdapter, lineaAdapter db.Adapter
	port := viper.GetInt("server.port")
	redirect := viper.GetBool("server.redirect")

	var pools []ServePool
	for _, p := range config.GetPoolsConfig() {
		adapter := p.GetDatabaseAdapter()
		indexer := adapter.GetIndexer()

		var parameter []string
		for prefix, value := range indexer.GetBindings() {
			for _, param := range value {
				transformed := []string{}
				for _, queryName := range param.Parameter {
					transformed = append(transformed, fmt.Sprintf("%v=_", queryName))
				}
				query := strings.Join(transformed, "&")
				parameter = append(parameter, fmt.Sprintf("%s%s?%s", p.Slug, prefix, query))
			}
		}

		serverPool := ServePool{
			Indexer:   indexer,
			Adapter:   adapter,
			Slug:      p.Slug,
			Parameter: parameter,
		}
		pools = append(pools, serverPool)
	}

	apiServer := &ApiServer{
		blobsAdapter: blobsAdapter,
		lineaAdapter: lineaAdapter,
		redirect:     redirect,
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	t, _ := template.New("").Parse(embeddedHTML)
	var templateBytes bytes.Buffer
	t.Execute(&templateBytes, pools)
	bytes := templateBytes.Bytes()

	r.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html", bytes)
	})

	// Enable caching
	r.Use(cachecontrol.New(cachecontrol.Config{
		MaxAge: cachecontrol.Duration(24 * time.Hour),
	}))

	for _, pool := range pools {
		paths := pool.Indexer.GetBindings()
		currentAdapter := pool.Adapter
		for p, para := range paths {
			path := fmt.Sprintf("%v%v", pool.Slug, p)
			params := para
			r.GET(path, func(ctx *gin.Context) {
				indexString, indexId, err := apiServer.findSelectedParameter(ctx, &params)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{
						"error": "unkown parameter",
					})
					return
				}
				apiServer.GetIndex(ctx, currentAdapter, indexString, indexId)
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

		query := []string{}
		for _, queryName := range param.Parameter {
			if c.Query(queryName) != "" {
				query = append(query, c.Query(queryName))
			}
		}

		if len(query) == len(param.Parameter) {
			return strings.Join(query, "-"), param.IndexId, nil
		}
	}

	// no fitting parameter
	return "", 0, fmt.Errorf("wrong parameter")
}

// GetIndex will search the database for the given query and serve the correct data item if one is found
// if the desired data item does not exist it serves an error
//
// `index` - is the name of the index that will be used e. g. block_height
// `indexId` - is the corresponding Id for the key e. g. block_height -> 0
func (apiServer *ApiServer) GetIndex(c *gin.Context, adapter db.Adapter, index string, indexId int) {
	file, err := adapter.Get(indexId, index)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	apiServer.resolveFile(c, file)
}

// serves the content of a SavedFile
// will either redirect to the link in the SavedFile or serve it directly
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
