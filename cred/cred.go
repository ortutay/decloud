package cred
// TODO(ortutay): different name?

import (
	"fmt"
	"oc/msg"
	"crypto/ecdsa"
	"crypto/rand"
	"bytes"
	"oc/util"
	"crypto/sha256"
	"crypto/elliptic"
	"math/big"
	"strings"
	"io"
)

const (
	PRIVATE_KEY_FILENAME = "nodeid-priv"
	OC_ID_PREFIX = 'd' // "d" prefix for "decloud"
	NODE_ID_RAND_NUM_BYTES = 256
	SIG_RAND_NUM_BYTES = 256
)

type OcCred struct {
	Priv *ecdsa.PrivateKey // TODO(ortutay): make private field?
}

func NewOcCred() (*OcCred, error) {
	randBytes := make([]byte, NODE_ID_RAND_NUM_BYTES)
	_, err := rand.Read(randBytes)
	if err != nil { return nil, err }

	curve := elliptic.P256()
	priv, err := ecdsa.GenerateKey(curve, bytes.NewReader(randBytes))
	if err != nil { return nil, err }

	ocCred := OcCred{
		Priv: priv,
	}
	return &ocCred, nil
}

func (cred *OcCred) StorePrivateKey(filename string)  error {
	if filename == "" { filename = PRIVATE_KEY_FILENAME }
	d := fmt.Sprintf("%x\n", cred.Priv.D)
	err := util.StoreAppData(filename, []byte(d), 0600)
	if err != nil { return err }
	return nil
}

func NewOcCredLoadFromFile(filename string) (*OcCred, error) {
	if filename == "" { filename = PRIVATE_KEY_FILENAME }
	file, err := util.GetAppData(filename)
	if err != nil { return nil, err }
	var d big.Int
	fmt.Fscanf(file, "%x\n", &d)
	curve := elliptic.P256()
	x, y := curve.ScalarBaseMult(d.Bytes())
	priv := ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
			X: x,
			Y: y,
		},
		D: &d,
	}

	ocCred := OcCred{
		Priv: &priv,
	}

	return &ocCred, nil
}

func getReqSigDataHash(req *msg.OcReq) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(req.Nonce)
	buf.WriteString(req.Service)
	buf.WriteString(req.Method)
	for _, arg := range req.Args {
		buf.WriteString(arg)
	}
	buf.WriteString(req.PaymentType)
	buf.WriteString(req.PaymentTxn)
	buf.Write(req.Body)

	hasher := sha256.New()
	_, err := hasher.Write(buf.Bytes())
	if err != nil { return nil, err}

	h := hasher.Sum([]byte{})
	return h, nil
}

func (cred *OcCred) SignOcReq(req *msg.OcReq) error {
	h, err := getReqSigDataHash(req)
	if err != nil { return err}

	randBytes := make([]byte, SIG_RAND_NUM_BYTES)
	_, err = rand.Read(randBytes)
	if err != nil {return err }

	r, s, err := ecdsa.Sign(bytes.NewReader(randBytes), cred.Priv, h)
	if err != nil {return err }
	// TODO(ortutay): compress pub key
	req.NodeId = append(req.NodeId, fmt.Sprintf("%c%x,%x",
		OC_ID_PREFIX, cred.Priv.PublicKey.X, cred.Priv.PublicKey.Y))
	req.Sig = append(req.Sig, fmt.Sprintf("%x,%x", r, s))

	return nil
}

func (cred *OcCred) VerifyOcReqSig(req *msg.OcReq) bool {
	if len(req.NodeId) != len(req.Sig) {
		return false
	}

	h, err := getReqSigDataHash(req)
	if err != nil { return false }
	
	for i, _ := range(req.NodeId) {
		nodeId := strings.NewReader(req.NodeId[i])
		sig := strings.NewReader(req.Sig[i])
		var x, y, r, s big.Int

		n, err := fmt.Fscanf(nodeId, "d%x,%x", &x, &y)
		if err != nil { return false }
		n, err = nodeId.Read(make([]byte, 1))
		if n != 0 || err != io.EOF {
			return false
		}

		n, err = fmt.Fscanf(sig, "%x,%x", &r, &s)
		if err != nil { return false }
		n, err = sig.Read(make([]byte, 1))
		if n != 0 || err != io.EOF {
			return false
		}

		curve := elliptic.P256()
		pub := ecdsa.PublicKey{
			Curve: curve,
			X: &x,
			Y: &y,
		}
		if !ecdsa.Verify(&pub, h, &r, &s) {
			return false
		}
	}

	return true
}
