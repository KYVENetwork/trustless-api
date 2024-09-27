package helper

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/KYVENetwork/trustless-api/merkle"
	"github.com/KYVENetwork/trustless-api/types"
	"github.com/KYVENetwork/trustless-api/types/celestia"
	"github.com/KYVENetwork/trustless-api/utils"
	"google.golang.org/protobuf/proto"
)

type CelestiaIndexer struct{}

func (*CelestiaIndexer) GetBindings() map[string]types.Endpoint {
	return map[string]types.Endpoint{
		"/Get": {
			QueryParameter: []types.ParameterIndex{
				{
					IndexId:     utils.IndexBlobByNamespace,
					Parameter:   []string{"height", "namespace", "commitment"},
					Description: []string{"celestia block height", "celestia share namespace", "blob commitment"},
				},
			},
			Schema: "JsonRPC",
		},
		"/GetAll": {
			QueryParameter: []types.ParameterIndex{
				{
					IndexId:     utils.IndexAllBlobsByNamespace,
					Parameter:   []string{"height", "namespaces"},
					Description: []string{"celestia block height", "celestia share namespaces"},
				},
			},
			Schema: "JsonRPC",
		},
		"/block": {
			QueryParameter: []types.ParameterIndex{
				{
					IndexId:     utils.IndexTendermintBlock,
					Parameter:   []string{"height"},
					Description: []string{"block height"},
				},
			},
			Schema: "TendermintBlock",
		},
	}
}

type CelestiaTendermintItem struct {
	Block struct {
		BlockId json.RawMessage `json:"block_id"`
		Block   struct {
			Header     json.RawMessage `json:"header"`
			Evidence   json.RawMessage `json:"evidence"`
			LastCommit json.RawMessage `json:"last_commit"`
			Data       struct {
				SquareSize string   `json:"square_size"`
				Txs        []string `json:"txs"`
			}
		}
	} `json:"block"`
	BlockResults json.RawMessage `json:"block_results"`
}

func (c *CelestiaIndexer) calculateBlobsMerkleRoot(blobs *[]types.CelestiaBlob) [32]byte {
	leafs := make([][32]byte, 0, len(*blobs))
	for _, blob := range *blobs {
		leafs = append(leafs, utils.CalculateSHA256Hash(blob))
	}
	return merkle.GetMerkleRoot(leafs)
}

