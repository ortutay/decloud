package cred

import (
	"testing"
	"fmt"
	"io/ioutil"
	"os"
	"oc/buyer/client"
)
var _ = fmt.Printf

func TestNewOcCred(t *testing.T) {
	_, err := NewOcCred()
	if err != nil { t.Errorf("%v", err) }
}

func TestStoreAndLoadOcCred(t *testing.T) {
	destDir, err := ioutil.TempDir("", "msgtest")
	dest := destDir + "/tmp-nodeid-priv"

	ocCred, err := NewOcCred()
	if err != nil { t.Errorf("%v", err) }

	err = ocCred.StorePrivateKey(dest)
	if err != nil { t.Errorf("%v", err) }

	ocCred2, err := NewOcCredLoadFromFile(dest)
	if err != nil { t.Errorf("%v", err) }

	priv := ocCred.Priv
	priv2 := ocCred2.Priv
	if (priv.D.Cmp(priv2.D) != 0 ||
		priv.PublicKey.X.Cmp(priv2.PublicKey.X) != 0 ||
		priv.PublicKey.Y.Cmp(priv2.PublicKey.Y) != 0) {
		t.Errorf("private keys differ:\n%v\n%v\n", priv.D, priv2.D)
	}

	err = os.RemoveAll(destDir)
	if err != nil { t.Errorf("%v", err) }
}

func TestSignRequest(t *testing.T) {
	ocReq := client.NewCalcReq([]string{"1 2 +"})

	ocCred, err := NewOcCred()
	if err != nil { t.Errorf("%v", err) }

	err = ocCred.SignOcReq(&ocReq)
	if err != nil { t.Errorf("%v", err) }
	if len(ocReq.NodeId) != 1 || len(ocReq.Sig) != 1 {
		t.Errorf("expected exactly 1 id and sig, got %v %v",
			len(ocReq.NodeId), len(ocReq.Sig))
	}

	ok := ocCred.VerifyOcReqSig(&ocReq)
	if !ok {
		t.Errorf("sig did not verify")
	}
}

func TestInvalidSignatureFails(t *testing.T) {
	ocReq := client.NewCalcReq([]string{"1 2 +"})

	ocCred, err := NewOcCred()
	if err != nil { t.Errorf("%v", err) }

	err = ocCred.SignOcReq(&ocReq)
	if err != nil { t.Errorf("%v", err) }

	originalSig := ocReq.Sig[0]

	ocReq.Sig[0] = originalSig[1:] + "1"
	if ocCred.VerifyOcReqSig(&ocReq) {
		t.Errorf("invalid sig %v verified", ocReq.Sig[0])
	}

	ocReq.Sig[0] = originalSig + "x"
	if ocCred.VerifyOcReqSig(&ocReq) {
		t.Errorf("invalid sig %v verified", ocReq.Sig[0])
	}

	originalNodeId := ocReq.NodeId[0]
	ocReq.NodeId[0] = originalNodeId[1:] + "1"
	if ocCred.VerifyOcReqSig(&ocReq) {
		t.Errorf("invalid node id %v verified", ocReq.NodeId[0])
	}

	ocReq.NodeId[0] = originalNodeId + "x"
	if ocCred.VerifyOcReqSig(&ocReq) {
		t.Errorf("invalid node id %v verified", ocReq.NodeId[0])
	}
}
