package helper

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/KYVENetwork/trustless-api/files"
	"github.com/KYVENetwork/trustless-api/merkle"
	"github.com/KYVENetwork/trustless-api/types"
	"github.com/KYVENetwork/trustless-api/utils"
)

type EVMIndexer struct {
	DefaultIndexer
}

func (*EVMIndexer) GetBindings() map[string]types.Endpoint {
	return map[string]types.Endpoint{
		"/blockByHash": {
			QueryParameter: []types.ParameterIndex{
				{
					IndexId:     utils.IndexEVMBlock,
					Parameter:   []string{"hash"},
					Description: []string{"hash of a block"},
				},
			},
			Schema: "EVMBlock",
		},
		"/transactionByHash": {
			QueryParameter: []types.ParameterIndex{
				{
					IndexId:     utils.IndexEVMTransaction,
					Parameter:   []string{"hash"},
					Description: []string{"hash of a transaction"},
				},
			},
			Schema: "EVMTransaction",
		},
		"/blockReceipts": {
			QueryParameter: []types.ParameterIndex{
				{
					IndexId:     utils.IndexEVMReceipt,
					Parameter:   []string{"hash"},
					Description: []string{"hash of a block"},
				},
			},
			Schema: "EVMBlockReceipts",
		},
	}
}

type EVMDataItem struct {
	Block struct {
		Hash                 string            `json:"hash"`
		ParentHash           string            `json:"parentHash"`
		Number               int64             `json:"number"`
		Timestamp            int64             `json:"timestamp"`
		Nonce                string            `json:"nonce"`
		Difficulty           int64             `json:"difficulty"`
		GasLimit             json.RawMessage   `json:"gasLimit"`
		GasUsed              json.RawMessage   `json:"gasUsed"`
		Miner                string            `json:"miner"`
		ExtraData            string            `json:"extraData"`
		Transactions         []json.RawMessage `json:"transactions"`
		BaseFeePerGas        json.RawMessage   `json:"baseFeePerGas"`
		UnderscoreDifficulty json.RawMessage   `json:"_difficulty"`
	} `json:"block"`
	Receipts []Receipt `json:"receipts"`
}

type EVMDataItemRaw struct {
	Block    json.RawMessage   `json:"block"`
	Receipts []json.RawMessage `json:"receipts"`
}

type Receipt struct {
	Status            string            `json:"status"`
	CumulativeGasUsed string            `json:"cumulativeGasUsed"`
	Logs              []json.RawMessage `json:"logs"`
	LogsBloom         string            `json:"logsBloom"`
	Type              string            `json:"type"`
	TransactionHash   string            `json:"transactionHash"`
	TransactionIndex  string            `json:"transactionIndex"`
	BlockHash         string            `json:"blockHash"`
	BlockNumber       string            `json:"blockNumber"`
	GasUsed           string            `json:"gasUsed"`
	EffectiveGasPrice string            `json:"effectiveGasPrice"`
	From              string            `json:"from"`
	To                *string           `json:"to"`
	ContractAddress   *string           `json:"contractAddress"`
}

type Transaction struct {
	Hash string `json:"hash"`
}

type Log struct {
	Address         string `json:"address"`
	BlockHash       string `json:"blockHash"`
	LogIndex        string `json:"logIndex"`
	TransactionHash string `json:"transactionHash"`
}

type ProcessedDataItem struct {
	Value             EVMDataItem        `json:"value"`
	Key               string             `json:"key"`
	BlockProof        []types.MerkleNode `json:"blockProof"`
	TransactionsProof []types.MerkleNode `json:"transactionsProof"`
	ReceiptsProof     []types.MerkleNode `json:"receiptsProof"`
}

type IntermediateItem struct {
	Item        ProcessedDataItem  `json:"item"`
	BundleProof []types.MerkleNode `json:"bundleProof"`
	BundleId    int64              `json:"bundleId"`
	PoolId      int64              `json:"poolId"`
	ChainId     string             `json:"chainId"`
}

func getMerkleRoot[T any](array *[]T) [32]byte {
	leafs := make([][32]byte, 0, len(*array))

	for _, item := range *array {
		leafs = append(leafs, utils.CalculateSHA256Hash(item))
	}

	return merkle.GetMerkleRoot(leafs)
}

