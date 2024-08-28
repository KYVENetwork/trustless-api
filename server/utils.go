package server

import (
	"encoding/json"
	"fmt"
	"github.com/KYVENetwork/trustless-api/db"
	"github.com/KYVENetwork/trustless-api/files"
	"github.com/KYVENetwork/trustless-api/types"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"io"
	"net/http"
	"strings"
)

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
	return "", 0, fmt.Errorf("wrong parameter")
}

// generateOpenApi generates the OpenAPI spec as yaml file
func generateOpenApi(pools []ServePool) ([]byte, error) {
	paths := map[string]interface{}{}

	for _, p := range pools {
		adapter := p.Adapter
		adapterIndexer := adapter.GetIndexer()

		for prefix, value := range adapterIndexer.GetBindings() {

			path := map[string]interface{}{}
			path["tags"] = []string{p.Slug}

			var parameters []map[string]interface{}
			for _, param := range value.QueryParameter {
				if len(param.Parameter) != len(param.Description) {
					logger.Error().Msg("parameter and description length mismatch")
					continue
				}

				for i, parameterName := range param.Parameter {
					currentParameter := map[string]interface{}{}
					currentParameter["name"] = parameterName
					currentParameter["in"] = "query"
					currentParameter["description"] = param.Description[i]
					currentParameter["required"] = false
					currentParameter["schema"] = map[string]interface{}{
						"type": "string",
					}
					parameters = append(parameters, currentParameter)
				}
			}

			parameters = append(parameters, map[string]interface{}{
				"name":        "proof",
				"in":          "query",
				"description": "disable KYVE Proof with `false`",
				"required":    false,
				"schema": map[string]interface{}{
					"type": "string",
				},
			})

			path["parameters"] = parameters

			var headers map[string]interface{}

			if p.ProofAttached {
				headers = map[string]interface{}{
					"x-kyve-proof": map[string]interface{}{
						"description": "KYVE Data Item Inclusion Proof Base64 encoded.",
						"schema": map[string]string{
							"type":    "string",
							"example": "AIQAAAA...Jhhf6ut",
						},
					},
				}
			}

			path["responses"] = map[int32]interface{}{
				http.StatusOK: map[string]interface{}{
					"description": "successful operation",
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]string{
								"$ref": fmt.Sprintf("#/components/schemas/%v", value.Schema),
							},
						},
					},
					"headers": headers,
				},
				http.StatusNotFound: map[string]interface{}{
					"description": "not found",
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]string{
								"$ref": "#/components/schemas/Error",
							},
						},
					},
				},
			}
			// because we only support get requests we set the method to get
			paths[fmt.Sprintf("/%v%v", p.Slug, prefix)] = map[string]interface{}{
				"get": path,
			}
		}
	}

	ymlString, err := yaml.Marshal(map[string]interface{}{
		"paths": paths,
	})
	if err != nil {
		logger.Error().Str("err", err.Error()).Msg("failed to marshal paths")
		return nil, err
	}

	return ymlString, nil
}

// getIndex will search the database for the given query and serve the correct data item if one is found
// if the desired data item does not exist it serves an error
//
// `index` - is the name of the index that will be used e. g. block_height
// `indexId` - is the corresponding Id for the key e. g. block_height -> 0
func (apiServer *ApiServer) getIndex(c *gin.Context, adapter db.Adapter, index string, indexId int) {
	file, err := adapter.Get(indexId, index)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
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
