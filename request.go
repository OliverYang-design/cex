package cex

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/go-resty/resty/v2"
)

// =========================================================== \\
// +++                                                     +++ \\
// +++               Cex REST Core: Request                +++ \\
// +++                                                     +++ \\
// =========================================================== \\

// ===========================================================
// Request
// -----------------------------------------------------------

// Request is the core of cex REST.
// This method is convenient for developers and callers.
//
// Developers just need to implement a simple ReqMaker to custom request creating,
// and implement a HTTPStatusCodeChecker and some simple RespBodyUnmarshaler -s
// to custom response analysing.
// After above works, whenever add new REST api, developer could just add
// new ReqConfig, and some very simple REST functions in exchange packages.
// Can see code in bnc(binance) package.
//
// Callers just need to find target ReqConfig in the target exchange package,
// and add opts appending needs, Request will do other things.
//
// If wanting to quickly familiarize with this frame, go directly to cex/bnc/user.go.
//
// Structured request error, RequestError, will help callers to check more
// details of error occurred during resting.
func Request[ReqDataType, RespDataType any](
	reqMaker ReqMaker,
	config ReqConfig[ReqDataType, RespDataType],
	reqData ReqDataType,
	opts ...CltOpt,
) (*resty.Response, RespDataType, RequestError) {
	var resp *resty.Response
	var data RespDataType
	var err RequestError
	for i := 0; i < 3; i++ {
		resp, data, err = request(reqMaker, config, reqData, opts...)
		if err.Is(ErrInvalidTimestamp) {
			continue
		}
		break
	}
	return resp, data, err
}

func request[ReqDataType, RespDataType any](
	reqMaker ReqMaker,
	config ReqConfig[ReqDataType, RespDataType],
	reqData ReqDataType,
	opts ...CltOpt,
) (*resty.Response, RespDataType, RequestError) {
	reqErr := RequestError{ReqBaseConfig: config.ReqBaseConfig}
	var respData RespDataType

	req, err := reqMaker.Make(config.ReqBaseConfig, reqData, opts...)
	if err != nil {
		return nil, respData, *reqErr.SetErr(fmt.Errorf("cex: make request, %w", err))
	}

	// here sets empty url
	// request maker should compose the whole url
	var resp *resty.Response
	switch config.Method {
	case http.MethodGet:
		resp, err = req.Get("")
	case http.MethodPost:
		resp, err = req.Post("")
	case http.MethodPut:
		resp, err = req.Put("")
	case http.MethodDelete:
		resp, err = req.Delete("")
	default:
		return resp, respData, *reqErr.SetErr(fmt.Errorf("cex: http method %v is not supported", config.Method))
	}

	// Ignore resty error, if response is not nil.
	// Resty will return err if status code > 399.
	// But the request with a response status code that bigger than 399
	// may not be failed.
	// ex. For binance, response status code that bigger than 500 means
	// that the status is unknown, and users can ignore.
	// https://binance-docs.github.io/apidocs/spot/en/#general-api-information
	// So if one err is returned, check that resp is nil or not.
	// If resp is nil, return directly.
	// Otherwise, go on.
	if err != nil && resp == nil {
		return resp, respData, *reqErr.SetErr(fmt.Errorf("cex: request err: %w", err))
	}

	if resp == nil {
		// Should not get here.
		// If getting here, err and resp are all nil.
		// Resty may have bugs.
		return resp, respData, *reqErr.SetErr(fmt.Errorf("cex: resp and err are all nil, resty may have bugs"))
	}

	var errResty error
	if err != nil {
		errResty = err
	}

	if config.HTTPStatusCodeChecker == nil {
		return resp, respData, *reqErr.SetErr(fmt.Errorf("cex: config http status code checker is nil"))
	}

	if config.RespBodyUnmarshaler == nil {
		return resp, respData, *reqErr.SetErr(fmt.Errorf("cex: config resp body unmarshaler is nil"))
	}

	errHttp := config.HTTPStatusCodeChecker(resp.StatusCode())

	respData, errBodyUnmarshal := config.RespBodyUnmarshaler(resp.Body())

	if errHttp == nil && errBodyUnmarshal == nil {
		return resp, respData, reqErr
	}

	// some cex may set detailed error msg in body, while request failed
	// so, collect http status and body data together
	if errHttp != nil {
		reqErr.HTTPError = &HTTPError{
			StatusCode: resp.StatusCode(),
			Status:     resp.Status(),
			Err:        errHttp,
		}
	}

	if errBodyUnmarshal != nil {
		reqErr.RespBodyUnmarshalerError = errBodyUnmarshal
	}

	reqErr.Err = fmt.Errorf("cex: request, resty err: %w, http err: %w, body unmarshal err: %w", errResty, errHttp, errBodyUnmarshal)

	return resp, respData, reqErr
}

// -----------------------------------------------------------
// Request
// ===========================================================

// ===========================================================
// Core Types
// -----------------------------------------------------------

// ReqBaseConfig save some read-only info.
// This struct is the real contain of ReqConfig.
type ReqBaseConfig struct {
	// ex. https://www.example.com
	BaseUrl string `json:"baseUrl" bson:"baseUrl"`

	// ex. /path/to/service
	Path string `json:"path" bson:"path"`

	// http method, GET, POST...
	// better to use const method value in http package directly
	Method string `json:"method" bson:"method"`

	// if true, should use api key
	IsUserData bool `json:"isUserData" bson:"isUserData"`

	// one user can rest every UserTimeInterval.
	// unit is millisecond
	UserTimeInterval int64 `json:"userTimeInterval" bson:"userTimeInterval"`

	// one ip can reset every IpTimeInterval
	// unit is millisecond
	IpTimeInterval int64 `json:"ipTimeInterval" bson:"ipTimeInterval"`
}