func (c *EVMIndexer) IndexBundle(bundle *types.Bundle) (*[]types.TrustlessDataItem, error) {

	leafs := make([][32]byte, 0, len(bundle.DataItems))
	items := make([]ProcessedDataItem, 0, len(bundle.DataItems))

	for _, item := range bundle.DataItems {
		var evmDataItem EVMDataItem
		err := json.Unmarshal(item.Value, &evmDataItem)
		if err != nil {
			return nil, err
		}

		// Flatten logs array of all receipts into one log array to create a Merkle root
		// for all blobs. This is the requirement to serve certain logs with a proof.
		var allLogs [][]json.RawMessage
		for _, receipt := range evmDataItem.Receipts {
			allLogs = append(allLogs, receipt.Logs)
		}

		var flattenedLogs []json.RawMessage
		for _, logs := range allLogs {
			flattenedLogs = append(flattenedLogs, logs...)
		}

		// Create all required hashes and Merkle roots to construct the data item's Merkle root.
		// A graphic of the Merkle tree can be found here: assets/evm_merkle_root.png
		rawDataItemValueHash := utils.CalculateSHA256Hash(item.Value)
		blockHash := utils.CalculateSHA256Hash(evmDataItem.Block)
		transactionsMerkleRoot := getMerkleRoot(&evmDataItem.Block.Transactions)
		receiptsHash := utils.CalculateSHA256Hash(&evmDataItem.Receipts)
		logsMerkleRoot := getMerkleRoot(&flattenedLogs)

		blockMerkleRoot := merkle.GetMerkleRoot([][32]byte{blockHash, transactionsMerkleRoot})
		receiptsLogsMerkleRoot := merkle.GetMerkleRoot([][32]byte{receiptsHash, logsMerkleRoot})
		blockReceiptsRoot := merkle.GetMerkleRoot([][32]byte{blockMerkleRoot, receiptsLogsMerkleRoot})
		subRoot := merkle.GetMerkleRoot([][32]byte{rawDataItemValueHash, blockReceiptsRoot})

		keyBytes := sha256.Sum256([]byte(item.Key))
		combined := append(keyBytes[:], subRoot[:]...)
		merkleRoot := sha256.Sum256(combined)

		leafs = append(leafs, merkleRoot)

		blockProof := []types.MerkleNode{
			{
				Left: true,
				Hash: hex.EncodeToString(transactionsMerkleRoot[:]),
			},
			{
				Left: true,
				Hash: hex.EncodeToString(receiptsLogsMerkleRoot[:]),
			},
			{
				Left: false,
				Hash: hex.EncodeToString(rawDataItemValueHash[:]),
			},
			{
				Left: false,
				Hash: hex.EncodeToString(keyBytes[:]),
			},
		}

		transactionsProof := []types.MerkleNode{
			{
				Left: false,
				Hash: hex.EncodeToString(blockHash[:]),
			},
			{
				Left: true,
				Hash: hex.EncodeToString(receiptsLogsMerkleRoot[:]),
			},
			{
				Left: false,
				Hash: hex.EncodeToString(rawDataItemValueHash[:]),
			},
			{
				Left: false,
				Hash: hex.EncodeToString(keyBytes[:]),
			},
		}

		receiptsProof := []types.MerkleNode{
			{
				Left: true,
				Hash: hex.EncodeToString(logsMerkleRoot[:]),
			},
			{
				Left: false,
				Hash: hex.EncodeToString(blockMerkleRoot[:]),
			},
			{
				Left: false,
				Hash: hex.EncodeToString(rawDataItemValueHash[:]),
			},
			{
				Left: false,
				Hash: hex.EncodeToString(keyBytes[:]),
			},
		}

		items = append(items, ProcessedDataItem{
			Value:             evmDataItem,
			Key:               item.Key,
			BlockProof:        blockProof,
			TransactionsProof: transactionsProof,
			ReceiptsProof:     receiptsProof,
		})
	}

	trustlessItems := make([]types.TrustlessDataItem, 0, len(items)*6)

	for index, item := range items {

		proof, err := merkle.GetHashesCompact(&leafs, index)
		if err != nil {
			return nil, err
		}

		intermediateItem := IntermediateItem{
			Item:        item,
			BundleProof: proof,
			BundleId:    bundle.BundleId,
			PoolId:      bundle.PoolId,
			ChainId:     bundle.ChainId,
		}

		rawItem, err := json.Marshal(intermediateItem)

		if err != nil {
			return nil, err
		}

		indices := []types.Index{
			{
				Index:   item.Value.Block.Hash,
				IndexId: utils.IndexEVMBlock,
			},
			{
				Index:   item.Value.Block.Hash,
				IndexId: utils.IndexEVMReceipt,
			},
		}

		for _, tx := range item.Value.Block.Transactions {
			var unmarshalledTx Transaction
			if err = json.Unmarshal(tx, &unmarshalledTx); err != nil {
				return nil, err
			}

			indices = append(indices, types.Index{
				Index:   unmarshalledTx.Hash,
				IndexId: utils.IndexEVMTransaction,
			})
		}

		trustlessItems = append(trustlessItems, types.TrustlessDataItem{
			PoolId:   bundle.PoolId,
			BundleId: bundle.BundleId,
			ChainId:  bundle.ChainId,
			Value:    rawItem,
			Proof:    "", // derive the proof from the raw item on interception
			Indices:  indices,
		})
	}

	return &trustlessItems, nil
}

