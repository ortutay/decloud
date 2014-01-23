package msg

import (
	"testing"
	"io/ioutil"
	"os"
)

func TestMakeNodeId(t *testing.T) {
	destDir, err := ioutil.TempDir("", "msgtest")
	dest := destDir + "/tmp-nodeid-priv"
	println(dest)
	err = MakeNodeId(dest)
	if err != nil {
		t.Errorf("%v", err)
	}
	err = os.RemoveAll(destDir)
	if err != nil {
		t.Errorf("%v", err)
	}
}
