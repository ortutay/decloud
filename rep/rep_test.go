package rep

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/util"
)

func TestRepPut(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	defer os.RemoveAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("dir %v\n", dir)
	util.SetAppDir(dir)

	rec := Record{
		Service:      "store",
		Method:       "put",
		Timestamp:    1234,
		OcID:         msg.OcID("id-123"),
		Status:       SUCCESS,
		PaymentType:  msg.TXID,
		PaymentValue: msg.PaymentValue{Amount: 1000, Currency: msg.BTC},
		Perf:         nil,
	}

	_, err = Put(&rec)
	if err != nil {
		t.Fatal(err)
	}
}
