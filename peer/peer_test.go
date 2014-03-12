package peer

import (
	"os"
	"testing"
	"io/ioutil"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/util"
	"code.google.com/p/leveldb-go/leveldb/db"
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
	_, err := ocIDForCoin("1abc")
	if err == nil || err != db.ErrNotFound {
		t.Fatalf("err: %v", err)
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
	if id != *id2 {
		t.Fatalf("%v %v", id, id2)
	}
}
