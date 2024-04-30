package pool

import (
	"fmt"

	"github.com/KYVENetwork/trustless-api/config"
	"github.com/KYVENetwork/trustless-api/types"
	"github.com/KYVENetwork/trustless-api/utils"
	"github.com/tendermint/tendermint/libs/json"
)

func GetPoolInfo(chainId string, poolId int64) (*types.PoolResponse, error) {
	restEndpoints := config.Endpoints.Chains[chainId]
	var data []byte
	var err error
	for _, r := range restEndpoints {
		data, err = utils.GetFromUrlWithBackoff(fmt.Sprintf("%s/kyve/query/v1beta1/pool/%d", r, poolId))
		if err == nil {
			break
		}
	}

	var poolResponse types.PoolResponse

	if err := json.Unmarshal(data, &poolResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pool response: %w", err)
	}

	return &poolResponse, nil
}
