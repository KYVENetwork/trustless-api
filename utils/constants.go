package utils

const (
	ChainIdMainnet  = "kyve-1"
	ChainIdKaon     = "kaon-1"
	ChainIdKorellia = "korellia-2"

	RestEndpointMainnet  = "https://api-eu-1.kyve.network"
	RestEndpointKaon     = "https://api-eu-1.kaon.kyve.network"
	RestEndpointKorellia = "https://api.korellia.kyve.network"

	RestEndpointArweave     = "https://arweave.net"
	RestEndpointBundlr      = "https://arweave.net"
	RestEndpointKYVEStorage = "https://storage.kyve.network"
)

const (
	BundlesPageLimit  = 100
	BackoffMaxRetries = 10
)

const (
	DefaultChainId     = ChainIdMainnet
	DefaultRegistryURL = "https://raw.githubusercontent.com/KYVENetwork/source-registry/main/.github/registry.yml"
)
