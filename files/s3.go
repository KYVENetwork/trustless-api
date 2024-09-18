package files

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/KYVENetwork/trustless-api/types"
	"github.com/KYVENetwork/trustless-api/utils"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/viper"
)

type S3FileInterface struct {
	client      *s3.Client
	bucket      string
	compression string
}

// Init prepares the session for the s3.Client
func (saveFile *S3FileInterface) Init() {

	gzip := viper.GetString("storage.compression") == "gzip"
	if gzip {
		saveFile.compression = "compress, gzip"
	} else {
		saveFile.compression = ""
	}
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
		config.WithRetryer(func() aws.Retryer {
			return retry.AddWithMaxBackoffDelay(retry.NewStandard(), time.Second*10)
		}),
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

	b, err := json.Marshal(dataitem)
	if err != nil {
		return SavedFile{}, err
	}
	reader := bytes.NewReader(b)

	filename := utils.GetUniqueDataitemName(dataitem)
	filepath := fmt.Sprintf("%v/%v/%v/%v.json", dataitem.ChainId, dataitem.PoolId, dataitem.BundleId, filename)

	_, err = saveFile.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:          aws.String(saveFile.bucket),
		Key:             aws.String(filepath),
		Body:            reader,
		ContentEncoding: aws.String(saveFile.compression),
		ContentType:     aws.String("application/json"), // set content type to application/json
	})

	if err != nil {
		return SavedFile{}, err
	}

	return SavedFile{Type: S3File, Path: filepath}, nil
}
