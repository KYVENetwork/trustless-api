package utils

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"runtime"
	runtimeDebug "runtime/debug"
	"strings"
	"time"

	"github.com/KYVENetwork/trustless-api/types"
)

var (
	logger = TrustlessApiLogger("utils")
)

func GetVersion() string {
	version, ok := runtimeDebug.ReadBuildInfo()
	if !ok {
		panic("failed to get version")
	}

	return strings.TrimSpace(version.Main.Version)
}

// GetFromUrl tries to fetch data from url with a custom User-Agent header
func GetFromUrl(url string) ([]byte, error) {
	// Create a custom http.Client with the desired User-Agent header
	client := &http.Client{}

	// Create a new GET request
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Set the User-Agent header
	version := GetVersion()

	if version != "" {
		if strings.HasPrefix(version, "v") {
			version = strings.TrimPrefix(version, "v")
		}
		request.Header.Set("User-Agent", fmt.Sprintf("trustless-api/%v (%v / %v / %v)", version, runtime.GOOS, runtime.GOARCH, runtime.Version()))
	} else {
		request.Header.Set("User-Agent", fmt.Sprintf("trustless-api/dev (%v / %v / %v)", runtime.GOOS, runtime.GOARCH, runtime.Version()))
	}

	// Perform the request
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("got status code %d != 200", response.StatusCode)
	}

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// GetFromUrlWithBackoff tries to fetch data from url with exponential backoff
func GetFromUrlWithBackoff(url string) (data []byte, err error) {
	for i := 0; i < BackoffMaxRetries; i++ {
		data, err = GetFromUrl(url)
		if err != nil {
			delaySec := math.Pow(2, float64(i))
			delay := time.Duration(delaySec) * time.Second

			logger.Error().Msg(fmt.Sprintf("failed to fetch from url %s, retrying in %d seconds", url, int(delaySec)))
			time.Sleep(delay)

			continue
		}

		// only log success message if there were errors previously
		if i > 0 {
			logger.Info().Msg(fmt.Sprintf("successfully fetch data from url %s", url))
		}
		return
	}

	logger.Error().Msg(fmt.Sprintf("failed to fetch data from url within maximum retry limit of %d", BackoffMaxRetries))
	return
}

func CreateSha256Checksum(input []byte) (hash string) {
	h := sha256.New()
	h.Write(input)
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs)
}

func DecompressGzip(input []byte) ([]byte, error) {
	var out bytes.Buffer
	r, err := gzip.NewReader(bytes.NewBuffer(input))
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(&out, r); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func GetChainRest(chainId, chainRest string) string {
	if chainRest != "" {
		// trim trailing slash
		return strings.TrimSuffix(chainRest, "/")
	}

	// if no custom rest endpoint was given we take it from the chainId
	if chainRest == "" {
		switch chainId {
		case ChainIdMainnet:
			return RestEndpointMainnet
		case ChainIdKaon:
			return RestEndpointKaon
		case ChainIdKorellia:
			return RestEndpointKorellia
		default:
			panic(fmt.Sprintf("flag --chain-id has to be either \"%s\", \"%s\" or \"%s\"", ChainIdMainnet, ChainIdKaon, ChainIdKorellia))
		}
	}

	return ""
}

func CalculateSHA256Hash(obj interface{}) [32]byte {
	// Serialize the object to JSON with keys sorted ascending by default
	serializedObj, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}

	// Calculate the SHA -256 hash
	sha256Hash := sha256.Sum256(serializedObj)

	return sha256Hash
}

func BytesToHex(bytes *[][32]byte) []string {
	var hexArray []string
	for _, b := range *bytes {
		hexArray = append(hexArray, hex.EncodeToString(b[:]))
	}
	return hexArray
}

func GetUniqueDataitemName(item *types.TrustlessDataItem) string {
	var output string

	for _, index := range item.Indices {
		output += fmt.Sprintf("%v-%v", index.Index, index.IndexId)
	}

	return output
}

// EncodeProof encodes the proof of a data item into a byte array
// encoded in big endian
// Structure:
// - 2  bytes: poolId (uint16)
// - 8  bytes: bundleId (uint64)
// - 16 bytes: chainId
// - Array of merkle nodes:
//   - 1 byte:  left (true/false)
//   - 32 bytes: hash (sha256)
func EncodeProof(dataItem *types.TrustlessDataItem) []byte {
	bytes := make([]byte, 26) // 2 + 8 + 16 bytes for poolId, bundleId, and chainId

	binary.BigEndian.PutUint16(bytes[0:2], uint16(dataItem.PoolId))
	binary.BigEndian.PutUint64(bytes[2:10], uint64(dataItem.BundleId))
	copy(bytes[10:26], []byte(dataItem.ChainId))

	for _, merkleNode := range dataItem.Proof {
		if merkleNode.Left {
			bytes = append(bytes, 1)
		} else {
			bytes = append(bytes, 0)
		}
		hashBytes, _ := hex.DecodeString(merkleNode.Hash)
		bytes = append(bytes, hashBytes...)
	}

	return bytes
}

// DecodeProof decodes the proof of a data item from a byte array
// see EncodeProof for more information
func DecodeProof(encodedProof []byte) (*types.TrustlessDataItem, error) {
	if len(encodedProof) < 26 {
		return nil, fmt.Errorf("encoded proof is too short")
	}

	dataItem := &types.TrustlessDataItem{}

	dataItem.PoolId = int64(binary.BigEndian.Uint16(encodedProof[0:2]))
	dataItem.BundleId = int64(binary.BigEndian.Uint64(encodedProof[2:10]))
	// Convert the byte slice to a null-terminated string
	dataItem.ChainId = string(bytes.TrimRight(encodedProof[10:26], "\x00"))

	proofBytes := encodedProof[26:]
	var proof []types.MerkleNode

	for len(proofBytes) >= 33 {
		merkleNode := types.MerkleNode{}
		merkleNode.Left = proofBytes[0] == 1
		merkleNode.Hash = hex.EncodeToString(proofBytes[1:33])
		proof = append(proof, merkleNode)
		proofBytes = proofBytes[33:]
	}

	if len(proofBytes) != 0 {
		return nil, fmt.Errorf("invalid proof encoding")
	}

	dataItem.Proof = proof

	return dataItem, nil
}
