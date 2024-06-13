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
)

const (
	IndexBlockHeight            = 0
	IndexSlotNumber             = 1
	IndexSharesByNamespace      = 3
	IndexTendermintBlock        = 4
	IndexTendermintBlockResults = 5
)

const (
	BundlesPageLimit  = 100
	BackoffMaxRetries = 10
)

const (
	DefaultChainId     = ChainIdMainnet
	DefaultRegistryURL = "https://raw.githubusercontent.com/KYVENetwork/source-registry/main/.github/registry.yml"
)
