package payment

import (
	"log"
	"testing"
	"os"

	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/util"
	"github.com/ortutay/decloud/testutil"
)

func TestGetAddr(t *testing.T) {
	defer os.RemoveAll(testutil.InitDir(t))
	btcConf, err := util.LoadBitcoindConf("")
	if err != nil {
		t.Errorf("%v", err)
	}
	req := NewPaymentAddrReq(msg.BTC)
	ps := PaymentService{BitcoindConf: btcConf}
	_, err = ps.getPaymentAddr(req)
	if err != nil {
		log.Fatal(err)
	}
}
