package config

import (
	"github.com/KYVENetwork/trustless-api/db"
	"github.com/KYVENetwork/trustless-api/db/adapters"
	"github.com/KYVENetwork/trustless-api/files"
	"github.com/KYVENetwork/trustless-api/indexer"
	"github.com/KYVENetwork/trustless-api/utils"
	"github.com/spf13/viper"
)

type CrawlerConfig struct {
	ChainId     string
	ChainRest   string
	Indexer     string
	PoolId      int64
	StorageRest string
}

var (
	logger = utils.TrustlessApiLogger("Config")
)

var (
	EthBlobsConfig = CrawlerConfig{PoolId: 21, Indexer: "EthBlobs", ChainRest: utils.RestEndpointKaon, StorageRest: utils.RestEndpointKYVEStorage, ChainId: "kaon-1"}
	LineaConfig    = CrawlerConfig{PoolId: 105, Indexer: "Height", ChainRest: utils.RestEndpointKorellia, StorageRest: utils.RestEndpointKYVEStorage, ChainId: "korellia-2"}
)

func loadDefaults() {
	// storage
	viper.SetDefault("storage.type", "local")
	viper.SetDefault("storage.path", "./data")
	viper.SetDefault("storage.cdn", "")
	viper.SetDefault("storage.aws-endpoint", "")
	viper.SetDefault("storage.region", "auto")
	viper.SetDefault("storage.bucketname", "")
	viper.SetDefault("storage.credentials.keyid", "")
	viper.SetDefault("storage.credentials.keysecret", "")

	// database
	viper.SetDefault("database.type", "sqlite")
	viper.SetDefault("database.dbname", "./database.db")
	viper.SetDefault("database.host", "")
	viper.SetDefault("database.user", "")
	viper.SetDefault("database.password", "")
	viper.SetDefault("database.port", 0)

	// server
	viper.SetDefault("server.port", 4242)
	viper.SetDefault("server.redirect", true)

	var pools []CrawlerConfig = []CrawlerConfig{
		EthBlobsConfig,
		LineaConfig,
	}
	viper.SetDefault("crawler.pools", pools)
}

func LoadConfig(configPath string) {

	viper.AutomaticEnv()
	loadDefaults()
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.SetConfigFile(configPath)
	err := viper.ReadInConfig()
	if err != nil {
		logger.Info().Msg("No config found! Will create config with default values!")
		err = viper.WriteConfigAs(configPath)
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to create config file")
			return
		}
	}
}

func GetSaveDataItemAdapter() files.SaveDataItem {
	switch viper.GetString("storage.type") {
	case "local":
		return &files.LocalFileAdapter
	case "s3":
		return &files.S3FileAdapter
	}

	logger.Fatal().Str("type", viper.GetString("storage.type")).Msg("Unkown storage type")
	return nil
}

func GetCrawlerConfig() []CrawlerConfig {
	var config []CrawlerConfig
	err := viper.UnmarshalKey("crawler.pools", &config)
	if err != nil {
		logger.Fatal().Msg("Failed to parse crawler pools")
	}
	return config
}

func GetDatabaseAdapter(saveDataItem files.SaveDataItem, indexer indexer.Indexer, poolId int64) db.Adapter {
	switch viper.GetString("database.type") {
	case "sqlite":
		adapter := adapters.GetSQLite(saveDataItem, indexer, poolId)
		return &adapter
	case "postgres":
		adapter := adapters.GetPostgres(saveDataItem, indexer, poolId)
		return &adapter
	}
	logger.Fatal().Str("type", viper.GetString("database.type")).Msg("Unkown database type")
	return nil
}

func (c CrawlerConfig) GetDatabaseAdapter() db.Adapter {
	var saveFile files.SaveDataItem = GetSaveDataItemAdapter()
	var idx indexer.Indexer = nil
	switch c.Indexer {
	case "EthBlobs":
		idx = &indexer.EthBlobIndexer
	case "Height":
		idx = &indexer.HeightIndexer
	}

	if idx == nil {
		logger.Fatal().Str("type", c.Indexer).Msg("Cannot resolve indexer")
		return nil
	}

	adapter := GetDatabaseAdapter(saveFile, idx, c.PoolId)
	return adapter
}
