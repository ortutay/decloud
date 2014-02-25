package msg

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestReadWriteOcReq(t *testing.T) {
	body := []byte("some body, just a string here, but could be binary data")
	args := []string{"1", "2"}
	argsJson, err := json.Marshal(args)
	if err != nil {
		t.Fatalf(err.Error())
	}
	req := OcReq{
		ID:            "id1",
		Sig:           "sig1",
		Coins:         []string{"1addr1", "1addr2"},
		CoinSigs:      []string{"addr1sig", "addr2sig"},
		Nonce:         "abcnonce",
		Service:       "testService",
		Method:        "testMethod",
		Args:          argsJson,
		PaymentType:   ATTACHED,
		PaymentValue:  &PaymentValue{Amount: 1e8, Currency: BTC},
		PaymentTxn:    "base64-btc-txn",
		ContentLength: len(body),
		Body:          body,
	}
	buf := bytes.Buffer{}
	err = req.Write(&buf)
	if err != nil {
		t.Fatalf(err.Error())
	}
	b := bufio.NewReader(&buf)
	req2, err := ReadOcReq(b)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if req.String() != req2.String() {
		t.Fatalf("%v != %v\n", req.String(), req2.String())
	}
}

func TestReadWriteOcResp(t *testing.T) {
	body := []byte("some body, just a string here, but could be binary data")
	resp := OcResp{
		ID:            "id1",
		Sig:           "sig1",
		Coins:         []string{"1addr1", "1addr2"},
		CoinSigs:      []string{"addr1sig", "addr2sig"},
		Nonce:         "abcnonce",
		Status:        OK,
		ContentLength: len(body),
		Body:          body,
	}
	buf := bytes.Buffer{}
	err := resp.Write(&buf)
	if err != nil {
		t.Fatalf(err.Error())
	}
	b := bufio.NewReader(&buf)
	resp2, err := ReadOcResp(b)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if resp.String() != resp2.String() {
		t.Fatalf("%v != %v\n", resp.String(), resp2.String())
	}
}

func TestGetPaymentValue(t *testing.T) {
	pv, err := NewPaymentValueParseString(".1BTC")
	if err != nil {
		t.Fatalf(err.Error())
	}
	if BTC != pv.Currency {
		t.Fatalf("expected %v, got %v", BTC, pv.Currency)
	}
	if 1e7 != pv.Amount {
		t.Fatalf("expected %v, got %v", 1e7, pv.Amount)
	}
}

func TestGetPaymentValueAlternateFormat(t *testing.T) {
	pv, err := NewPaymentValueParseString("2.1 BTC")
	if err != nil {
		t.Fatalf(err.Error())
	}
	if BTC != pv.Currency {
		t.Fatalf("expected %v, got %v", BTC, pv.Currency)
	}
	if 2.1e8 != pv.Amount {
		t.Fatalf("expected %v, got %v", 2.1e8, pv.Amount)
	}
}

func TestGetPaymentValueOverMaxPrecision(t *testing.T) {
	_, err := NewPaymentValueParseString(".123456789BTC")
	if err == nil {
		t.FailNow()
	}
	if !strings.HasPrefix(err.Error(), "max precision is 8") {
		t.FailNow()
	}
}
