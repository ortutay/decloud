package cred

import (
	"errors"
	"fmt"
	"github.com/conformal/btcjson"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/util"
	"io/ioutil"
	"os"
	"testing"
)

var _ = fmt.Printf

func newReq() *msg.OcReq {
	msg := msg.OcReq{
		Id:          "",
		Sig:         "",
		Coins:       []string{},
		CoinSigs:    []string{},
		Nonce:       "",
		Service:     "storage",
		Method:      "get",
		Args:        []byte("123"),
		PaymentType: "None",
		PaymentTxn:  "blob",
		Body:        []byte(""),
	}
	return &msg
}

func TestNewOcID(t *testing.T) {
	_, err := NewOcID()
	if err != nil {
		t.Errorf("%v", err)
	}
}

func TestStoreAndLoadOcID(t *testing.T) {
	destDir, err := ioutil.TempDir("", "msgtest")
	dest := destDir + "/tmp-nodeid-priv"

	ocID, err := NewOcID()
	if err != nil {
		t.Errorf("%v", err)
	}

	err = ocID.StorePrivateKey(dest)
	if err != nil {
		t.Errorf("%v", err)
	}

	ocID2, err := NewOcIDLoadFromFile(dest)
	if err != nil {
		t.Errorf("%v", err)
	}

	priv := ocID.Priv
	priv2 := ocID2.Priv
	if priv.D.Cmp(priv2.D) != 0 ||
		priv.PublicKey.X.Cmp(priv2.PublicKey.X) != 0 ||
		priv.PublicKey.Y.Cmp(priv2.PublicKey.Y) != 0 {
		t.Errorf("private keys differ:\n%v\n%v\n", priv.D, priv2.D)
	}

	err = os.RemoveAll(destDir)
	if err != nil {
		t.Errorf("%v", err)
	}
}

func TestSignRequest(t *testing.T) {
	ocReq := newReq()

	ocID, err := NewOcID()
	if err != nil {
		t.Fatalf("%v", err)
	}

	err = ocID.SignOcReq(ocReq)
	if err != nil {
		t.Fatalf("%v", err)
	}

	ok, err := VerifyOcReqSig(ocReq, nil)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if !ok {
		t.Fatalf("sig did not verify")
	}
}

func TestInvalidOcSignatureFails(t *testing.T) {
	ocReq := newReq()
	ocID, err := NewOcID()
	if err != nil {
		t.Errorf("%v", err)
	}

	err = ocID.SignOcReq(ocReq)
	if err != nil {
		t.Errorf("%v", err)
	}

	originalSig := ocReq.Sig
	ocReq.Sig = originalSig[0:len(originalSig)-2] + "1"
	if ok, _ := VerifyOcReqSig(ocReq, nil); ok {
		t.Errorf("invalid sig %v verified", ocReq.Sig)
	}

	ocReq.Sig = originalSig + "x"
	if ok, _ := VerifyOcReqSig(ocReq, nil); ok {
		t.Errorf("invalid sig %v verified", ocReq.Sig)
	}

	originalId := ocReq.Id
	ocReq.Id = originalId[1:] + "1"
	if ok, _ := VerifyOcReqSig(ocReq, nil); ok {
		t.Errorf("invalid node id %v verified", ocReq.Id)
	}

	if ok, _ := VerifyOcReqSig(ocReq, nil); ok {
		t.Errorf("invalid node id %v verified", ocReq.Id)
	}
}

type AddressResult struct {
	Address       string  `json:"address"`
	Account       string  `json:"account"`
	Amount        float64 `json:"amount"`
	Confirmations int     `json:"confirmations"`
}

type ListReceivedByAddressResult struct {
	Addresses []AddressResult
}

func getAnyBtcAddr(conf *util.BitcoindConf) (string, error) {
	msg, err := btcjson.NewListReceivedByAddressCmd(nil, 0, true)
	if err != nil {
		return "", err
	}

	json, err := msg.MarshalJSON()
	if err != nil {
		return "", err
	}

	resp, err := btcjson.RpcCommand(conf.User, conf.Password, conf.Server, json)
	if err != nil {
		return "", err
	}

	for _, r := range resp.Result.([]interface{}) {
		result := r.(map[string]interface{})
		return result["address"].(string), nil
	}

	return "", errors.New("no address found")
}

func printBitcoindExpected() {
	println("Note: bitcoind daemon expected to be running")
}

func TestBtcCredSign(t *testing.T) {
	printBitcoindExpected()
	conf, err := util.LoadBitcoindConf("")
	if err != nil {
		t.Errorf("%v", err)
	}

	ocReq := newReq()

	addr, err := getAnyBtcAddr(conf)
	if err != nil {
		t.Errorf("%v", err)
	}

	if err != nil {
		t.Errorf("%v", err)
	}
	btcCred := BtcCred{
		Addr: addr,
	}

	err = btcCred.SignOcReq(ocReq, conf)
	if err != nil {
		t.Errorf("%v", err)
	}

	ok, err := VerifyOcReqSig(ocReq, conf)
	if err != nil {
		t.Errorf("%v", err)
	}
	if !ok {
		t.Errorf("sig did not verify")
	}
}

func TestInvalidBtcSignatureFails(t *testing.T) {
	printBitcoindExpected()
	conf, err := util.LoadBitcoindConf("")
	if err != nil {
		t.Errorf("%v", err)
	}

	ocReq := newReq()

	addr, err := getAnyBtcAddr(conf)
	if err != nil {
		t.Errorf("%v", err)
	}

	if err != nil {
		t.Errorf("%v", err)
	}
	btcCred := BtcCred{
		Addr: addr,
	}

	err = btcCred.SignOcReq(ocReq, conf)
	if err != nil {
		t.Errorf("%v", err)
	}

	originalSig := ocReq.CoinSigs[0]
	ocReq.CoinSigs[0] = originalSig[0:len(originalSig)-2] + "1"
	ok, err := VerifyOcReqSig(ocReq, conf)
	if ok {
		t.Errorf("invalid sig %v verified", ocReq.CoinSigs[0])
	}
	if err == nil || err.Error() != "-5: Malformed base64 encoding" {
		t.Errorf("expected malformed base64 encoding error, but got  %v", err)
	}

	originalId := ocReq.Coins[0]
	ocReq.Coins[0] = originalId[0:len(originalId)-2] + "1"
	ok, err = VerifyOcReqSig(ocReq, conf)
	if ok {
		t.Errorf("invalid node id %v verified", ocReq.Coins[0])
	}
	if err == nil || err.Error() != "-3: Invalid address" {
		t.Errorf("expected invalid address error, but got  %v", err)
	}
}
