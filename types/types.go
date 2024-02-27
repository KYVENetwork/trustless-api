package types

import (
	"encoding/json"
)

type HeightResponse struct {
	Result struct {
		Response struct {
			LastBlockHeight string `json:"last_block_height"`
		} `json:"response"`
	} `json:"result"`
}

type PoolResponse = struct {
	Pool struct {
		Id   int64 `json:"id"`
		Data struct {
			Runtime      string `json:"runtime"`
			StartKey     string `json:"start_key"`
			CurrentKey   string `json:"current_key"`
			TotalBundles int64  `json:"total_bundles"`
			Config       string `json:"config"`
		} `json:"data"`
	} `json:"pool"`
}

type DataItem struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

type Bundle = []DataItem

type Pagination struct {
	NextKey []byte `json:"next_key"`
}

type FinalizedBundle struct {
	Id                string `json:"id,omitempty"`
	StorageId         string `json:"storage_id,omitempty"`
	StorageProviderId string `json:"storage_provider_id,omitempty"`
	CompressionId     string `json:"compression_id,omitempty"`
	FromKey           string `json:"from_key,omitempty"`
	ToKey             string `json:"to_key,omitempty"`
	DataHash          string `json:"data_hash,omitempty"`
}

type FinalizedBundlesResponse = struct {
	FinalizedBundles []FinalizedBundle `json:"finalized_bundles"`
	Pagination       Pagination        `json:"pagination"`
}

type FinalizedBundleResponse = struct {
	FinalizedBundle FinalizedBundle `json:"finalized_bundle"`
}

type Networks struct {
	Kaon *NetworkProperties `yaml:"kaon-1,omitempty"`
	Kyve *NetworkProperties `yaml:"kyve-1,omitempty"`
}

type NetworkProperties struct {
	LatestBlockKey *string
	LatestStateKey *string
	BlockStartKey  *string
	StateStartKey  *string
	Integrations   *Integrations   `yaml:"integrations,omitempty"`
	Pools          *[]Pool         `yaml:"pools,omitempty"`
	SourceMetadata *SourceMetadata `yaml:"properties,omitempty"`
}

type Integrations struct {
	KSYNC *KSYNCIntegration `yaml:"ksync,omitempty"`
}

type KSYNCIntegration struct {
	BlockSyncPool *int `yaml:"block-sync-pool"`
	StateSyncPool *int `yaml:"state-sync-pool"`
}

type SourceMetadata struct {
	Title string `yaml:"title"`
}

type Pool struct {
	Id      *int   `yaml:"id"`
	Runtime string `yaml:"runtime"`
}

type Entry struct {
	ConfigVersion *int     `yaml:"config-version"`
	Networks      Networks `yaml:"networks"`
	SourceID      string   `yaml:"source-id"`
}

type SourceRegistry struct {
	Entries map[string]Entry `yaml:",inline"`
}
