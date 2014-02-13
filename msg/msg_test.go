package msg

import (
	"bufio"
	"bytes"
	"testing"
)

func TestReadWriteOcReq(t *testing.T) {
	body := []byte("some body, just a string here, but could be binary data")
	req := OcReq{
		Id:            []string{"id1", "id2", "id3"},
		Sig:           []string{"sig1", "sig2", "sig3"},
		Nonce:         "abcnonce",
		Service:       "testService",
		Method:        "testMethod",
		Args:          []string{"1", "2"},
		PaymentType:   ATTACHED,
		PaymentValue:  &PaymentValue{Amount: 1e8, Currency: BTC},
		PaymentTxn:    "base64-btc-txn",
		ContentLength: len(body),
		Body:          body,
	}
	buf := bytes.Buffer{}
	err := req.Write(&buf)
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
		Id:            []string{"id1", "id2", "id3"},
		Sig:           []string{"sig1", "sig2", "sig3"},
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
