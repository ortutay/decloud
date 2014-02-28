package rep

import(
	"fmt"
	"testing"
	"github.com/ortutay/decloud/util"
	"github.com/ortutay/decloud/msg"
	"io/ioutil"
	// "os"
)

func TestRepPut(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	// defer os.RemoveAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("dir %v\n", dir)
	util.SetAppDir(dir)

	rec := Record{
		Service: "store",
		Method: "put",
		Timestamp: 1234,
		OcID: msg.OcID("id-123"),
		Status: SUCCESS,
		PaymentType: msg.TXID,
		PaymentValue: msg.PaymentValue{Amount: 1000, Currency: msg.BTC},
		Perf: nil,
	}

	Put(rec)
}
