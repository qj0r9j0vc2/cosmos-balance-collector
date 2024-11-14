package main

import (
	authv1beta1 "cosmossdk.io/api/cosmos/auth/v1beta1"
	bankv1beta1 "cosmossdk.io/api/cosmos/bank/v1beta1"
	distributionv1beta1 "cosmossdk.io/api/cosmos/distribution/v1beta1"
	stakingv1beta1 "cosmossdk.io/api/cosmos/staking/v1beta1"
	"cosmossdk.io/api/tendermint/abci"
	"cosmossdk.io/math"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/pkg/errors"
	log "github.com/xlab/suplog"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"time"
)

//func getBalancesPeriod(c *gin.Context) {
//
//	address := c.Param("address")
//	coins, err := queryEveryBalances(address)
//	if err != nil {
//		log.Error(err.Error())
//	}
//
//	c.IndentedJSON(http.StatusOK,
//		Balance{
//			Balances: coins,
//			Address:  address,
//		})
//}

type Day int

const (
	SUN Day = iota
	MON
	TUE
	WED
	THU
	FRI
	SAT
)

type QueryBalanceFunction func(chain, address string, height int64) (types.Coins, error)

type BalanceSource int

const (
	COSMOSSDK_BANK_BALANCE BalanceSource = iota
	COSMOSSDK_STAKING_DELEGATION
	COSMOSSDK_STAKING_UNBONDING
	COSMOSSDK_DISTRIBUTION_REWARD
	COSMOSSDK_DISTRIBUTION_COMMISSION
	COSMOSSDK_AUTH_VESTING
)

var (
	methods = map[BalanceSource]QueryBalanceFunction{
		COSMOSSDK_BANK_BALANCE:        queryBankAllBalances,
		COSMOSSDK_STAKING_DELEGATION:  queryStakingDelegatorDelegations,
		COSMOSSDK_STAKING_UNBONDING:   queryStakingDelegatorUnbondingDelegations,
		COSMOSSDK_DISTRIBUTION_REWARD: queryDistributionDelegationRewards,
		//queryDistributionValidatorCommission: 4,
		//queryAuthVesting: 5,
	}
)

func queryEveryBalances(chain, address string, height int64) (map[BalanceSource]types.Coins, error) {

	var (
		wg     = sync.WaitGroup{}
		mtx    = sync.Mutex{}
		result = make(map[BalanceSource]types.Coins)
	)
	for source, method := range methods {
		wg.Add(1)
		go func(source BalanceSource, m QueryBalanceFunction) {
			defer wg.Done()

			coins, err := m(chain, address, height)
			if err != nil {
				log.WithFields(log.Fields{
					"func": runtime.FuncForPC(reflect.ValueOf(m).Pointer()).Name(),
				}).Error(err.Error())
			}
			mtx.Lock()
			result[source] = coins
			mtx.Unlock()

		}(source, method)
	}

	wg.Wait()

	return result, nil
}

func queryBankAllBalances(chain, address string, height int64) (types.Coins, error) {

	msg := banktypes.QueryAllBalancesRequest{
		Address: address,
		Pagination: &query.PageRequest{
			Limit:      5, // TODO(fetch until to limit logic needed)
			Offset:     0,
			CountTotal: true,
		},
	}

	b, err := msg.Marshal()
	if err != nil {
		return nil, err
	}

	// Convert to hex string
	hexString := hex.EncodeToString(b)
	log.Debug("Hex-encoded Protobuf data: 0x%s\n", hexString)

	var resp []byte
	if c, exists := cfg.Chains[chain]; exists {
		req := abci.RequestQuery{
			Data:   b,
			Path:   bankv1beta1.Query_AllBalances_FullMethodName,
			Height: height,
		}

		resp, err = c.Client.Query(ABCI_QUERY_PATH, map[string]string{
			"data":   fmt.Sprintf("0x%x", req.Data),
			"path":   fmt.Sprintf("\"%s\"", req.Path),
			"prove":  fmt.Sprintf("%t", req.Prove),
			"height": fmt.Sprintf("%d", req.Height),
		})
		if err != nil {
			return nil, err
		}
	}

	var abciResponse = &ABCIQueryResult{}
	err = json.Unmarshal(resp, abciResponse)
	if err != nil {
		return nil, err
	}

	if abciResponse.Result == nil {
		return nil, errors.New("request didn't complete successfully")
	}

	var allBalancesResponse = &banktypes.QueryAllBalancesResponse{}
	x, err := base64.StdEncoding.DecodeString(abciResponse.Result.Response.Value)

	err = allBalancesResponse.Unmarshal(x)
	if err != nil {
		return nil, err
	}

	var coins types.Coins

	for _, balance := range allBalancesResponse.Balances {
		coins = append(coins, balance)
	}

	return coins, nil
}

