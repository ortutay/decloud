package payment

import (
	"encoding/json"
	"fmt"
	"github.com/conformal/btcjson"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/util"
)

const (
	SERVICE_NAME        = "payment"
	TXID_METHOD         = "txid"
	PAYMENT_ADDR_METHOD = "paymentAddr"
)

type TxidArgs struct {
	Currency msg.Currency `json:"currency"`
	Txid     string       `json:"txid"`
}

func NewPaymentAddrReq(currency msg.Currency) *msg.OcReq {
	argsJson, err := json.Marshal(currency)
	if err != nil {
		panic(err)
	}
	msg := msg.OcReq{
		Id:          []string{},
		Sig:         []string{},
		Nonce:       "",
		Service:     SERVICE_NAME,
		Method:      PAYMENT_ADDR_METHOD,
		Args:        argsJson,
		PaymentType: "",
		PaymentTxn:  "",
		Body:        []byte(""),
	}
	return &msg
}

// TODO(ortutay): sign these reqs with an input to the txn to prove ownership
func NewBtcTxidReq(txid msg.BtcTxid) *msg.OcReq {
	txidArgs := TxidArgs{
		Currency: msg.BTC,
		Txid:     string(txid),
	}
	argsJson, err := json.Marshal(txidArgs)
	if err != nil {
		panic(err)
	}
	msg := msg.OcReq{
		Id:          []string{},
		Sig:         []string{},
		Nonce:       "",
		Service:     SERVICE_NAME,
		Method:      TXID_METHOD,
		Args:        argsJson,
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
	var reqCurrency string
	err := json.Unmarshal(req.Args, &reqCurrency)
	if err != nil {
		return msg.NewRespError(msg.INVALID_ARGUMENTS), nil
	}
	switch reqCurrency {
	case string(msg.BTC):
		if ps.BitcoindConf == nil {
			return msg.NewRespError(msg.SERVER_ERROR), nil
		}
		// TODO(ortutay): smarter handling to map request ID to address
		btcAddr, err := ps.fetchNewBtcAddr()
		if err != nil {
			return msg.NewRespError(msg.SERVER_ERROR), nil
		}
		payAddr := msg.PaymentAddr{Currency: msg.BTC, Addr: btcAddr}
		return msg.NewRespOk([]byte(payAddr.String())), nil
	default:
		return msg.NewRespError(msg.CURRENCY_UNSUPPORTED), nil
	}
}

func (ps *PaymentService) fetchNewBtcAddr() (string, error) {
	cmd, err := btcjson.NewGetNewAddressCmd("")
	if err != nil {
		return "", fmt.Errorf("error while making cmd: %v", err.Error())
	}
	resp, err := util.SendBtcRpc(cmd, ps.BitcoindConf)
	addr, ok := resp.Result.(string)
	if !ok {
		return "", fmt.Errorf("error during bitcoind JSON-RPC: %v", resp)
	}
	return addr, nil
}

func (ps *PaymentService) txid(req *msg.OcReq) (*msg.OcResp, error) {
	var txidArgs TxidArgs
	err := json.Unmarshal(req.Args, &txidArgs)
	if err != nil {
		return msg.NewRespError(msg.INVALID_ARGUMENTS), nil
	}
	switch txidArgs.Currency {
	case msg.BTC:
		if ps.BitcoindConf == nil {
			return msg.NewRespError(msg.SERVER_ERROR), nil
		}
		// For now, just see if we can find the transaction
		cmd, err := btcjson.NewGetTransactionCmd("", string(txidArgs.Txid))
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
	}
	return nil, nil
}
