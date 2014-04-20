package payment

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strings"

	"github.com/conformal/btcjson"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/util"
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
	switch reqCurrency {
	case string(msg.BTC):
		if ps.BitcoindConf == nil {
			return msg.NewRespError(msg.SERVER_ERROR), nil
		}
		// TODO(ortutay): smarter handling to map request ID to address
		btcAddr, err := ps.addrForOcID(req.ID)
		if err != nil {
			return msg.NewRespError(msg.SERVER_ERROR), nil
		}
		payAddr := msg.PaymentAddr{Currency: msg.BTC, Addr: btcAddr}
		return msg.NewRespOk([]byte(payAddr.String())), nil
	default:
		return msg.NewRespError(msg.CURRENCY_UNSUPPORTED), nil
	}
}

func addrDBPath() string {
	return util.AppDir() + "/payment-addrs-diskv.db"
}

func (ps *PaymentService) addrForOcID(id msg.OcID) (string, error) {
	d := util.GetOrCreateDB(addrDBPath())
	addrsSer, _ := d.Read(id.String())
	fmt.Printf("read addrs: %v\n", addrsSer)
	if addrsSer == nil || len(addrsSer) == 0 {
		fmt.Printf("no addrs read, making...\n")
		var addrs []string
		for i := 0; i < ADDRS_PER_ID; i++ {
			btcAddr, err := ps.fetchNewBtcAddr()
			if err != nil {
				log.Printf("error while generating addresses: %v\n", err)
				return "", err
			}
			addrs = append(addrs, btcAddr)
		}
		ser, err := json.Marshal(addrs)
		if err != nil {
			return "", err
		}
		err = d.Write(id.String(), ser)
		if err != nil {
			return "", err
		}
		fmt.Printf("generated addresses: %v ser: %v\n", addrs, string(ser))
		addrsSer, _ = d.Read(id.String())
	}
	var addrs []string
	err := json.Unmarshal(addrsSer, &addrs)
	if err != nil {
		return "", err
	}
	if addrs == nil || len(addrs) == 0 {
		panic("unexpected empty list")
	}
	return addrs[rand.Int()%len(addrs)], nil
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
