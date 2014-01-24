package msg

import (
	"testing"
	"io/ioutil"
	"os"
	"fmt"
)
var _ = fmt.Printf

func TestMakeNodeId(t *testing.T) {
	destDir, err := ioutil.TempDir("", "msgtest")
	dest := destDir + "/tmp-nodeid-priv"

	err = MakeNodeId(dest)
	if err != nil { t.Errorf("%v", err) }

	err = os.RemoveAll(destDir)
	if err != nil { t.Errorf("%v", err) }
}

func TestStoreAndGetNodeId(t *testing.T) {
	destDir, err := ioutil.TempDir("", "msgtest")
	dest := destDir + "/tmp-nodeid-priv"

	err = MakeNodeId(dest)
	if err != nil { t.Errorf("%v", err) }

	// get private key
	priv, err := GetNodePrivateKey(dest)
	if err != nil { t.Errorf("%v", err) }

	// store it
	err = StoreNodePrivateKey(dest, priv)
	if err != nil { t.Errorf("%v", err) }

	// get again, ensure they are the same
	priv2, err := GetNodePrivateKey(dest)
	if err != nil { t.Errorf("%v", err) }

	if (priv.D.Cmp(priv2.D) != 0 ||
		priv.PublicKey.X.Cmp(priv2.PublicKey.X) != 0 ||
		priv.PublicKey.Y.Cmp(priv2.PublicKey.Y) != 0) {
		t.Errorf("private keys differ:\n%v\n%v\n", priv.D, priv2.D)
	}

	err = os.RemoveAll(destDir)
	if err != nil { t.Errorf("%v", err) }
}
