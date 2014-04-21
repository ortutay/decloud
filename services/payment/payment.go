package payment

import (
	"strings"
	"encoding/json"

	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/util"
	"github.com/ortutay/decloud/peer"
	"github.com/ortutay/decloud/conf"
)

const (
	SERVICE_NAME        = "payment"
	PAYMENT_ADDR_METHOD = "addr"
	BALANCE_METHOD = "balance"
)

const ADDRS_PER_ID int = 2

func NewPaymentAddrReq(currency msg.Currency) *msg.OcReq {
	msg := msg.OcReq{
		ID:          "",
		Sig:         "",
		Coins:       []string{},
		CoinSigs:    []string{},
		Nonce:       "",
		Service:     SERVICE_NAME,
		Method:      PAYMENT_ADDR_METHOD,
		Args:        []string{currency.String()},
		PaymentType: "",
		PaymentTxn:  "",
		Body:        []byte(""),
	}
	return &msg
}

type PaymentService struct {
	Conf *conf.Conf
	BitcoindConf *util.BitcoindConf
}

func (ps *PaymentService) Handle(req *msg.OcReq) (*msg.OcResp, error) {
	methods := make(map[string]func(*msg.OcReq) (*msg.OcResp, error))
	methods[PAYMENT_ADDR_METHOD] = ps.getPaymentAddr
	methods[BALANCE_METHOD] = ps.balance

	if method, ok := methods[req.Method]; ok {
		return method(req)
	} else {
		return msg.NewRespError(msg.METHOD_UNSUPPORTED), nil
	}
}

type BalanceResponse struct {
	Balance *msg.PaymentValue `json:"balance"`
	MaxBalance *msg.PaymentValue `json:"maxBalance"`
	Addr string `json:"addr"`
}

func (ps *PaymentService) balance(req *msg.OcReq) (*msg.OcResp, error) {
	p, err := peer.NewPeerFromReq(req, ps.BitcoindConf)
	if err != nil {
		return msg.NewRespError(msg.SERVER_ERROR), nil
	}
	balance, err := p.Balance(0, ps.BitcoindConf)
	if err != nil {
		return msg.NewRespError(msg.SERVER_ERROR), nil
	}
	maxBalanceConf, err := ps.Conf.PolicyForCmd(conf.MAX_BALANCE)
	if err != nil {
		// TODO(ortutay): handle more configuration around max balance
		panic(err)
	}
	maxBalance := maxBalanceConf.Args[0].(*msg.PaymentValue)
	btcAddr, err := p.PaymentAddr(ADDRS_PER_ID, ps.BitcoindConf)
	if err != nil {
		return msg.NewRespError(msg.SERVER_ERROR), nil
	}
	resp := BalanceResponse{
		Balance: balance,
		MaxBalance: maxBalance,
		Addr: btcAddr,
	}
	body, err := json.Marshal(&resp)
	if err != nil {
		return msg.NewRespError(msg.SERVER_ERROR), nil
	}
	return msg.NewRespOk(body), nil
}

func (ps *PaymentService) getPaymentAddr(req *msg.OcReq) (*msg.OcResp, error) {
	if len(req.Args) > 1 {
		return msg.NewRespError(msg.INVALID_ARGUMENTS), nil
	}
	reqCurrency := string(msg.BTC)
	if len(req.Args) == 1 {
		reqCurrency = strings.ToUpper(req.Args[0])
	}
	p, err := peer.NewPeerFromReq(req, ps.BitcoindConf)
	if err != nil {
		return msg.NewRespError(msg.SERVER_ERROR), nil
	}
	switch reqCurrency {
	case string(msg.BTC):
		if ps.BitcoindConf == nil {
			return msg.NewRespError(msg.SERVER_ERROR), nil
		}
		// TODO(ortutay): smarter handling to map request ID to address
		btcAddr, err := p.PaymentAddr(ADDRS_PER_ID, ps.BitcoindConf)
		if err != nil {
			return msg.NewRespError(msg.SERVER_ERROR), nil
		}
		payAddr := msg.PaymentAddr{Currency: msg.BTC, Addr: btcAddr}
		return msg.NewRespOk([]byte(payAddr.String())), nil
	default:
		return msg.NewRespError(msg.CURRENCY_UNSUPPORTED), nil
	}
}
