# Trustless API

The trustless Celestia and EVM Blobs API, providing validated data through KYVE.

## Build from Source
```bash
git clone https://github.com/KYVENetwork/trustless-api.git

cd trustless-api

make

cp build/trustless-api ~/go/bin/trustless-api
```

## How it works

The Trustless API works in multiple phases. First the `crawler` goes through every bundle, downloads its content and stores every data item together with a proof of inclusion to a given data destination. Finally, the `crawler` creates indices on those data items, to quickly retrieve its origin (poolId, bundleId) and content again. These indices will be stored in the given database.

A user then requests a data item with a key, the Trustless API will then search for the key in the previously created indices of the database and serve the corresponding data item.

These steps are independent on the code level, meaning that it is required to first start a process with the `crawler` in order to correctly serve the crawled data items.

## Crawler

You can start the crawling process with the following command:

```sh
trustless-api crawler
```

## Server

To serve the crawled data items you have to start the process with the following arguments:

```sh
trustless-api start
```

## Config

The following config serves as an example, utilizing a Postgres database and an S3 bucket. You can find the template configuration here: `.config.template.yml`

```yml
chain-id: kaon-1 # the chain-id which is being used, chain endpoint & storage endpoints are based on that
crawler: # the pools that will be crawled when running `crawler`
    pools:
        - poolid: 21        # pool id form the desired pool, depends on the chain-id
          indexer: EthBlobs # what indexer to use, available indexer: EthBlobs
database: # config for the database
    type: postgres      # supported databases: sqlite (default), postgres
    dbname: indexer     # the database name, if you use sqlite this will the the database file. default: ./database.db
    # following attributes are only relevant when using postgres, you don't need them for sqlite
    host: "localhost"
    port: 5432 # IMPORTANT: this is postgres database port, not the port the app will use to serve
    user: "admin"
    password: "root"
server: # configuration when running `start`
    no-cache: false # keep this false on production! If set to true, the server will query the chain data live on request and download & build the relevant data 
    port: 4242 # port of the server
    redirect: false # will redirect to the CDN defined in `storage` if set to false the server will fetch the content on request and serve it directly
storage:
    type: s3 # the type of storage to use. available options: local (default), s3
    path: ./data # only relevant when using local storage
    # S3 configuration
    aws-endpoint: "http://example-bucket.s3-website.us-west-2.amazonaws.com/" # your R2 or AWS endpoint
    bucketname: "example-bucket" # your bucket name
    cdn: "https://example.domain/" # CDN where to fetch the data, default will be the aws-endpoint
    credentials:
        keyid: "<access_key_id>" # your access key id
        keysecret: "<access_key_secret>" #your access key secret
    region: auto # default: auto
```
