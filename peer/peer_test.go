package peer

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ortutay/decloud/cred"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/services/calc"
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

func TestGetNotFound(t *testing.T) {
	defer os.RemoveAll(initDir(t))
	v, err := ocIDForCoin("1abc")
	if err != nil {
		t.Fatal(err)
	}
	if v != nil {
		t.FailNow()
	}
}

func TestGetAndSet(t *testing.T) {
	defer os.RemoveAll(initDir(t))
	id := msg.OcID("1abc")
	err := setOcIDForCoin("1abc", &id)
	if err != nil {
		t.Fatalf("err: %v", err.Error())
	}
	id2, err := ocIDForCoin("1abc")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	fmt.Printf("id2: %v\n", id2)
	if id != *id2 {
		t.Fatalf("%v %v", id, id2)
	}
}

func TestPeerFromReq(t *testing.T) {
	defer os.RemoveAll(initDir(t))
	ocCred := cred.NewOcCred()
	btcConf, err := util.LoadBitcoindConf("")
	if err != nil {
		t.Fatal(err)
	}
	// TODO(ortutay): this test is flakey, as we may not have any BTC at all
	btcCreds, err := cred.GetBtcCredInRange(0, util.B2S(1000), btcConf)
	if err != nil {
		t.Fatal(err)
	}
	req := calc.NewCalcReq([]string{"1 2 +"})
	err = ocCred.SignOcReq(req)
	if err != nil {
		t.Fatal(err)
	}
	for _, bc := range *btcCreds {
		err = bc.SignOcReq(req, btcConf)
		if err != nil {
			t.Fatal(err)
		}
	}
	p, err := NewPeerFromReq(req, btcConf)
	if err != nil {
		t.Fatal(err)
	}
	if p.OcID != req.ID {
		t.FailNow()
	}
}

func TestPeerFromReqCoinReuse(t *testing.T) {
	defer os.RemoveAll(initDir(t))
	ocCred1 := cred.NewOcCred()
	ocCred2 := cred.NewOcCred()
	btcConf, err := util.LoadBitcoindConf("")
	if err != nil {
		t.Fatal(err)
	}
	// TODO(ortutay): this test is flakey, as we may not have any BTC at all
	btcCreds, err := cred.GetBtcCredInRange(0, util.B2S(1000), btcConf)
	if err != nil {
		t.Fatal(err)
	}
	req1 := calc.NewCalcReq([]string{"1 2 +"})
	err = ocCred1.SignOcReq(req1)
	if err != nil {
		t.Fatal(err)
	}
	req2 := calc.NewCalcReq([]string{"1 2 +"})
	err = ocCred2.SignOcReq(req2)
	if err != nil {
		t.Fatal(err)
	}
	for _, bc := range *btcCreds {
		err = bc.SignOcReq(req1, btcConf)
		err = bc.SignOcReq(req2, btcConf)
		if err != nil {
			t.Fatal(err)
		}
	}
	p1, err := NewPeerFromReq(req1, btcConf)
	if err != nil {
		t.Fatal(err)
	}
	if p1.OcID != req1.ID {
		t.FailNow()
	}
	p2, err := NewPeerFromReq(req2, btcConf)
	if err == nil || err != COIN_REUSE {
		t.Fatal("Expected COIN_REUSE error")
	}
	if p2 != nil {
		t.FailNow()
	}
}
