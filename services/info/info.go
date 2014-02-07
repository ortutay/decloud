package info

import (
	"encoding/json"
	"fmt"
	"github.com/ortutay/decloud/msg"
)

const (
	SERVICE_NAME      = "info"
	PAYMENT_ADDR = "paymentAddr"
)

type PaymentAddr struct {
	Currency msg.Currency
	Addr string
}

func (pa *PaymentAddr) ToString() string {
	// TODO(ortutay): figure out real wire format
	b, err := json.Marshal(pa)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func NewPaymentAddr(str string) (*PaymentAddr, error) {
	// TODO(ortutay): figure out real wire format
	var pa PaymentAddr
	err := json.Unmarshal([]byte(str), &pa)
	if err != nil {
		return nil, fmt.Errorf("couldn't create PaymentAddr from %v", str)
	} else {
		return &pa, nil
	}
}

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
}

func (is *InfoService) Handle(req *msg.OcReq) (*msg.OcResp, error) {
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
	case reqCurrency == string(msg.BTC):
		payAddr := PaymentAddr{Currency: msg.BTC, Addr: "TODO: fetch addr"}
		return msg.NewRespOk([]byte(payAddr.ToString())), nil
	default:
		return msg.NewRespError(msg.CURRENCY_UNSUPPORTED), nil
	}
}
