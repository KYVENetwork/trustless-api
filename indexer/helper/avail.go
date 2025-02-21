package helper

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/KYVENetwork/trustless-api/files"
	"github.com/KYVENetwork/trustless-api/merkle"
	"github.com/KYVENetwork/trustless-api/types"
	"github.com/KYVENetwork/trustless-api/utils"
	prim "github.com/availproject/avail-go-sdk/primitives"
	jsoniter "github.com/json-iterator/go"

	SDK "github.com/availproject/avail-go-sdk/sdk"
)

type AvailIndexer struct {
}

func (*AvailIndexer) GetBindings() map[string]types.Endpoint {
	return map[string]types.Endpoint{
		"/GetBlock": {
			QueryParameter: []types.ParameterIndex{
				{
					IndexId:     utils.IndexAvailBlock,
					Parameter:   []string{"hash"},
					Description: []string{"block hash"},
				},
			},
			Schema: "JsonRPC",
		},
		"/GetBlockByHeight": {
			QueryParameter: []types.ParameterIndex{
				{
					IndexId:     utils.IndexAvailBlockByHeight,
					Parameter:   []string{"height"},
					Description: []string{"block height"},
				},
			},
			Schema: "JsonRPC",
		},
		"/GetDataSubmissions": {
			QueryParameter: []types.ParameterIndex{
				{
					IndexId:     utils.IndexAvailSubmissions,
					Parameter:   []string{"hash", "app_id"},
					Description: []string{"block hash", "app_id"},
				},
				{
					IndexId:     utils.IndexAvailSubmissions,
					Parameter:   []string{"hash"},
					Description: []string{"block hash"},
				},
			},
			Schema: "JsonRPC",
		},
		"/GetDataSubmissionsByHeight": {
			QueryParameter: []types.ParameterIndex{
				{
					IndexId:     utils.IndexAvailSubmissionsByHeight,
					Parameter:   []string{"height", "app_id"},
					Description: []string{"height", "app_id"},
				},
				{
					IndexId:     utils.IndexAvailSubmissionsByHeight,
					Parameter:   []string{"height"},
					Description: []string{"height"},
				},
			},
			Schema: "JsonRPC",
		},

		"/GetDataSubmission": {
			QueryParameter: []types.ParameterIndex{
				{
					IndexId:     utils.IndexAvailSubmission,
					Parameter:   []string{"extrinsic_hash"},
					Description: []string{"extrinsic_hash"},
				},
				{
					IndexId:     utils.IndexAvailSubmission,
					Parameter:   []string{"extrinsics_id"},
					Description: []string{"extrinsics_id"},
				},
			},
			Schema: "JsonRPC",
		},
	}
}

type AvailItem struct {
	Block struct {
		Header struct {
			ParentHash     string          `json:"parentHash"`
			Number         string          `json:"number"`
			StateRoot      json.RawMessage `json:"stateRoot"`
			ExtrinsicsRoot json.RawMessage `json:"extrinsicsRoot"`
			Digest         json.RawMessage `json:"digest"`
			Extension      json.RawMessage `json:"extension"`
		} `json:"header"`
		Extrinsics []string `json:"extrinsics"`
	} `json:"block"`
	Hash   string `json:"hash"`
	Height string `json:"height"`
}

func (c *AvailIndexer) IndexBundle(bundle *types.Bundle) (*[]types.TrustlessDataItem, error) {
	leafs := merkle.GetBundleHashes(&bundle.DataItems)
	trustlessItems := make([]types.TrustlessDataItem, 0, len(bundle.DataItems))

	for index, item := range bundle.DataItems {
		proof, err := merkle.GetHashesCompact(leafs, index)
		if err != nil {
			return nil, err
		}

		// create trustless api item for the raw block
		var availItem AvailItem
		err = jsoniter.Unmarshal(item.Value, &availItem)
		if err != nil {
			return nil, err
		}
		availItem.Height = item.Key

		indices := []types.Index{
			{
				Index:   availItem.Hash,
				IndexId: utils.IndexAvailBlock,
			},
			{
				Index:   item.Key,
				IndexId: utils.IndexAvailBlockByHeight,
			},
			{
				Index:   availItem.Hash,
				IndexId: utils.IndexAvailSubmissions,
			},
			{
				Index:   item.Key,
				IndexId: utils.IndexAvailSubmissionsByHeight,
			},
		}

		// add indices for all the submissions
		submissions, err := c.getSubmissions(&availItem, SDK.Filter{})
		if err != nil {
			return nil, err
		}

		for _, sub := range submissions {
			indices = append(indices, types.Index{
				Index:   sub.TxHash, // extrinsic hash
				IndexId: utils.IndexAvailSubmission,
			})
			indices = append(indices, types.Index{
				Index:   fmt.Sprintf("%v-%v", item.Key, sub.TxIndex), // extrinsic id = height-tx_id
				IndexId: utils.IndexAvailSubmission,
			})
		}

		encodedProof := utils.EncodeProof(bundle.PoolId, bundle.BundleId, bundle.ChainId, item.Key, "value", proof)

		rawItem, err := jsoniter.Marshal(availItem)
		if err != nil {
			return nil, err
		}

		trustlessItems = append(trustlessItems, types.TrustlessDataItem{
			PoolId:   bundle.PoolId,
			BundleId: bundle.BundleId,
			ChainId:  bundle.ChainId,
			Value:    rawItem,
			Proof:    encodedProof,
			Indices:  indices,
		})
	}

	return &trustlessItems, nil
}

