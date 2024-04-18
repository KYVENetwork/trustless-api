package config

import (
	"fmt"

	"github.com/KYVENetwork/trustless-rpc/db"
	"github.com/KYVENetwork/trustless-rpc/db/adapters"
	"github.com/KYVENetwork/trustless-rpc/files"
	"github.com/KYVENetwork/trustless-rpc/indexer"
	"github.com/KYVENetwork/trustless-rpc/utils"
	"github.com/spf13/viper"
)

type CrawlerConfig struct {
	PoolId  int64
	Indexer string
}

var (
	logger = utils.TrustlessRpcLogger("Config")
)

var (
	EthBlobsConfig = CrawlerConfig{PoolId: 21, Indexer: "EthBlobs"}
)

func loadDefaults() {
	// storage
	viper.SetDefault("storage.type", "local")
	viper.SetDefault("storage.path", "./data")
	viper.SetDefault("storage.cdn", "")
	viper.SetDefault("storage.aws-endpoint", "")
	viper.SetDefault("storage.region", "auto")
	viper.SetDefault("storage.bucketname", "trustless-cache")
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
	viper.SetDefault("server.no-cache", false)
	viper.SetDefault("server.port", 4242)
	viper.SetDefault("server.redirect", true)

	var pools []CrawlerConfig = []CrawlerConfig{
		EthBlobsConfig,
	}
	viper.SetDefault("crawler.pools", pools)
}

func LoadConfig() {

	viper.AutomaticEnv()
	loadDefaults()
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
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
	switch c.Indexer {
	case "EthBlobs":
		adapter := GetDatabaseAdapter(GetSaveDataItemAdapter(), &indexer.EthBlobIndexer, c.PoolId)
		return adapter
	}
	logger.Fatal().Str("type", c.Indexer).Msg("Cannot resolve indexer")

	return nil
}
