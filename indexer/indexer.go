package indexer

type Indexer interface {
	// SetupDB sets up the used database to create the mapping and the index.
	SetupDB()

	// CreateKeyBundleMapping creates a key -> bundleId mapping for the given poolIds and chainIds.
	CreateKeyBundleMapping(poolIds []int, chainIds []string)

	// GetBundleIdByKey returns the bundle ID associated with the given key and pool ID.
	GetBundleIdByKey(key int, poolId int) int

	// GetLatestKey returns the latest key associated with the given pool ID.
	GetLatestKey(poolId int) int
}
