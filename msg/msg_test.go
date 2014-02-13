package msg

import(
	"bufio"
	"testing"
	"bytes"
)

func TestEncodeDecodeOcReq(t *testing.T) {
	req := OcReq{
		Id: []string{"id1", "id2", "id3"},
		Sig: []string{"sig1", "sig2", "sig3"},
		Nonce: "abcnonce",
		Service: "testService",
		Method: "testMethod",
		Args: []string{"1", "2"},
		PaymentType: ATTACHED,
		PaymentValue: &PaymentValue{Amount: 1e8, Currency: BTC},
		PaymentTxn: "base64-btc-txn",
		Body: []byte("some body, just a string here, but could be binary data"),
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