func queryStakingDelegatorUnbondingDelegations(chain, address string, height int64) (types.Coins, error) {

	if cfg.Chains[chain].StakingTokenDenom == "" {
		return nil, errors.New("stakingTokenDenom must be set")
	}

	msg := stakingtypes.QueryDelegatorUnbondingDelegationsRequest{
		DelegatorAddr: address,
		Pagination: &query.PageRequest{
			Limit:      5, // TODO(fetch until to limit logic needed)
			Offset:     0,
			CountTotal: true,
		},
	}

	b, err := msg.Marshal()
	if err != nil {
		return nil, err
	}

	// Convert to hex string
	hexString := hex.EncodeToString(b)
	log.Debug("Hex-encoded Protobuf data: 0x%s\n", hexString)

	var resp []byte
	if c, exists := cfg.Chains[chain]; exists {
		req := abci.RequestQuery{
			Data:   b,
			Path:   stakingv1beta1.Query_DelegatorUnbondingDelegations_FullMethodName,
			Height: height,
		}

		resp, err = c.Client.Query(ABCI_QUERY_PATH, map[string]string{
			"data":   fmt.Sprintf("0x%x", req.Data),
			"path":   fmt.Sprintf("\"%s\"", req.Path),
			"prove":  fmt.Sprintf("%t", req.Prove),
			"height": fmt.Sprintf("%d", req.Height),
		})
		if err != nil {
			return nil, err
		}
	}

	var abciResponse = &ABCIQueryResult{}
	err = json.Unmarshal(resp, abciResponse)
	if err != nil {
		return nil, err
	}

	if abciResponse.Result == nil {
		return nil, errors.New("request didn't complete successfully")
	}

	var unbonding = &stakingtypes.QueryDelegatorUnbondingDelegationsResponse{}
	x, err := base64.StdEncoding.DecodeString(abciResponse.Result.Response.Value)

	err = unbonding.Unmarshal(x)
	if err != nil {
		return nil, err
	}

	var coins types.Coins
	for _, u := range unbonding.UnbondingResponses {
		for _, entry := range u.Entries {
			coins = append(coins, types.Coin{
				Denom:  cfg.Chains[chain].StakingTokenDenom,
				Amount: entry.Balance,
			})
		}
	}

	return coins, nil
}

func queryStakingDelegatorDelegations(chain, address string, height int64) (types.Coins, error) {

	msg := stakingtypes.QueryDelegatorDelegationsRequest{
		DelegatorAddr: address,
		Pagination: &query.PageRequest{
			Limit:      5, // TODO(fetch until to limit logic needed)
			Offset:     0,
			CountTotal: true,
		},
	}

	b, err := msg.Marshal()
	if err != nil {
		return nil, err
	}

	// Convert to hex string
	hexString := hex.EncodeToString(b)
	log.Debug("Hex-encoded Protobuf data: 0x%s\n", hexString)

	var resp []byte
	if c, exists := cfg.Chains[chain]; exists {
		req := abci.RequestQuery{
			Data:   b,
			Path:   stakingv1beta1.Query_DelegatorDelegations_FullMethodName,
			Height: height,
		}

		resp, err = c.Client.Query(ABCI_QUERY_PATH, map[string]string{
			"data":   fmt.Sprintf("0x%x", req.Data),
			"path":   fmt.Sprintf("\"%s\"", req.Path),
			"prove":  fmt.Sprintf("%t", req.Prove),
			"height": fmt.Sprintf("%d", req.Height),
		})
		if err != nil {
			return nil, err
		}
	}

	var abciResponse = &ABCIQueryResult{}
	err = json.Unmarshal(resp, abciResponse)
	if err != nil {
		return nil, err
	}

	if abciResponse.Result == nil {
		return nil, errors.New("request didn't complete successfully")
	}

	var delegations = &stakingtypes.QueryDelegatorDelegationsResponse{}
	x, err := base64.StdEncoding.DecodeString(abciResponse.Result.Response.Value)

	err = delegations.Unmarshal(x)
	if err != nil {
		return nil, err
	}

	var coins types.Coins
	for _, delegation := range delegations.DelegationResponses {
		coins = append(coins, delegation.Balance)
	}

	return coins, nil
}