type DataSubmission struct {
	TxHash   string `json:"hash"`
	TxIndex  uint32 `json:"index"`
	Data     []byte `json:"data"`
	TxSigner string `json:"signer"`
	AppId    uint32 `json:"appid"`
}

func (*AvailIndexer) getSubmissions(item *AvailItem, filter SDK.Filter) ([]DataSubmission, error) {
	rpcBlock, err := SDK.NewRPCBlockFromPrimBlock(prim.Block{
		Extrinsics: item.Block.Extrinsics,
	})
	if err != nil {
		return nil, err
	}

	block := SDK.Block{
		Block: rpcBlock,
	}
	rawSubmissions := block.DataSubmissions(filter)
	submissions := make([]DataSubmission, 0, len(rawSubmissions))
	for _, s := range rawSubmissions {
		submissions = append(submissions, DataSubmission{
			TxHash:   s.TxHash.ToHex(),
			TxIndex:  s.TxIndex,
			AppId:    s.AppId,
			TxSigner: s.TxHash.ToHex(),
			Data:     s.Data,
		})
	}
	return submissions, nil
}

func (*AvailIndexer) GetErrorResponse(message string, data any) any {
	return utils.WrapIntoJsonRpcErrorResponse(message, data)
}

func (a *AvailIndexer) InterceptRequest(get files.Get, indexId int, query []string) (*types.InterceptionResponse, error) {
	if len(query) < 1 {
		return nil, fmt.Errorf("query paramter count mismatch")
	}

	// the first argument should always be engouh to fetch the raw avail item
	item, err := get(indexId, query[0])
	if err != nil {
		return nil, err
	}

	bytes, err := item.Resolve()
	if err != nil {
		return nil, err
	}

	rawItem := struct {
		Value AvailItem `json:"value"`
	}{}
	err = jsoniter.Unmarshal(bytes, &rawItem)
	if err != nil {
		return nil, err
	}

	switch indexId {
	case utils.IndexAvailBlock:
		fallthrough
	case utils.IndexAvailBlockByHeight:
		// in this case we just serve the raw avail block
		rpcResponse, err := utils.WrapIntoJsonRpcResponse(rawItem.Value.Block)
		return &types.InterceptionResponse{
			Data:  &rpcResponse,
			Proof: "",
		}, err
	case utils.IndexAvailSubmissions:
		fallthrough
	case utils.IndexAvailSubmissionsByHeight:

		// query all submissions
		if len(query) == 1 {
			submissions, err := a.getSubmissions(&rawItem.Value, SDK.Filter{})
			if err != nil {
				return nil, err
			}
			rpcResponse, err := utils.WrapIntoJsonRpcResponse(submissions)
			return &types.InterceptionResponse{
				Data:  &rpcResponse,
				Proof: "",
			}, err
		}

		// check if the app id is present
		if len(query) != 2 {
			return nil, fmt.Errorf("query paramter count mismatch")
		}
		filter, err := strconv.Atoi(query[1])
		if err != nil {
			return nil, fmt.Errorf("invalid app id: %v", err)
		}

		submissions, err := a.getSubmissions(&rawItem.Value, SDK.Filter{}.WAppId(uint32(filter)))
		if err != nil {
			return nil, err
		}

		rpcResponse, err := utils.WrapIntoJsonRpcResponse(submissions)
		return &types.InterceptionResponse{
			Data:  &rpcResponse,
			Proof: "",
		}, err
	case utils.IndexAvailSubmission:

		if len(query) != 1 {
			return nil, fmt.Errorf("query paramter count mismatch")
		}
		filter := query[0]
		submissions, err := a.getSubmissions(&rawItem.Value, SDK.Filter{})
		if err != nil {
			return nil, err
		}

		filtered := []DataSubmission{}
		for _, s := range submissions {
			if s.TxHash == filter || fmt.Sprintf("%v-%v", rawItem.Value.Height, s.TxIndex) == filter {
				filtered = append(filtered, s)
			}
		}

		rpcResponse, err := utils.WrapIntoJsonRpcResponse(filtered)
		return &types.InterceptionResponse{
			Data:  &rpcResponse,
			Proof: "",
		}, err
	}

	return nil, nil
}
