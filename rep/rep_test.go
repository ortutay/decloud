package rep

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/util"
)

func initDir(t *testing.T) string {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	util.SetAppDir(dir)
	return dir
}

func TestRepPut(t *testing.T) {
	defer os.RemoveAll(initDir(t))
	rec := Record{
		Role:         SERVER,
		Service:      "store",
		Method:       "put",
		Timestamp:    1234,
		OcID:         msg.OcID("id-123"),
		Status:       SUCCESS_PAID,
		PaymentType:  msg.TXID,
		PaymentValue: &msg.PaymentValue{Amount: 1000, Currency: msg.BTC},
		Perf:         nil,
	}

	_, err := Put(&rec)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSuccessRate(t *testing.T) {
	defer os.RemoveAll(initDir(t))
	id := msg.OcID("id-123")
	otherID := msg.OcID("id-other")
	_, err := Put(&Record{Role: SERVER, OcID: id, Status: SUCCESS_PAID})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Put(&Record{Role: SERVER, OcID: id, Status: PENDING})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Put(&Record{Role: SERVER, OcID: id, Status: FAILURE})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Put(&Record{Role: SERVER, OcID: otherID, Status: FAILURE})
	if err != nil {
		t.Fatal(err)
	}

	rate, err := SuccessRate(&Record{Role: SERVER, OcID: id})
	if err != nil {
		t.Fatal(err)
	}
	if .5 != rate {
		t.Fatal("Expected %v, got %v", .5, rate)
	}

	rate, err = SuccessRate(&Record{Role: CLIENT, OcID: id})
	if err != nil {
		t.Fatal(err)
	}
	if -1 != rate {
		t.Fatal("Expected %v, got %v", -1, rate)
	}
}