func queryDistributionDelegationRewards(chain, address string, height int64) (types.Coins, error) {

	msg := distributiontypes.QueryDelegationTotalRewardsRequest{
		DelegatorAddress: address,
	}

	b, err := msg.Marshal()
	if err != nil {
		return nil, err
	}
	// Convert to hex string
	hexString := hex.EncodeToString(b)
	log.Debug("Hex-encoded Protobuf data: 0x%s\n", hexString)

	var resp []byte
	if c, exists := cfg.Chains[chain]; exists {
		req := abci.RequestQuery{
			Data:   b,
			Path:   distributionv1beta1.Query_DelegationRewards_FullMethodName,
			Height: height,
		}

		resp, err = c.Client.Query(ABCI_QUERY_PATH, map[string]string{
			"data":   fmt.Sprintf("0x%x", req.Data),
			"path":   fmt.Sprintf("\"%s\"", req.Path),
			"prove":  fmt.Sprintf("%t", req.Prove),
			"height": fmt.Sprintf("%d", req.Height),
		})
		if err != nil {
			return nil, err
		}
	}

	var abciResponse = &ABCIQueryResult{}
	err = json.Unmarshal(resp, abciResponse)
	if err != nil {
		return nil, err
	}

	if abciResponse.Result == nil {
		return nil, errors.New("request didn't complete successfully")
	}

	var rewardResponse = &distributiontypes.QueryDelegationTotalRewardsResponse{}
	x, err := base64.StdEncoding.DecodeString(abciResponse.Result.Response.Value)

	err = rewardResponse.Unmarshal(x)
	if err != nil {
		return nil, err
	}

	var coins = types.Coins{}
	for _, rewards := range rewardResponse.Rewards {
		for _, dcoin := range rewards.Reward {
			coins = append(coins, types.Coin{
				Denom:  dcoin.Denom,
				Amount: math.Int(dcoin.Amount),
			})
		}
	}

	return coins, nil
}

func queryAccountInfo(chain, address string, height int64) (*authtypes.QueryAccountInfoResponse, error) {
	accountInfoReq := authtypes.QueryAccountInfoRequest{
		Address: address,
	}
	accountInfoReqB, err := accountInfoReq.Marshal()
	if err != nil {
		return nil, err
	}

	// Convert to hex string
	hexString := hex.EncodeToString(accountInfoReqB)
	log.Debug("Hex-encoded Protobuf data: 0x%s\n", hexString)

	var resp []byte
	if c, exists := cfg.Chains[chain]; exists {
		req := abci.RequestQuery{
			Data:   accountInfoReqB,
			Path:   distributionv1beta1.Query_ValidatorCommission_FullMethodName,
			Height: height,
		}

		resp, err = c.Client.Query(ABCI_QUERY_PATH, map[string]string{
			"data":   fmt.Sprintf("0x%x", req.Data),
			"path":   fmt.Sprintf("\"%s\"", req.Path),
			"prove":  fmt.Sprintf("%t", req.Prove),
			"height": fmt.Sprintf("%d", req.Height),
		})
		if err != nil {
			return nil, err
		}
	}

	var abciResponse = &ABCIQueryResult{}
	err = json.Unmarshal(resp, abciResponse)
	if err != nil {
		return nil, err
	}

	if abciResponse.Result == nil {
		return nil, errors.New("request didn't complete successfully")
	}

	var accountInfoResponse = &authtypes.QueryAccountInfoResponse{}
	x, err := base64.StdEncoding.DecodeString(abciResponse.Result.Response.Value)

	err = accountInfoResponse.Unmarshal(x)
	if err != nil {
		return nil, err
	}

	return accountInfoResponse, nil
}

