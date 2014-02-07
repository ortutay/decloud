package info

import (
	"errors"
	"fmt"
	"github.com/conformal/btcjson"
	"github.com/ortutay/decloud/conf"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/util"
)

const (
	SERVICE_NAME = "info"
	PAYMENT_ADDR = "paymentAddr"
)

func NewPaymentAddrReq(currency msg.Currency) *msg.OcReq {
	msg := msg.OcReq{
		NodeId:      []string{},
		Sig:         []string{},
		Nonce:       "",
		Service:     SERVICE_NAME,
		Method:      PAYMENT_ADDR,
		Args:        []string{string(currency)},
		PaymentType: "",
		PaymentTxn:  "",
		Body:        []byte(""),
	}
	return &msg
}

type InfoService struct {
	BitcoindConf *util.BitcoindConf
}

func (is *InfoService) Handle(req *msg.OcReq, policies *[]conf.Policy) (*msg.OcResp, error) {
	methods := make(map[string]func(*msg.OcReq) (*msg.OcResp, error))
	methods[PAYMENT_ADDR] = is.GetPaymentAddr

	if method, ok := methods[req.Method]; ok {
		return method(req)
	} else {
		return msg.NewRespError(msg.METHOD_UNSUPPORTED), nil
	}
}

func (is *InfoService) GetPaymentAddr(req *msg.OcReq) (*msg.OcResp, error) {
	reqCurrency := req.Args[0]
	switch {
	case reqCurrency == string(msg.BTC) && is.BitcoindConf != nil:
		// TODO(ortutay): smarter handling to map request ID to address
		btcAddr, err := is.fetchNewBtcAddr()
		if err != nil {
			return msg.NewRespError(msg.SERVER_ERROR), nil
		}
		payAddr := msg.PaymentAddr{Currency: msg.BTC, Addr: btcAddr}
		return msg.NewRespOk([]byte(payAddr.ToString())), nil
	default:
		return msg.NewRespError(msg.CURRENCY_UNSUPPORTED), nil
	}
}

func (is *InfoService) fetchNewBtcAddr() (string, error) {
	msg, err := btcjson.NewGetNewAddressCmd("")
	if err != nil {
		return "", fmt.Errorf("error while making cmd: %v", err.Error())
	}
	json, err := msg.MarshalJSON()
	if err != nil {
		return "", fmt.Errorf("error while marshaling: %v", err.Error())
	}
	resp, err := btcjson.RpcCommand(
		is.BitcoindConf.User,
		is.BitcoindConf.Password,
		is.BitcoindConf.Server,
		json)
	if err != nil {
		return "", fmt.Errorf("error while making bitcoind JSON-RPC: %v",
			err.Error())
	}
	addr, ok := resp.Result.(string)
	if !ok {
		return "", errors.New("error during bitcoind JSON-RPC")
	}
	return addr, nil
}
