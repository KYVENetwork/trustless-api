package server

import (
	"fmt"
	"net/http"

	"gopkg.in/yaml.v3"
)

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

			if !p.ExcludeProof {
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
								"$ref": fmt.Sprintf("#/components/schemas/%vError", value.Schema),
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
