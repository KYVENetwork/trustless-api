package helper

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"

	"github.com/KYVENetwork/trustless-api/files"
	"github.com/KYVENetwork/trustless-api/merkle"
	"github.com/KYVENetwork/trustless-api/types"
	"github.com/KYVENetwork/trustless-api/utils"
)

type EVMIndexer struct {
}

func (*EVMIndexer) GetBindings() map[string]types.Endpoint {
	return map[string]types.Endpoint{
		//"/rawValue": {
		//	QueryParameter: []types.ParameterIndex{
		//		{
		//			IndexId:     utils.IndexEVMValue,
		//			Parameter:   []string{"height"},
		//			Description: []string{"EVM block number"},
		//		},
		//	},
		//	Schema: "JsonRPC",
		//},
		"/blockByHash": {
			QueryParameter: []types.ParameterIndex{
				{
					IndexId:     utils.IndexEVMBlock,
					Parameter:   []string{"hash"},
					Description: []string{"hash of a block"},
				},
			},
			Schema: "JsonRPC",
		},
		"/transactionByHash": {
			QueryParameter: []types.ParameterIndex{
				{
					IndexId:     utils.IndexEVMTransaction,
					Parameter:   []string{"hash"},
					Description: []string{"hash of a transaction"},
				},
			},
			Schema: "JsonRPC",
		},
		"/blockReceipts": {
			QueryParameter: []types.ParameterIndex{
				{
					IndexId:     utils.IndexEVMReceipt,
					Parameter:   []string{"hash"},
					Description: []string{"hash of a block"},
				},
			},
			Schema: "JsonRPC",
		},
		//"/logs": {
		//	QueryParameter: []types.ParameterIndex{
		//		{
		//			IndexId:     utils.IndexEVMLog,
		//			Parameter:   []string{"address", "blockHash"},
		//			Description: []string{"contract address or a list of addresses from which logs should originate", "EVM block hash"},
		//		},
		//	},
		//	Schema: "JsonRPC",
		//},
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

func (c *EVMIndexer) calculateMerkleRoot(item interface{}) [32]byte {
	var leafs [][32]byte

	switch v := item.(type) {
	case *[]json.RawMessage:
		leafs = make([][32]byte, 0, len(*v))
		for _, i := range *v {
			leafs = append(leafs, utils.CalculateSHA256Hash(i))
		}
	case *[]Receipt:
		leafs = make([][32]byte, 0, len(*v))
		for _, i := range *v {
			leafs = append(leafs, utils.CalculateSHA256Hash(i))
		}
	default:
		panic("unsupported type")
	}

	return merkle.GetMerkleRoot(leafs)
}

func (c *EVMIndexer) IndexBundle(bundle *types.Bundle) (*[]types.TrustlessDataItem, error) {
	type ProcessedDataItem struct {
		value             EVMDataItem
		key               string
		rawDataItemProof  []types.MerkleNode
		blockProof        []types.MerkleNode
		transactionsProof []types.MerkleNode
		receiptsProof     []types.MerkleNode
		//logsProof         []types.MerkleNode
	}

	leafs := make([][32]byte, 0, len(bundle.DataItems))
	items := make([]ProcessedDataItem, 0, len(bundle.DataItems))

	for _, item := range bundle.DataItems {
		var evmDataItem EVMDataItem
		err := json.Unmarshal(item.Value, &evmDataItem)
		if err != nil {
			return nil, err
		}

		var evmDataItemRaw EVMDataItemRaw
		err = json.Unmarshal(item.Value, &evmDataItemRaw)
		if err != nil {
			return nil, err
		}

		// Verfiy whether the schema is correct
		marshalledEvmDataItem, _ := json.Marshal(evmDataItem)
		if string(marshalledEvmDataItem) != string(item.Value) {
			os.WriteFile("evm_raw_item.json", item.Value, 0666)
			os.WriteFile("evm_unmarshalled_item.json", marshalledEvmDataItem, 0666)
			panic("EVM schema is not correct")
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
		transactionsMerkleRoot := c.calculateMerkleRoot(&evmDataItem.Block.Transactions)
		receiptsHash := utils.CalculateSHA256Hash(&evmDataItem.Receipts)
		logsMerkleRoot := c.calculateMerkleRoot(&flattenedLogs)

		blockMerkleRoot := merkle.GetMerkleRoot([][32]byte{blockHash, transactionsMerkleRoot})
		receiptsLogsMerkleRoot := merkle.GetMerkleRoot([][32]byte{receiptsHash, logsMerkleRoot})
		blockReceiptsRoot := merkle.GetMerkleRoot([][32]byte{blockMerkleRoot, receiptsLogsMerkleRoot})
		subRoot := merkle.GetMerkleRoot([][32]byte{rawDataItemValueHash, blockReceiptsRoot})

		keyBytes := sha256.Sum256([]byte(item.Key))
		combined := append(keyBytes[:], subRoot[:]...)
		merkleRoot := sha256.Sum256(combined)

		leafs = append(leafs, merkleRoot)

		rawDataItemProof := []types.MerkleNode{
			{
				Left: true,
				Hash: hex.EncodeToString(blockReceiptsRoot[:]),
			},
			{
				Left: false,
				Hash: hex.EncodeToString(keyBytes[:]),
			},
		}

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

		//logsProof := []types.MerkleNode{
		//	{
		//		Left: false,
		//		Hash: hex.EncodeToString(receiptsHash[:]),
		//	},
		//	{
		//		Left: false,
		//		Hash: hex.EncodeToString(blockMerkleRoot[:]),
		//	},
		//	{
		//		Left: false,
		//		Hash: hex.EncodeToString(rawDataItemHash[:]),
		//	},
		//	{
		//		Left: false,
		//		Hash: hex.EncodeToString(keyBytes[:]),
		//	},
		//}

		items = append(items, ProcessedDataItem{
			value:             evmDataItem,
			key:               item.Key,
			rawDataItemProof:  rawDataItemProof,
			blockProof:        blockProof,
			transactionsProof: transactionsProof,
			receiptsProof:     receiptsProof,
			//logsProof:         logsProof,
		})
	}

	trustlessItems := make([]types.TrustlessDataItem, 0, len(items)*6)

	for index, item := range items {
		proof, err := merkle.GetHashesCompact(&leafs, index)
		if err != nil {
			return nil, err
		}

		txLeafs := make([][32]byte, 0, len(item.value.Block.Transactions))
		for _, tx := range item.value.Block.Transactions {
			txLeafs = append(txLeafs, utils.CalculateSHA256Hash(tx))
		}

		// Iterate through all transactions and add it to trustless items to serve them individually.
		for txIndex, tx := range item.value.Block.Transactions {
			txProof, err := merkle.GetHashesCompact(&txLeafs, txIndex)
			if err != nil {
				return nil, err
			}

			txProof = append(txProof, item.transactionsProof...)

			encodedProof := utils.EncodeProof(bundle.PoolId, bundle.BundleId, bundle.ChainId, "", "result", append(txProof, proof...))

			rpcResponse, err := utils.WrapIntoJsonRpcResponse(tx)
			if err != nil {
				return nil, err
			}

			var unmarshalledTx Transaction
			if err = json.Unmarshal(tx, &unmarshalledTx); err != nil {
				return nil, err
			}

			trustlessItems = append(trustlessItems, types.TrustlessDataItem{
				PoolId:   bundle.PoolId,
				BundleId: bundle.BundleId,
				ChainId:  bundle.ChainId,
				Value:    rpcResponse,
				Proof:    encodedProof,
				Indices: []types.Index{
					{
						Index:   unmarshalledTx.Hash,
						IndexId: utils.IndexEVMTransaction,
					},
				},
			})
		}

		//var allLogs [][]json.RawMessage
		//for _, receipt := range item.value.Receipts {
		//	allLogs = append(allLogs, receipt.Logs)
		//}

		//// Flatten the array of arrays into a single array
		//var flattenedLogs []json.RawMessage
		//for _, logs := range allLogs {
		//	flattenedLogs = append(flattenedLogs, logs...)
		//}
		//
		//logLeafs := make([][32]byte, 0, len(flattenedLogs))
		//for _, log := range flattenedLogs {
		//	logLeafs = append(logLeafs, utils.CalculateSHA256Hash(log))
		//}
		//
		//for logIndex, log := range flattenedLogs {
		//	logRaw, err := json.Marshal(log)
		//	if err != nil {
		//		return nil, err
		//	}
		//
		//	rpcResponse, err := utils.WrapIntoJsonRpcResponse(json.RawMessage(logRaw))
		//	if err != nil {
		//		return nil, err
		//	}
		//
		//	logProof, err := merkle.GetHashesCompact(&logLeafs, logIndex)
		//	if err != nil {
		//		return nil, err
		//	}
		//
		//	logProof = append(logProof, item.receiptsProof...)
		//
		//	encodedProof := utils.EncodeProof(bundle.PoolId, bundle.BundleId, bundle.ChainId, "", "result", append(logProof, proof...))
		//
		//	var unmarshalledLog Log
		//	if err = json.Unmarshal(logRaw, &unmarshalledLog); err != nil {
		//		return nil, err
		//	}
		//
		//	trustlessItems = append(trustlessItems, types.TrustlessDataItem{
		//		PoolId:   bundle.PoolId,
		//		BundleId: bundle.BundleId,
		//		ChainId:  bundle.ChainId,
		//		Value:    rpcResponse,
		//		Proof:    encodedProof,
		//		Indices: []types.Index{
		//			{
		//				Index:   fmt.Sprintf("%v-%v-%v", unmarshalledLog.BlockHash, unmarshalledLog.TransactionHash, unmarshalledLog.LogIndex),
		//				IndexId: utils.IndexEVMLog,
		//			},
		//		},
		//	})
		//}

		rpcResponse, err := utils.WrapIntoJsonRpcResponse(item.value.Receipts)
		if err != nil {
			return nil, err
		}

		encodedProof := utils.EncodeProof(bundle.PoolId, bundle.BundleId, bundle.ChainId, "", "result", append(item.receiptsProof, proof...))
		trustlessItems = append(trustlessItems, types.TrustlessDataItem{
			Proof: encodedProof,
			Indices: []types.Index{
				{
					Index:   item.value.Block.Hash,
					IndexId: utils.IndexEVMReceipt,
				},
			},
			Value:    rpcResponse,
			PoolId:   bundle.PoolId,
			BundleId: bundle.BundleId,
			ChainId:  bundle.ChainId,
		})

		rpcResponse, err = utils.WrapIntoJsonRpcResponse(item.value.Block)
		if err != nil {
			return nil, err
		}

		encodedProof = utils.EncodeProof(bundle.PoolId, bundle.BundleId, bundle.ChainId, "", "result", append(item.blockProof, proof...))
		trustlessItems = append(trustlessItems, types.TrustlessDataItem{
			PoolId:   bundle.PoolId,
			BundleId: bundle.BundleId,
			ChainId:  bundle.ChainId,
			Value:    rpcResponse,
			Proof:    encodedProof,
			Indices: []types.Index{
				{
					Index:   item.value.Block.Hash,
					IndexId: utils.IndexEVMBlock,
				},
			},
		})
	}

	return &trustlessItems, nil
}

func (*EVMIndexer) GetErrorResponse(message string, data any) any {
	return utils.WrapIntoJsonRpcErrorResponse(message, data)
}

func (d *EVMIndexer) InterceptRequest(get files.Get, indexId int, query []string) (*[]byte, error) {
	return nil, nil	
}
