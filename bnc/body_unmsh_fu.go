package bnc

import (
	"encoding/json"
	"fmt"

	"github.com/dwdwow/cex"
)

func fuBodyUnmshWrapper[D any](unmarshaler cex.RespBodyUnmarshaler[D]) cex.RespBodyUnmarshaler[D] {
	return func(body []byte) (D, *cex.RespBodyUnmarshalerError) {
		var d D
		if err := fuBodyUnmshCodeMsg(body); err != nil {
			return d, err
		}
		return unmarshaler(body)
	}
}

func fuBodyUnmshCodeMsg(body []byte) *cex.RespBodyUnmarshalerError {
	codeMsg := CodeMsg{}

	_ = json.Unmarshal(body, &codeMsg)

	code := codeMsg.Code
	msg := codeMsg.Msg

	if code == 0 || code == 200 {
		return nil
	}

	if code > 0 {
		// should not get here
		return &cex.RespBodyUnmarshalerError{
			CexErrCode: code,
			CexErrMsg:  msg,
			Err: fmt.Errorf(
				"bnc: %w: code: %d, msg: %s",
				cex.ErrUnexpected, code, msg,
			),
		}
	}

	errCtm := spotCexCustomErrCodes[code]
	//switch errCtm {
	//case ErrFutureNoNeedToChangePositionSide:
	//	return nil
	//default:
	//}
	if errCtm == nil {
		errCtm = fmt.Errorf("%v, %v", code, msg)
	}

	return &cex.RespBodyUnmarshalerError{
		CexErrCode: code,
		CexErrMsg:  msg,
		Err:        fmt.Errorf("bnc: %w", errCtm),
	}
}