// it should be validatorAddress
func queryDistributionValidatorCommission(chain, address string, height int64) (types.Coins, error) {

	msg := distributiontypes.QueryValidatorCommissionRequest{
		ValidatorAddress: address,
	}

	b, err := msg.Marshal()
	if err != nil {
		return nil, err
	}

	// Convert to hex string
	hexString := hex.EncodeToString(b)
	log.Debug("Hex-encoded Protobuf data: 0x%s\n", hexString)

	var resp []byte
	if c, exists := cfg.Chains[chain]; exists {
		req := abci.RequestQuery{
			Data:   b,
			Path:   distributionv1beta1.Query_ValidatorCommission_FullMethodName,
			Height: height,
		}

		resp, err = c.Client.Query(ABCI_QUERY_PATH, map[string]string{
			"data":   fmt.Sprintf("0x%x", req.Data),
			"path":   fmt.Sprintf("\"%s\"", req.Path),
			"prove":  fmt.Sprintf("%t", req.Prove),
			"height": fmt.Sprintf("%d", req.Height),
		})
		if err != nil {
			return nil, err
		}
	}

	var abciResponse = &ABCIQueryResult{}
	err = json.Unmarshal(resp, abciResponse)
	if err != nil {
		return nil, err
	}

	if abciResponse.Result == nil {
		return nil, errors.New("request didn't complete successfully")
	}

	var allBalancesResponse = &distributiontypes.QueryValidatorCommissionResponse{}
	x, err := base64.StdEncoding.DecodeString(abciResponse.Result.Response.Value)

	err = allBalancesResponse.Unmarshal(x)
	if err != nil {
		return nil, err
	}

	var coins = types.Coins{}

	for _, dcoin := range allBalancesResponse.Commission.Commission {
		coins = append(coins, types.Coin{
			Denom:  dcoin.Denom,
			Amount: math.Int(dcoin.Amount),
		})
	}

	return coins, nil
}

func queryAuthVesting(chain, address string, height int64) (*authtypes.QueryAccountResponse, error) {

	msg := authtypes.QueryAccountRequest{
		Address: address,
	}

	b, err := msg.Marshal()
	if err != nil {
		return nil, err
	}

	// Convert to hex string
	hexString := hex.EncodeToString(b)
	log.Debug("Hex-encoded Protobuf data: 0x%s\n", hexString)

	var resp []byte
	if c, exists := cfg.Chains[chain]; exists {
		req := abci.RequestQuery{
			Data:   b,
			Path:   authv1beta1.Query_Accounts_FullMethodName,
			Height: height,
		}

		resp, err = c.Client.Query(ABCI_QUERY_PATH, map[string]string{
			"data":   fmt.Sprintf("0x%x", req.Data),
			"path":   fmt.Sprintf("\"%s\"", req.Path),
			"prove":  fmt.Sprintf("%t", req.Prove),
			"height": fmt.Sprintf("%d", req.Height),
		})
		if err != nil {
			return nil, err
		}
	}

	var abciResponse = &ABCIQueryResult{}
	err = json.Unmarshal(resp, abciResponse)
	if err != nil {
		return nil, err
	}

	if abciResponse.Result == nil {
		return nil, errors.New("request didn't complete successfully")
	}

	var allBalancesResponse = &authtypes.QueryAccountResponse{}
	x, err := base64.StdEncoding.DecodeString(abciResponse.Result.Response.Value)

	err = allBalancesResponse.Unmarshal(x)
	if err != nil {
		return nil, err
	}

	return allBalancesResponse, nil
}

func GetBlockTime(chain string, height int64) (*time.Time, error) {

	var (
		resp []byte
		err  error
	)
	if c, exists := cfg.Chains[chain]; exists {

		resp, err = c.Client.Query(BLOCK_PATH, map[string]string{
			"height": fmt.Sprintf("%d", height),
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to query /block?height=%d", height)
		}
	}

	var r = &BlockResponse{}
	err = json.Unmarshal(resp, r)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal json")
	}

	parsedTime, err := time.Parse(time.RFC3339Nano, r.Result.Block.Header.Time)
	if err != nil {

		return nil, errors.Wrapf(err, "Error parsing time while processing height: %d", height)
	}

	return &parsedTime, nil
}

func GetLatestHeight(chain string) (int64, error) {

	var (
		resp []byte
		err  error
	)
	if c, exists := cfg.Chains[chain]; exists {

		resp, err = c.Client.Query(STATUS_PATH, map[string]string{})
		if err != nil {
			return 0, err
		}
	}

	var r = &StatusResponse{}
	err = json.Unmarshal(resp, r)
	if err != nil {
		return 0, err
	}

	latestHeight, err := strconv.ParseInt(r.Result.SyncInfo.LatestBlockHeight, 0, 64)
	if err != nil {
		return 0, err
	}
	return latestHeight, nil
}
