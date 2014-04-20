package payment

import (
	"log"
	"testing"
	"os"

	"github.com/conformal/btcjson"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/util"
	"github.com/ortutay/decloud/testutil"
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

func TestGetAddr(t *testing.T) {
	defer os.RemoveAll(testutil.InitDir(t))
	btcConf, err := util.LoadBitcoindConf("")
	if err != nil {
		t.Errorf("%v", err)
	}
	ps := PaymentService{BitcoindConf: btcConf}
	addr1, err := ps.addrForOcID(msg.OcID("123id"), 1)
	if err != nil {
		log.Fatal(err)
	}
	addr2, err := ps.addrForOcID(msg.OcID("123id"), 1)
	if err != nil {
		log.Fatal(err)
	}
	if addr1 != addr2 {
		t.Fatalf("%v != %v\n", addr1, addr2)
	}
}
