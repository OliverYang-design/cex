package bnctest

import (
	"github.com/dwdwow/cex"
	"github.com/dwdwow/cex/bnc"
	"github.com/dwdwow/cex/test/cextest"
	"github.com/dwdwow/props"
	"testing"
)

func readApiKey() cex.Api {
	apiKeys := cextest.MustReadApiKey()
	apiKey, ok := apiKeys[cex.BINANCE]
	if !ok {
		panic("no binance api key")
	}
	return apiKey
}

func TestCoinInfo(t *testing.T) {
	apiKey := readApiKey()
	user := bnc.NewUser(apiKey.ApiKey, apiKey.SecretKey)
	_, respData, err := cex.Request(user, bnc.CoinInfoConfig, nil)
	props.PanicIfNotNil(err)
	props.PrintlnIndent(respData)
}

func TestSpotAccount(t *testing.T) {
	apiKey := readApiKey()
	user := bnc.NewUser(apiKey.ApiKey, apiKey.SecretKey)
	_, respData, err := cex.Request(user, bnc.SpotAccountConfig, nil)
	props.PanicIfNotNil(err)
	props.PrintlnIndent(respData)
}

func TestUniversalTransfer(t *testing.T) {
	apiKey := readApiKey()
	user := bnc.NewUser(apiKey.ApiKey, apiKey.SecretKey)
	_, respData, err := cex.Request(user, bnc.UniversalTransferConfig, bnc.UniversalTransferParams{
		Type:       bnc.TranType_MAIN_UMFUTURE,
		Asset:      "USDT",
		Amount:     10,
		FromSymbol: "",
		ToSymbol:   "",
	})
	props.PanicIfNotNil(err)
	props.PrintlnIndent(respData)
}

func TestFlexibleProduct(t *testing.T) {
	apiKey := readApiKey()
	user := bnc.NewUser(apiKey.ApiKey, apiKey.SecretKey)
	_, respData, err := cex.Request(user, bnc.FlexibleProductConfig, bnc.FlexibleProductListParams{
		Asset: "BTC",
	})
	props.PanicIfNotNil(err)
	props.PrintlnIndent(respData)
}

func TestCryptoLoansIncomeHistories(t *testing.T) {
	apiKey := readApiKey()
	user := bnc.NewUser(apiKey.ApiKey, apiKey.SecretKey)
	_, respData, err := cex.Request(user, bnc.CryptoLoansIncomeHistoriesConfig, bnc.CryptoLoansIncomeHistoriesParams{})
	props.PanicIfNotNil(err)
	props.PrintlnIndent(respData)
}

func TestFlexibleBorrow(t *testing.T) {
	apiKey := readApiKey()
	user := bnc.NewUser(apiKey.ApiKey, apiKey.SecretKey)
	_, respData, err := cex.Request(user, bnc.FlexibleBorrowConfig, bnc.FlexibleBorrowParams{
		LoanCoin:         "USDT",
		LoanAmount:       100,
		CollateralCoin:   "ETH",
		CollateralAmount: 0,
	})
	props.PanicIfNotNil(err)
	props.PrintlnIndent(respData)
}

func TestFlexibleOngoingOrders(t *testing.T) {
	apiKey := readApiKey()
	user := bnc.NewUser(apiKey.ApiKey, apiKey.SecretKey)
	_, respData, err := cex.Request(user, bnc.FlexibleOngoingOrdersConfig, bnc.FlexibleOngoingOrdersParams{
		LoanCoin:       "USDT",
		CollateralCoin: "ETH",
		Current:        0,
		Limit:          0,
	})
	props.PanicIfNotNil(err)
	props.PrintlnIndent(respData)
}
