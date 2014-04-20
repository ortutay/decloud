package rep

import (
	"os"
	"testing"

	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/testutil"
)

func TestRepPut(t *testing.T) {
	defer os.RemoveAll(testutil.InitDir(t))
	rec := Record{
		Role:         SERVER,
		Service:      "store",
		Method:       "put",
		Timestamp:    1234,
		ID:         msg.OcID("id-123"),
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
	defer os.RemoveAll(testutil.InitDir(t))
	id := msg.OcID("id-123")
	otherID := msg.OcID("id-other")
	_, err := Put(&Record{Role: SERVER, ID: id, Status: SUCCESS_PAID})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Put(&Record{Role: SERVER, ID: id, Status: PENDING})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Put(&Record{Role: SERVER, ID: id, Status: FAILURE})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Put(&Record{Role: SERVER, ID: otherID, Status: FAILURE})
	if err != nil {
		t.Fatal(err)
	}

	rate, err := SuccessRate(&Record{Role: SERVER, ID: id})
	if err != nil {
		t.Fatal(err)
	}
	if .5 != rate {
		t.Fatal("Expected %v, got %v", .5, rate)
	}

	rate, err = SuccessRate(&Record{Role: CLIENT, ID: id})
	if err != nil {
		t.Fatal(err)
	}
	if -1 != rate {
		t.Fatal("Expected %v, got %v", -1, rate)
	}
}

func TestPaymentValueServedToOcID(t *testing.T) {
	defer os.RemoveAll(testutil.InitDir(t))
	id := msg.OcID("id-123")
	otherID := msg.OcID("id-other")
	_, err := Put(&Record{
		Role: SERVER,
		ID: id,
		Status: SUCCESS_PAID,
		PaymentValue: &msg.PaymentValue{Amount: 1000, Currency: msg.BTC},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Put(&Record{
		Role: SERVER,
		ID: id,
		Status: SUCCESS_UNPAID,
		PaymentValue: &msg.PaymentValue{Amount: 2000, Currency: msg.BTC},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Put(&Record{
		Role: SERVER,
		ID: otherID,
		Status: SUCCESS_PAID,
		PaymentValue: &msg.PaymentValue{Amount: 3000, Currency: msg.BTC},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Put(&Record{
		Role: SERVER,
		ID: id,
		Status: PENDING,
		PaymentValue: &msg.PaymentValue{Amount: 4000, Currency: msg.BTC},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Put(&Record{
		Role: SERVER,
		ID: id,
		Status: FAILURE,
		PaymentValue: &msg.PaymentValue{Amount: 5000, Currency: msg.BTC},
	})
	if err != nil {
		t.Fatal(err)
	}

	pv, err := PaymentValueServedToOcID(id)
	if err != nil {
		t.Fatal(err)
	}
	if 3000 != pv.Amount {
		t.Fatalf("%v != %v", 3000, pv.Amount)
	}
}
