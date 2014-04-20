package payment

import (
	"fmt"
	"strings"

	"github.com/conformal/btcjson"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/util"
	"github.com/ortutay/decloud/peer"
)

const (
	SERVICE_NAME        = "payment"
	TXID_METHOD         = "txid"
	PAYMENT_ADDR_METHOD = "addr"
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

// TODO(ortutay): sign these reqs with an input to the txn to prove ownership
func NewBtcTxidReq(txid msg.BtcTxid) *msg.OcReq {
	msg := msg.OcReq{
		ID:          "",
		Sig:         "",
		Coins:       []string{},
		CoinSigs:    []string{},
		Nonce:       "",
		Service:     SERVICE_NAME,
		Method:      TXID_METHOD,
		Args:        []string{txid.String(), string(msg.BTC)},
		PaymentType: msg.TXID,
		Body:        []byte(""),
	}
	return &msg
}

type PaymentService struct {
	BitcoindConf *util.BitcoindConf
}

func (ps *PaymentService) Handle(req *msg.OcReq) (*msg.OcResp, error) {
	methods := make(map[string]func(*msg.OcReq) (*msg.OcResp, error))
	methods[PAYMENT_ADDR_METHOD] = ps.getPaymentAddr
	methods[TXID_METHOD] = ps.txid


	if method, ok := methods[req.Method]; ok {
		return method(req)
	} else {
		return msg.NewRespError(msg.METHOD_UNSUPPORTED), nil
	}
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

func (ps *PaymentService) txid(req *msg.OcReq) (*msg.OcResp, error) {
	reqTxid := req.Args[0]
	reqCurrency := msg.Currency(req.Args[1])
	switch reqCurrency {
	case msg.BTC:
		if ps.BitcoindConf == nil {
			return msg.NewRespError(msg.SERVER_ERROR), nil
		}
		// For now, just see if we can find the transaction
		cmd, err := btcjson.NewGetTransactionCmd("", string(reqTxid))
		if err != nil {
			return nil, fmt.Errorf("error while making cmd: %v", err.Error())
		}
		resp, err := util.SendBtcRpc(cmd, ps.BitcoindConf)
		if err != nil {
			return msg.NewRespError(msg.SERVER_ERROR), nil
		}
		if resp.Error != nil {
			if resp.Error.Code == -5 {
				return msg.NewRespError(msg.INVALID_TXID), nil
			} else {
				// Catch-all
				return msg.NewRespError(msg.PAYMENT_DECLINED), nil
			}
		}
		return msg.NewRespOk([]byte("")), nil
	default:
		return msg.NewRespError(msg.CURRENCY_UNSUPPORTED), nil
	}
}
