package cred

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/conformal/btcjson"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/util"
)

var _ = fmt.Printf

func newReq() *msg.OcReq {
	msg := msg.OcReq{
		ID:          "",
		Sig:         "",
		Coins:       []string{},
		CoinSigs:    []string{},
		Nonce:       "",
		Service:     "storage",
		Method:      "get",
		Args:        []string{"123"},
		PaymentType: "None",
		PaymentTxn:  "blob",
		Body:        []byte(""),
	}
	return &msg
}

func TestNewOcCred(t *testing.T) {
	_, err := NewOcCred()
	if err != nil {
		t.Errorf("%v", err)
	}
}

func TestStoreAndLoadOcCred(t *testing.T) {
	destDir, err := ioutil.TempDir("", "msgtest")
	dest := destDir + "/tmp-nodeid-priv"

	ocCred, err := NewOcCred()
	if err != nil {
		t.Errorf("%v", err)
	}

	err = ocCred.StorePrivateKey(dest)
	if err != nil {
		t.Errorf("%v", err)
	}

	ocCred2, err := NewOcCredLoadFromFile(dest)
	if err != nil {
		t.Errorf("%v", err)
	}

	priv := ocCred.Priv
	priv2 := ocCred2.Priv
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

	ocCred, err := NewOcCred()
	if err != nil {
		t.Fatalf("%v", err)
	}

	err = ocCred.SignOcReq(ocReq)
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
	ocCred, err := NewOcCred()
	if err != nil {
		t.Errorf("%v", err)
	}

	err = ocCred.SignOcReq(ocReq)
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

	originalID := ocReq.ID
	ocReq.ID = originalID[1:] + "1"
	if ok, _ := VerifyOcReqSig(ocReq, nil); ok {
		t.Errorf("invalid node id %v verified", ocReq.ID)
	}

	if ok, _ := VerifyOcReqSig(ocReq, nil); ok {
		t.Errorf("invalid node id %v verified", ocReq.ID)
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

	originalID := ocReq.Coins[0]
	ocReq.Coins[0] = originalID[0:len(originalID)-2] + "1"
	ok, err = VerifyOcReqSig(ocReq, conf)
	if ok {
		t.Errorf("invalid node id %v verified", ocReq.Coins[0])
	}
	if err == nil || err.Error() != "-3: Invalid address" {
		t.Errorf("expected invalid address error, but got  %v", err)
	}
}

func TestInputsInRangeFound(t *testing.T) {
	addrsList := []addressBalance{
		addressBalance{Amount: 100, Address: "123"},
		addressBalance{Amount: 200, Address: "456"},
		addressBalance{Amount: 300, Address: "789"},
		addressBalance{Amount: 300, Address: "012"},
	}
	use, err := inputsInRange(&addrsList, 150, 350, 2, len(addrsList)-1)
	if err != nil {
		t.Error(err)
	}
	if (*use)[0].Addr != "123" || (*use)[1].Addr != "456" {
		t.FailNow()
	}
}

func TestInputsInRangeNotFound(t *testing.T) {
	addrsList := []addressBalance{
		addressBalance{Amount: 100, Address: "123"},
		addressBalance{Amount: 200, Address: "456"},
		addressBalance{Amount: 300, Address: "789"},
		addressBalance{Amount: 300, Address: "012"},
	}
	_, err := inputsInRange(&addrsList, 150, 350, 3, len(addrsList)-1)
	if err == nil {
		t.FailNow()
	}
}
