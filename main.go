package main

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	log "github.com/xlab/suplog"
	"gopkg.in/yaml.v2"
	"io"
	"net/http"
	"os"
	"time"
)

var cfg = Config{}

func init() {
	f, err := os.Open("config.yaml")
	if err != nil {
		panic(err)
	}

	b, err := io.ReadAll(f)
	if err != nil {
		log.Fatalln(err)
	}

	err = yaml.Unmarshal(b, &cfg)
	if err != nil {
		log.Fatalln(err)
	}

}

func main() {

	for k, chain := range cfg.Chains {
		var timeout = chain.Timeout
		if timeout == 0 {
			timeout = DEFAULT_TIMEOUT
		}

		if chain.RPCUrl == "" {
			log.Fatalln("each chain must have rpcURL")
		} else if len(chain.RPCUrl) < 4 || chain.RPCUrl[:4] != "http" {
			log.Fatalln("rpcURL must be formatted as http.")
		}
		client, err := NewHTTPClient(chain.RPCUrl, timeout)
		if err != nil {
			log.Fatalln(err)
		}
		if client == nil {
			log.Fatalln("client shouldn't be <nil>")
		}
		c := cfg.Chains[k]
		c.Client = client
		cfg.Chains[k] = c
	}

	router := gin.Default()
	router.GET("/balances/:chain/:address", getBalances)

	err := router.Run(fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port))
	if err != nil {
		panic(err)
	}

}

type Message struct {
	Error     string      `json:"error"`
	IsSuccess bool        `json:"isSuccess"`
	Content   interface{} `json:"content"`
}

type Balance struct {
	Address  string                        `json:"address"`
	Balances map[BalanceSource]types.Coins `json:"balances"`
}

func getBalances(c *gin.Context) {

	chainParam := c.Param("chain")
	addressParam := c.Param("address")

	startedAt := c.Query("startedAt")
	endedAt := c.Query("endedAt")

	var (
		coins map[BalanceSource]types.Coins
		err   error
	)
	if startedAt == "" || endedAt == "" {
		coins, err = queryEveryBalances(chainParam, addressParam, 0)
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, Message{
				errors.Wrap(err, "failed to query balances").Error(),
				false,
				struct{}{},
			})
			return
		}
	} else {
		parsedStartedAt, err := time.Parse(time.DateOnly, startedAt)
		if err != nil {
			c.IndentedJSON(http.StatusBadRequest, Message{
				errors.Wrap(err, "failed to parse time").Error(),
				false,
				struct{}{},
			})
			return
		}
		parsedEndedAt, err := time.Parse(time.DateOnly, endedAt)
		if err != nil {
			c.IndentedJSON(http.StatusBadRequest, Message{
				errors.Wrap(err, "failed to parse time").Error(),
				false,
				struct{}{},
			})
			return
		}

		latestHeight, err := GetLatestHeight(chainParam)
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, Message{
				errors.Wrap(err, "failed to get latestHeight").Error(),
				false,
				struct{}{},
			})
			return
		}

		// calculate block time
		latestBlockTime, err := GetBlockTime(chainParam, latestHeight)
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, Message{
				errors.Wrap(err, "failed to get block time").Error(),
				false,
				struct{}{},
			})
			return
		}

		secondBlockTime, err := GetBlockTime(chainParam, latestHeight-1)
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, Message{
				errors.Wrap(err, "failed to get block time").Error(),
				false,
				struct{}{},
			})
			return
		}

		expectedBlockInterval := latestBlockTime.Sub(*secondBlockTime)

		endedAtHeight, err := calculateTargetTimeAndHeight(chainParam, parsedEndedAt.Truncate(24*time.Hour), *latestBlockTime, expectedBlockInterval, latestHeight)
		if err != nil {
			c.IndentedJSON(http.StatusInternalServerError, Message{
				errors.Wrap(err, "failed to get block time").Error(),
				false,
				struct{}{},
			})
			return
		}

		r := calculateDailyHeights(chainParam, parsedStartedAt.Truncate(24*time.Hour), parsedEndedAt.Truncate(24*time.Hour), expectedBlockInterval, endedAtHeight)

		var periodCoins = make(map[string]map[BalanceSource]types.Coins)
		for k, v := range r {
			coins, err = queryEveryBalances(chainParam, addressParam, v)
			periodCoins[k.String()] = coins
		}

		c.IndentedJSON(http.StatusOK,
			Message{
				"",
				false,
				periodCoins,
			})
		return
	}

	if err != nil {
		log.Error(err.Error())
	}

	c.IndentedJSON(http.StatusOK,
		Message{
			"",
			false,
			Balance{
				Balances: coins,
				Address:  addressParam,
			},
		})
}