func (c *CelestiaIndexer) IndexBundle(bundle *types.Bundle) (*[]types.TrustlessDataItem, error) {
	type ProcessedDataItem struct {
		value                  types.TendermintValue
		blobs                  []types.CelestiaBlob
		key                    string
		localBlockProof        []types.MerkleNode
		localBlockResultsProof []types.MerkleNode
		localBlobsProof        []types.MerkleNode
	}

	leafs := make([][32]byte, 0, len(bundle.DataItems))
	items := make([]ProcessedDataItem, 0, len(bundle.DataItems))

	// first we decode each data item to get its blobs and calculate its dataitem hash
	for _, item := range bundle.DataItems {
		// unmarshal raw tendermint item
		var celestiaItem CelestiaTendermintItem
		err := json.Unmarshal(item.Value, &celestiaItem)
		if err != nil {
			return nil, err
		}

		// we assume there are 4 blobs per block
		blobs := make([]types.CelestiaBlob, 0, 4)
		// iterate over all txs and check if its a BlobTx
		for _, tx := range celestiaItem.Block.Block.Data.Txs {
			blobTx := &celestia.BlobTx{}
			// tx is encoded in base64
			txBytes, err := base64.StdEncoding.DecodeString(tx)
			if err != nil {
				return nil, err
			}
			if err := proto.Unmarshal(txBytes, blobTx); err != nil {
				// not a BlobTx -> no blobs available
				continue
			}

			// extract MsgPayForBlobs to find commitments and other relevant information
			// we have to unmarshal the based sdk.Tx transaction for this

			tendermintTx := &celestia.Tx{}
			if err := proto.Unmarshal(blobTx.Tx, tendermintTx); err != nil {
				return nil, err
			}

			var msgPayForBlobs *celestia.MsgPayForBlobs
			for _, msg := range tendermintTx.Body.Messages {
				// typeUrl for MsgPayForBlobs
				if msg.TypeUrl == "/celestia.blob.v1.MsgPayForBlobs" {
					// initilize pointer
					msgPayForBlobs = &celestia.MsgPayForBlobs{}
					if err := proto.Unmarshal(msg.Value, msgPayForBlobs); err != nil {
						return nil, err
					}
				}
			}

			if msgPayForBlobs == nil {
				return nil, fmt.Errorf("missing MsgPayForBlobs in Tx")
			}

			for index, blob := range blobTx.Blobs {
				blobs = append(blobs, types.CelestiaBlob{
					Namespace:    base64.StdEncoding.EncodeToString(blob.NamespaceId), //TODO: check for formatting -> base64 or not
					Data:         blob.Data,
					ShareVersion: blob.ShareVersion,
					Commitment:   base64.StdEncoding.EncodeToString(msgPayForBlobs.ShareCommitments[index]),
					Index:        -1,
				})
			}
		}

		// create leaf hash for the celestia data item
		var tendermintValue types.TendermintValue
		err = json.Unmarshal(item.Value, &tendermintValue)
		if err != nil {
			return nil, err
		}

		blockHash := utils.CalculateSHA256Hash(tendermintValue.Block)
		blockResultsHash := utils.CalculateSHA256Hash(tendermintValue.BlockResults)

		tendermintMerkleRoot := merkle.GetMerkleRoot([][32]byte{blockHash, blockResultsHash})
		blobsMerkleRoot := c.calculateBlobsMerkleRoot(&blobs)

		merkleRootCombined := merkle.GetMerkleRoot([][32]byte{tendermintMerkleRoot, blobsMerkleRoot})

		keyBytes := sha256.Sum256([]byte(item.Key))
		combined := append(keyBytes[:], merkleRootCombined[:]...)

		leafs = append(leafs, sha256.Sum256(combined))

		blockProof := []types.MerkleNode{
			{
				Left: true,
				Hash: hex.EncodeToString(blockResultsHash[:]),
			},
			{
				Left: true,
				Hash: hex.EncodeToString(blobsMerkleRoot[:]),
			},
		}

		blockResultsProof := []types.MerkleNode{
			{
				Left: false,
				Hash: hex.EncodeToString(blockHash[:]),
			},
			{
				Left: true,
				Hash: hex.EncodeToString(blobsMerkleRoot[:]),
			},
		}

		blobsProof := []types.MerkleNode{
			{
				Left: false,
				Hash: hex.EncodeToString(tendermintMerkleRoot[:]),
			},
		}

		items = append(items, ProcessedDataItem{
			value:                  tendermintValue,
			blobs:                  blobs,
			key:                    item.Key,
			localBlockProof:        blockProof,
			localBlockResultsProof: blockResultsProof,
			localBlobsProof:        blobsProof,
		})
	}

	// assume we have 4 blobs per block + block & block_results
	trustlessItems := make([]types.TrustlessDataItem, 0, len(items)*6)

	for index, item := range items {
		proof, err := merkle.GetHashesCompact(&leafs, index)
		if err != nil {
			return nil, err
		}

		blobLeafs := make([][32]byte, 0, len(item.blobs))
		for _, blob := range item.blobs {
			blobLeafs = append(leafs, utils.CalculateSHA256Hash(blob))
		}

		// first create trustless data items for each blob
		for blobIndex, blob := range item.blobs {

			blobRaw, err := json.Marshal(blob)
			if err != nil {
				return nil, err
			}

			rpcResponse, err := utils.WrapIntoJsonRpcResponse(json.RawMessage(blobRaw))
			if err != nil {
				return nil, err
			}

			blobProof, err := merkle.GetHashesCompact(&blobLeafs, blobIndex)
			if err != nil {
				return nil, err
			}

			encodedProof := utils.EncodeProof(bundle.PoolId, bundle.BundleId, bundle.ChainId, "", "result", append(blobProof, proof...))

			trustlessItems = append(trustlessItems, types.TrustlessDataItem{
				PoolId:   bundle.PoolId,
				BundleId: bundle.BundleId,
				ChainId:  bundle.ChainId,
				Value:    rpcResponse,
				Proof:    encodedProof,
				Indices: []types.Index{
					{
						Index:   fmt.Sprintf("%v-%v-%v", item.key, blob.Namespace, blob.Commitment),
						IndexId: utils.IndexBlobByNamespace,
					},
				},
			})
		}

		rawAllBlobs, err := json.Marshal(item.blobs)
		if err != nil {
			return nil, err
		}

		// create a trustless item for all blobs
		trustlessItems = append(trustlessItems, types.TrustlessDataItem{
			Proof: "",
			Indices: []types.Index{
				{
					Index:   item.key,
					IndexId: utils.IndexAllBlobsByNamespace,
				},
			},
			Value:    rawAllBlobs,
			PoolId:   bundle.PoolId,
			BundleId: bundle.BundleId,
			ChainId:  bundle.ChainId,
		})

		rpcResponse, err := utils.WrapIntoJsonRpcResponse(item.value.Block)
		if err != nil {
			return nil, err
		}

		encodedProof := utils.EncodeProof(bundle.PoolId, bundle.BundleId, bundle.ChainId, "", "result", append(item.localBlockProof, proof...))
		trustlessItems = append(trustlessItems, types.TrustlessDataItem{
			PoolId:   bundle.PoolId,
			BundleId: bundle.BundleId,
			ChainId:  bundle.ChainId,
			Value:    rpcResponse,
			Proof:    encodedProof,
			Indices: []types.Index{
				{
					Index:   item.key,
					IndexId: utils.IndexTendermintBlock,
				},
			},
		})

		rpcResponse, err = utils.WrapIntoJsonRpcResponse(item.value.BlockResults)
		if err != nil {
			return nil, err
		}

		encodedProof = utils.EncodeProof(bundle.PoolId, bundle.BundleId, bundle.ChainId, "", "result", append(item.localBlockResultsProof, proof...))
		trustlessItems = append(trustlessItems, types.TrustlessDataItem{
			PoolId:   bundle.PoolId,
			BundleId: bundle.BundleId,
			ChainId:  bundle.ChainId,
			Value:    rpcResponse,
			Proof:    encodedProof,
			Indices: []types.Index{
				{
					Index:   item.key,
					IndexId: utils.IndexTendermintBlockResults,
				},
			},
		})
	}

	return &trustlessItems, nil
}

func (*CelestiaIndexer) GetErrorResponse(message string, data any) any {
	return utils.WrapIntoJsonRpcErrorResponse(message, data)
}
