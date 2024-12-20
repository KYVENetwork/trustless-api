package config

import (
	_ "embed"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/KYVENetwork/trustless-api/db"
	"github.com/KYVENetwork/trustless-api/db/adapters"
	"github.com/KYVENetwork/trustless-api/files"
	"github.com/KYVENetwork/trustless-api/indexer"
	"github.com/KYVENetwork/trustless-api/utils"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

type PoolsConfig struct {
	BundleStartId int64
	ChainId       string
	Indexer       string
	PoolId        int64
	Slug          string
	ExcludeProof  bool `json:"excludeProof"`
}

type ConfigEndpoints struct {
	Storage map[int][]string
	Chains  map[string][]string
}

var (
	logger    = utils.TrustlessApiLogger("Config")
	Endpoints = ConfigEndpoints{
		Storage: map[int][]string{
			1: {utils.RestEndpointArweave},
			2: {utils.RestEndpointBundlr},
			3: {utils.RestEndpointKYVEStorage},
			4: {utils.RestEndpointTurbo},
		},
		Chains: map[string][]string{
			utils.ChainIdMainnet:  {utils.RestEndpointMainnet},
			utils.ChainIdKaon:     {utils.RestEndpointKaon},
			utils.ChainIdKorellia: {utils.RestEndpointKorellia},
		},
	}
)

//go:embed config.template.yml
var DefaultTemplate []byte

func LoadDefaults() {
	// log level
	viper.SetDefault("log", "info")

	// RAM
	viper.SetDefault("RAM", uint64(1024))

	viper.SetDefault("crawler.threads", 4)

	// prometheus
	viper.SetDefault("prometheus.enabled", false)
	viper.SetDefault("prometheus.port", 2112)

	// storage
	viper.SetDefault("storage.type", "local")
	viper.SetDefault("storage.path", "./data")
	viper.SetDefault("storage.compression", "gzip")
	viper.SetDefault("storage.cdn", "")
	viper.SetDefault("storage.aws-endpoint", "")
	viper.SetDefault("storage.region", "auto")
	viper.SetDefault("storage.bucketname", "")
	viper.SetDefault("storage.credentials.keyid", "")
	viper.SetDefault("storage.credentials.keysecret", "")
	viper.SetDefault("storage.threads", 8)

	// database
	viper.SetDefault("database.type", "sqlite")
	viper.SetDefault("database.dbname", "./database.db")
	viper.SetDefault("database.host", "")
	viper.SetDefault("database.user", "")
	viper.SetDefault("database.password", "")
	viper.SetDefault("database.port", 0)

	// server
	viper.SetDefault("server.port", 4242)

	var pools []PoolsConfig
	viper.SetDefault("pools", pools)

	viper.SetDefault("endpoints", Endpoints)
}

func LoadConfig(configPath string) {
	viper.AutomaticEnv()
	LoadDefaults()
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.SetConfigFile(configPath)

	// if the config file does not exist yet
	if _, err := os.Stat(configPath); err != nil {
		logger.Info().Str("path", configPath).Msg("no config found! will create one with default values.")

		// first get the config directory and create it if it doesnt exit yet
		dirPath := filepath.Dir(configPath)
		os.MkdirAll(dirPath, os.ModePerm)

		// finally write the embedded template config
		fo, err := os.Create(configPath)
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to create config file")
			return
		}

		fo.Write(DefaultTemplate)
	}

	err := viper.ReadInConfig()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load config.")
	}

	_ = viper.BindEnv("RAM", "RAM")

	_ = viper.BindEnv("database.dbname", "DATABASE_NAME")
	_ = viper.BindEnv("database.user", "DATABASE_USER")
	_ = viper.BindEnv("database.port", "DATABASE_PORT")
	_ = viper.BindEnv("database.host", "DATABASE_HOST")
	_ = viper.BindEnv("database.password", "DATABASE_PASSWORD")

	_ = viper.BindEnv("server.port", "PORT")

	_ = viper.BindEnv("crawler.threads", "CRAWLER_THREADS")

	_ = viper.BindEnv("storage.aws-endpoint", "AWS_ENDPOINT")
	_ = viper.BindEnv("storage.bucketname", "BUCKET_NAME")
	_ = viper.BindEnv("storage.cdn", "CDN")
	_ = viper.BindEnv("storage.credentials.keyid", "ACCESS_KEY_ID")
	_ = viper.BindEnv("storage.credentials.keysecret", "SECRET_ACCESS_KEY")
	_ = viper.BindEnv("storage.threads", "STORAGE_THREADS")

	_ = viper.BindEnv("prometheus.enabled", "PROMETHEUS_ENABLED")
	_ = viper.BindEnv("prometheus.port", "PROMETHEUS_PORT")

	loadEndpoints()
	setLogLevel()

	if viper.GetBool("prometheus.enabled") {
		utils.StartPrometheus(viper.GetString("prometheus.port"))
	}

	// setting memory limit

	max_ram := viper.GetInt64("RAM")
	debug.SetMemoryLimit(max_ram * 1024 * 1024)

}

func setLogLevel() {
	switch viper.GetString("log") {
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warning":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "none":
		zerolog.SetGlobalLevel(zerolog.Disabled)
	}
}

func loadEndpoints() {
	var config ConfigEndpoints
	err := viper.UnmarshalKey("endpoints", &config)
	if err != nil {
		logger.Fatal().Msg("Failed to parse endpoints")
		return
	}
	Endpoints = config
}

// GetSaveDataItemAdapter returns the SaveDataItem interface that is configured in the config file
func GetSaveDataItemAdapter() files.SaveDataItem {
	switch viper.GetString("storage.type") {
	case "local":
		return &files.LocalFileAdapter
	case "s3":
		return &files.S3FileAdapter
	}

	logger.Fatal().Str("type", viper.GetString("storage.type")).Msg("Unknown storage type")
	return nil
}

func GetPoolsConfig() []PoolsConfig {
	var config []PoolsConfig
	err := viper.UnmarshalKey("pools", &config)
	if err != nil || len(config) == 0 {
		logger.Fatal().Msg("Failed to parse pools")
	}
	return config
}

// GetDatabaseAdapter returns the correct db.Adapter that is configured in the config file
func GetDatabaseAdapter(saveDataItem files.SaveDataItem, indexer indexer.Indexer, poolId int64, chainId string) db.Adapter {
	switch viper.GetString("database.type") {
	case "sqlite":
		adapter := adapters.GetSQLite(saveDataItem, indexer, poolId, chainId)
		return &adapter
	case "postgres":
		adapter := adapters.GetPostgres(saveDataItem, indexer, poolId, chainId)
		return &adapter
	}
	logger.Fatal().Str("type", viper.GetString("database.type")).Msg("Unkown database type")
	return nil
}

// GetDatabaseAdapter returns the db.Adapter for each pool config
// as each pool has its own adapter
func (c PoolsConfig) GetDatabaseAdapter() db.Adapter {
	var saveFile files.SaveDataItem = GetSaveDataItemAdapter()
	var idx indexer.Indexer
	switch c.Indexer {
	case "EthBlobs":
		idx = &indexer.EthBlobIndexer
	case "Height":
		idx = &indexer.HeightIndexer
	case "Celestia":
		idx = &indexer.CelestiaIndexer
	case "Tendermint":
		idx = &indexer.TendermintIndexer
	case "EVM":
		idx = &indexer.EVMIndexer
	default:
		logger.Fatal().Str("type", c.Indexer).Msg("failed to resolve indexer")
		return nil
	}

	adapter := GetDatabaseAdapter(saveFile, idx, c.PoolId, c.ChainId)
	return adapter
}
