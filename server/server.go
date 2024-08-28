package server

import (
	"bytes"
	_ "embed"
	"fmt"
	"html"
	"html/template"
	"net/http"
	"time"

	"github.com/KYVENetwork/trustless-api/config"
	"github.com/KYVENetwork/trustless-api/db"
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
var embeddedHTML string

//go:embed openapi.yml
var openapi string

type ApiServer struct {
	blobsAdapter db.Adapter
	lineaAdapter db.Adapter
}

type ServePool struct {
	Slug          string
	Adapter       db.Adapter
	Indexer       indexer.Indexer
	ProofAttached bool
}

func StartApiServer() *ApiServer {
	var blobsAdapter, lineaAdapter db.Adapter
	port := viper.GetInt("server.port")

	var pools []ServePool
	for _, p := range config.GetPoolsConfig() {
		adapter := p.GetDatabaseAdapter()
		indexer := adapter.GetIndexer()

		serverPool := ServePool{
			Indexer:       indexer,
			Adapter:       adapter,
			Slug:          p.Slug,
			ProofAttached: p.ProofAttached,
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

	// Enable caching
	r.Use(cachecontrol.New(cachecontrol.Config{
		MaxAge: cachecontrol.Duration(24 * time.Hour),
	}))

	for _, pool := range pools {
		paths := pool.Indexer.GetBindings()
		currentAdapter := pool.Adapter
		for p, endpoint := range paths {
			path := fmt.Sprintf("%v%v", pool.Slug, p)
			r.GET(path, func(ctx *gin.Context) {
				indexString, indexId, err := apiServer.findSelectedParameter(ctx, &endpoint.QueryParameter)
				if err != nil {
					ctx.JSON(http.StatusBadRequest, gin.H{
						"error": "unkown parameter",
					})
					return
				}
				apiServer.getIndex(ctx, currentAdapter, indexString, indexId)
			})
		}
	}

	if err := r.Run(fmt.Sprintf(":%v", port)); err != nil {
		logger.Error().Str("err", err.Error()).Msg("failed to run api server")
	}

	return apiServer
}
