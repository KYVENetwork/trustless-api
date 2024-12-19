package utils

const (
	ChainIdMainnet  = "kyve-1"
	ChainIdKaon     = "kaon-1"
	ChainIdKorellia = "korellia-2"

	RestEndpointMainnet  = "https://api.kyve.network"
	RestEndpointKaon     = "https://api.kaon.kyve.network"
	RestEndpointKorellia = "https://api.korellia.kyve.network"

	RestEndpointArweave     = "https://arweave.net"
	RestEndpointBundlr      = "https://arweave.net"
	RestEndpointKYVEStorage = "https://storage.kyve.network"
	RestEndpointTurbo       = "https://arweave.net"
)

const (
	IndexBlockHeight            = 0
	IndexSlotNumber             = 1
	IndexBlobByNamespace        = 2
	IndexSharesByNamespace      = 3
	IndexTendermintBlock        = 4
	IndexTendermintBlockResults = 5
	IndexTendermintBlockByHash  = 6
	IndexAllBlobsByNamespace    = 7
	IndexEVMValue               = 8
	IndexEVMBlock               = 9
	IndexEVMTransaction         = 10
	IndexEVMReceipt             = 11
	IndexEVMLog                 = 12
)

const (
	BundlesPageLimit  = 100
	BackoffMaxRetries = 10
)

const (
	DefaultChainId     = ChainIdMainnet
	DefaultRegistryURL = "https://raw.githubusercontent.com/KYVENetwork/source-registry/main/.github/registry.yml"
)
