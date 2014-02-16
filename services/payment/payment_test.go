package payment

import (
	"github.com/conformal/btcjson"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/util"
	"log"
	"testing"
)

func TestTxid(t *testing.T) {
	btcConf, err := util.LoadBitcoindConf("")
	if err != nil {
		t.Errorf("%v", err)
	}
	ps := PaymentService{BitcoindConf: btcConf}
	addr, err := ps.fetchNewBtcAddr()
	if err != nil {
		log.Fatal(err)
	}

	cmd, err := btcjson.NewSendToAddressCmd("", addr, 1e6)
	if err != nil {
		log.Fatal(err)
	}
	sendBtcResp, err := util.SendBtcRpc(cmd, btcConf)
	txid, ok := sendBtcResp.Result.(string)
	if !ok {
		log.Fatal(sendBtcResp)
	}

	req := NewBtcTxidReq(msg.BtcTxid(txid))
	resp, err := ps.txid(req)
	if err != nil {
		log.Fatal(err)
	}
	if resp.Status != msg.OK {
		log.Fatalf("expected %v, but got %v\n", msg.OK, resp.Status)
	}
}

func TestFakeTxidError(t *testing.T) {
	btcConf, err := util.LoadBitcoindConf("")
	if err != nil {
		t.Errorf("%v", err)
	}
	ps := PaymentService{BitcoindConf: btcConf}
	req := NewBtcTxidReq(msg.BtcTxid("fake txid"))
	resp, err := ps.txid(req)
	if err != nil {
		log.Fatal(err)
	}
	if resp.Status != msg.INVALID_TXID {
		log.Fatalf("expected %v, but got %v\n", msg.INVALID_TXID, resp.Status)
	}
}
