package files

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/KYVENetwork/trustless-rpc/types"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/viper"
)

const (
	LocalFile = iota
	S3File    = iota
)

type SavedFile struct {
	Type int
	Path string
}

var (
	LocalFileAdapter = SaveLocalFileInterface{}
	S3FileAdapter    = S3FileInterface{}
)

type SaveDataItem interface {
	Save(dataitem *types.TrustlessDataItem) (SavedFile, error)
}

type SaveLocalFileInterface struct{}

func (saveFile *SaveLocalFileInterface) Save(dataitem *types.TrustlessDataItem) (SavedFile, error) {

	json, err := json.Marshal(dataitem)

	if err != nil {
		return SavedFile{}, err
	}
	path := viper.GetString("storage.path")
	dir := fmt.Sprintf("%v/%v/%v", path, dataitem.PoolId, dataitem.BundleId)
	err = os.MkdirAll(dir, 0777)
	if err != nil {
		return SavedFile{}, err
	}
	filepath := fmt.Sprintf("%v/%v.json", dir, dataitem.Value.Key)

	file, err := os.Create(filepath)
	if err != nil {
		return SavedFile{}, err
	}
	file.Write(json)

	return SavedFile{Type: LocalFile, Path: filepath}, nil
}

func LoadLocalFile(link string) (types.TrustlessDataItem, error) {
	file, err := os.ReadFile(link)

	if err != nil {
		return types.TrustlessDataItem{}, err
	}

	var dataItem types.TrustlessDataItem

	err = json.Unmarshal(file, &dataItem)
	if err != nil {
		return types.TrustlessDataItem{}, err
	}

	return dataItem, nil
}

type S3FileInterface struct {
	client *s3.Client
	bucket string
}

func (saveFile *S3FileInterface) Init() {

	awsEndpoint := viper.GetString("storage.aws-endpoint")
	accessKeyId := viper.GetString("storage.credentials.keyid")
	accessKeySecret := viper.GetString("storage.credentials.keysecret")
	region := viper.GetString("storage.region")

	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: awsEndpoint,
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyId, accessKeySecret, "")),
		config.WithRegion(region),
	)
	if err != nil {
		log.Fatal(err)
	}

	saveFile.client = s3.NewFromConfig(cfg)
	saveFile.bucket = viper.GetString("storage.bucketname")
}

func (saveFile *S3FileInterface) Save(dataitem *types.TrustlessDataItem) (SavedFile, error) {
	if saveFile.client == nil {
		saveFile.Init()
	}

	json, err := json.Marshal(dataitem)
	if err != nil {
		return SavedFile{}, err
	}
	reader := bytes.NewReader(json)

	filepath := fmt.Sprintf("%v/%v/%v.json", dataitem.PoolId, dataitem.BundleId, dataitem.Value.Key)

	_, err = saveFile.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(saveFile.bucket),
		Key:         aws.String(filepath),
		Body:        reader,
		ContentType: aws.String("application/json"),
	})

	if err != nil {
		return SavedFile{}, err
	}

	return SavedFile{Type: S3File, Path: filepath}, nil
}
