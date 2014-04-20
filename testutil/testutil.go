package testutil

import (
	"io/ioutil"
	"testing"

	"github.com/ortutay/decloud/util"
)
func InitDir(t *testing.T) string {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	util.SetAppDir(dir)
	return dir
}