// HTTPStatusCodeChecker checks HTTP status code.
// If request is failed, return error.
type HTTPStatusCodeChecker func(int) error

// RespBodyUnmarshaler unmarshal HTTP response body.
// Cex may have its own diy error code and msg.
// Generally, these infos are contained in body,
// so should get these infos by unmarshalling.
type RespBodyUnmarshaler[D any] func([]byte) (D, *RespBodyUnmarshalerError)

// NilReqData means that no request data.
// If a ReqConfig ReqDataType is this,
// reqData should be nil.
type NilReqData any

// ReqConfig is wrapper of ReqBaseConfig.
// This struct makes it convenient to call Request.
// ReqDataType and RespDataType are not used in ReqConfig,
// but in practice, it is very useful.
// In practice, we call Request to query cex data,
// but we should know ReqBaseConfig, ReqDataType and RespDataType simultaneously.
// We have many config implementations in all cex packages.
// These config with patterns bind ReqBaseConfig,
// ReqDataType and RespDataType together.
// Set a ReqBaseConfig instance as input of Request,
// all Request patterns are defined.
type ReqConfig[ReqDataType, RespDataType any] struct {
	ReqBaseConfig
	// status code and its status message
	HTTPStatusCodeChecker HTTPStatusCodeChecker
	RespBodyUnmarshaler   RespBodyUnmarshaler[RespDataType]
}

// CltOpt is function option that can custom request.
type CltOpt func(*resty.Client)

// ReqMaker should be implemented in all cex package
type ReqMaker interface {
	Make(config ReqBaseConfig, reqData any, opts ...CltOpt) (*resty.Request, error)
	//HandleResp(*resty.Response, *resty.Request) error
}

// -----------------------------------------------------------
// Core Types
// ===========================================================

// ===========================================================
// Custom Errors
// -----------------------------------------------------------

// RespBodyUnmarshalerError contains cex own diy error code and msg.
// Why should specific this struct? See RespBodyUnmarshaler.
type RespBodyUnmarshalerError struct {
	CexErrCode int    `json:"cexErrCode,omitempty"`
	CexErrMsg  string `json:"cexErrMsg,omitempty"`

	// Err is unmarshal error or cex err.
	Err error `json:"err,omitempty"`
}

func (e *RespBodyUnmarshalerError) Error() string {
	return fmt.Sprintf("code: %v, msg: %v, err: %v", e.CexErrCode, e.CexErrMsg, e.Err)
}

func (e *RespBodyUnmarshalerError) Is(target error) bool {
	return e.Err != nil && errors.Is(e.Err, target)
}

func (e *RespBodyUnmarshalerError) SetErr(err error) *RespBodyUnmarshalerError {
	e.Err = err
	return e
}

// HTTPError contains raw info and cex package custom http error.
type HTTPError struct {
	StatusCode int    `json:"statusCode"`
	Status     string `json:"status"`
	Err        error  `json:"err"`
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf(
		"code: %v, status: %v, httperr: %v",
		e.StatusCode, e.Status, e.Err,
	)
}

func (e *HTTPError) Is(target error) bool {
	return e.Err != nil && errors.Is(e.Err, target)
}

// RequestError
// Structured error info is better.
type RequestError struct {
	ReqBaseConfig            ReqBaseConfig             `json:"reqBaseConfig"`
	HTTPError                *HTTPError                `json:"HTTPError"`
	RespBodyUnmarshalerError *RespBodyUnmarshalerError `json:"respBodyUnmarshalerError"`
	Err                      error                     `json:"err"`
}

func (e *RequestError) Error() string {
	return e.String()
}

func (e *RequestError) String() string {
	return fmt.Sprintf(
		"%v %v%v, %v",
		e.ReqBaseConfig.Method,
		e.ReqBaseConfig.BaseUrl,
		e.ReqBaseConfig.Path,
		e.Err,
	)
}

func (e *RequestError) Is(target error) bool {
	return e.Err != nil && errors.Is(e.Err, target)
}

func (e *RequestError) SetErr(err error) *RequestError {
	e.Err = err
	return e
}

func (e *RequestError) IsNotNil() bool {
	return e.Err != nil
}

func (e *RequestError) IsNil() bool {
	return e.Err == nil
}

// -----------------------------------------------------------
// Custom Errors
// ===========================================================

// ===========================================================
// Resp Data Unmarshaler
// -----------------------------------------------------------

func StdBodyUnmarshaler[D any](data []byte) (D, *RespBodyUnmarshalerError) {
	errUnmar := new(RespBodyUnmarshalerError)
	respData := new(D)

	respType := reflect.TypeOf(respData).Elem()

	var anyRes any

	switch respType.Kind() {
	case reflect.String:
		anyRes = any(string(data))
	case reflect.Slice, reflect.Struct, reflect.Map:
		if err := json.Unmarshal(data, respData); err != nil {
			return *respData, errUnmar.SetErr(fmt.Errorf("%w: unmarshal response body, %w", ErrJsonUnmarshal, err))
		}
		anyRes = any(*respData)
	default:
		return *respData, errUnmar.SetErr(fmt.Errorf("response data type %v is not supported", respType.Kind()))
	}

	res, ok := anyRes.(D)

	if !ok {
		errUnmar.Err = fmt.Errorf("cex: cannot convert to %T", res)
	} else {
		errUnmar = nil
	}

	return res, errUnmar
}

// -----------------------------------------------------------
// Resp Data Unmarshaler
// ===========================================================