func (*EVMIndexer) GetErrorResponse(message string, data any) any {
	return utils.WrapIntoJsonRpcErrorResponse(message, data)
}

func (*EVMIndexer) serveTransactions(intermediateItem *IntermediateItem, query []string) (*types.InterceptionResponse, error) {

	hash := query[0]
	item := intermediateItem.Item

	txLeafs := make([][32]byte, 0, len(item.Value.Block.Transactions))
	for _, tx := range item.Value.Block.Transactions {
		txLeafs = append(txLeafs, utils.CalculateSHA256Hash(tx))
	}

	// Iterate through all transactions and add it to trustless items to serve them individually.
	for txIndex, tx := range item.Value.Block.Transactions {

		var unmarshalledTx Transaction
		if err := json.Unmarshal(tx, &unmarshalledTx); err != nil {
			return nil, err
		}

		if unmarshalledTx.Hash != hash {
			continue
		}

		txProof, err := merkle.GetHashesCompact(&txLeafs, txIndex)
		if err != nil {
			return nil, err
		}

		txProof = append(txProof, item.TransactionsProof...)

		encodedProof := utils.EncodeProof(intermediateItem.PoolId, intermediateItem.BundleId, intermediateItem.ChainId, "", "result", append(txProof, intermediateItem.BundleProof...))

		rpcResponse, err := utils.WrapIntoJsonRpcResponse(tx)
		if err != nil {
			return nil, err
		}

		return &types.InterceptionResponse{
			Data:  &rpcResponse,
			Proof: encodedProof,
		}, nil
	}

	return nil, fmt.Errorf("transaction not found")
}

func (e *EVMIndexer) InterceptRequest(get files.Get, indexId int, query []string) (*types.InterceptionResponse, error) {
	if len(query) != 1 {
		return nil, fmt.Errorf("query paramter count mismatch")
	}

	item, err := get(indexId, query[0])
	if err != nil {
		return nil, err
	}

	bytes, err := item.Resolve()
	if err != nil {
		return nil, err
	}

	intermediateItem := struct {
		Value IntermediateItem `json:"value"`
	}{}
	err = json.Unmarshal(bytes, &intermediateItem)
	if err != nil {
		return nil, err
	}

	rawItem := intermediateItem.Value

	switch indexId {
	case utils.IndexEVMTransaction:
		return e.serveTransactions(&rawItem, query)
	case utils.IndexEVMReceipt:
		rpcResponse, err := utils.WrapIntoJsonRpcResponse(rawItem.Item.Value.Receipts)
		encodedProof := utils.EncodeProof(rawItem.PoolId, rawItem.BundleId, rawItem.ChainId, "", "result", append(rawItem.Item.ReceiptsProof, rawItem.BundleProof...))
		return &types.InterceptionResponse{
			Data:  &rpcResponse,
			Proof: encodedProof,
		}, err
	case utils.IndexEVMBlock:
		rpcResponse, err := utils.WrapIntoJsonRpcResponse(rawItem.Item.Value.Block)
		encodedProof := utils.EncodeProof(rawItem.PoolId, rawItem.BundleId, rawItem.ChainId, "", "result", append(rawItem.Item.BlockProof, rawItem.BundleProof...))
		return &types.InterceptionResponse{
			Data:  &rpcResponse,
			Proof: encodedProof,
		}, err
	}

	return nil, nil
}
