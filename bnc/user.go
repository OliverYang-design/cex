package bnc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/dwdwow/cex"
	"github.com/dwdwow/s2m"
	"github.com/go-resty/resty/v2"
)

type User struct {
	api cex.Api
}

func NewUser(apiKey, secretKey string) User {
	return User{
		api: cex.Api{
			Cex:       cex.BINANCE,
			ApiKey:    apiKey,
			SecretKey: secretKey,
		},
	}
}

// ============================== requester start ==============================

func (u User) Make(config cex.ReqBaseConfig, reqData any, opts ...cex.ReqOpt) (*resty.Request, error) {
	if config.IsUserData {
		return u.makePrivateReq(config, reqData, opts...)
	} else {
		return u.makePublicReq(config, reqData, opts...)
	}
}

func (u User) makePublicReq(config cex.ReqBaseConfig, reqData any, opts ...cex.ReqOpt) (*resty.Request, error) {
	m, err := s2m.ToStrMap(reqData)
	if err != nil {
		return nil, fmt.Errorf("bnc: make public request, %w", err)
	}
	val := url.Values{}
	for k, v := range m {
		val.Set(k, v)
	}
	clt := resty.New().
		SetBaseURL(config.BaseUrl)
	req := clt.R().
		SetQueryString(val.Encode())
	for _, opt := range opts {
		opt(clt, req)
	}
	return nil, nil
}

func (u User) makePrivateReq(config cex.ReqBaseConfig, reqData any, opts ...cex.ReqOpt) (*resty.Request, error) {
	query, err := u.sign(reqData)
	if err != nil {
		return nil, err
	}
	// must compose url by self
	// url.Values composing is alphabetical
	// but binance require signature as the last one
	clt := resty.New().
		SetHeader("X-MBX-APIKEY", u.api.ApiKey).
		SetBaseURL(config.BaseUrl + config.Path + "?" + query)
	req := clt.R()
	for _, opt := range opts {
		opt(clt, req)
	}
	return req, nil
}

func (u User) HandleResp(resp *resty.Response, req *resty.Request) error {
	if resp == nil {
		return errors.New("bnc: response checker: response is nil")
	}

	// check http code
	httpCode := resp.StatusCode()
	if httpCode != 200 {
		cexStdErr := HTTPStatusCodeChecker(httpCode)
		if cexStdErr != nil {
			return fmt.Errorf("bnc: http code: %v, status: %v, err: %w", httpCode, resp.Status(), cexStdErr)
		}
	}

	// check binance error code
	body := resp.Body()
	codeMsg := new(CodeMsg)
	if err := json.Unmarshal(body, codeMsg); err != nil {
		// nil err means body is not CodeMsg
		return nil
	}
	if codeMsg.Code >= 0 {
		return nil
	}
	return fmt.Errorf("bnc: msg: %v, code: %v", codeMsg.Msg, codeMsg.Code)
}

// ============================== requester end ==============================

// ============================== sign start ==============================

func (u User) sign(data any) (query string, err error) {
	return signReqData(data, u.api.SecretKey)
}

func signReqData(data any, key string) (query string, err error) {
	m, err := s2m.ToStrMap(data)
	if err != nil {
		err = fmt.Errorf("%w: %w", cex.ErrS2M, err)
		return
	}
	val := url.Values{
		"timestamp": []string{strconv.FormatInt(time.Now().UnixMilli(), 10)},
	}
	for k, v := range m {
		val.Set(k, v)
	}
	query = val.Encode()
	sig := cex.SignByHmacSHA256ToHex(query, key)
	query += "&signature=" + sig
	return
}

// ============================== sign end ==============================
