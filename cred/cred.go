package cred

// TODO(ortutay): different name?

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/conformal/btcjson"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/util"
	"io"
	"math/big"
	"strings"
)

const (
	PRIVATE_KEY_FILENAME   = "nodeid-priv"
	OC_ID_PREFIX           = 'c' // "c" for open"c"loud
	NODE_ID_RAND_NUM_BYTES = 256
	SIG_RAND_NUM_BYTES     = 256
)

type Signer interface {
	SignOcReq(req *msg.OcReq) error
}

type Cred struct {
	Signers []Signer
}

func (cred *Cred) SignOcReq(req *msg.OcReq) error {
	for _, signer := range cred.Signers {
		err := signer.SignOcReq(req)
		if err != nil {
			return fmt.Errorf("error while signing: %v", err.Error())
		}
	}
	return nil
}

type OcCred struct {
	Priv *ecdsa.PrivateKey // TODO(ortutay): make private field?
}

type BtcCred struct {
	Addr string
}

func NewOcCred() (*OcCred, error) {
	randBytes := make([]byte, NODE_ID_RAND_NUM_BYTES)
	_, err := rand.Read(randBytes)
	if err != nil {
		return nil, errors.New("error generating random bytes")
	}

	curve := elliptic.P256()
	priv, err := ecdsa.GenerateKey(curve, bytes.NewReader(randBytes))
	if err != nil {
		return nil, errors.New("error generating ECDSA key")
	}

	ocCred := OcCred{
		Priv: priv,
	}
	return &ocCred, nil
}

func (cred *OcCred) StorePrivateKey(filename string) error {
	if filename == "" {
		filename = PRIVATE_KEY_FILENAME
	}
	d := fmt.Sprintf("%x\n", cred.Priv.D)
	err := util.StoreAppData(filename, []byte(d), 0600)
	if err != nil {
		return fmt.Errorf("error storing app data: %v", err.Error())
	}
	return nil
}

func NewOcCredLoadFromFile(filename string) (*OcCred, error) {
	if filename == "" {
		filename = PRIVATE_KEY_FILENAME
	}
	file, err := util.GetAppData(filename)
	if err != nil {
		return nil, fmt.Errorf("error getting app data: %v", err.Error())
	}
	var d big.Int
	fmt.Fscanf(file, "%x\n", &d)
	curve := elliptic.P256()
	x, y := curve.ScalarBaseMult(d.Bytes())
	priv := ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
			X:     x,
			Y:     y,
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
	req.WriteSignablePortion(&buf)

	hasher := sha256.New()
	_, err := hasher.Write(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("error while hashing: %v", err.Error())
	}

	h := hasher.Sum([]byte{})
	return h, nil
}

// TODO(ortutay): move sign and verify to msg?
func (cred *OcCred) SignOcReq(req *msg.OcReq) error {
	h, err := getReqSigDataHash(req)
	if err != nil {
		return err
	}

	randBytes := make([]byte, SIG_RAND_NUM_BYTES)
	_, err = rand.Read(randBytes)
	if err != nil {
		return errors.New("error generating random bytes")
	}

	r, s, err := ecdsa.Sign(bytes.NewReader(randBytes), cred.Priv, h)
	if err != nil {
		return fmt.Errorf("error during ECDSA signature: %v", err.Error())
	}
	// TODO(ortutay): compress pub key
	req.Id = append(req.Id, fmt.Sprintf("%c%x,%x",
		OC_ID_PREFIX, cred.Priv.PublicKey.X, cred.Priv.PublicKey.Y))
	req.Sig = append(req.Sig, fmt.Sprintf("%x,%x", r, s))

	return nil
}

func VerifyOcReqSig(req *msg.OcReq, conf *util.BitcoindConf) (bool, error) {
	if len(req.Id) != len(req.Sig) {
		return false, nil
	}

	h, err := getReqSigDataHash(req)
	if err != nil {
		return false, err
	}

	for i, _ := range req.Id {
		nodeId := req.Id[i]
		sig := req.Sig[i]
		fmt.Printf("verify %v %v\n", nodeId, sig)

		switch nodeId[0] {
		case OC_ID_PREFIX:
			ok := verifyOcSig(h, nodeId, sig)
			if !ok {
				return false, nil
			}
		case '1', 'm', 'n':
			if conf == nil {
				return false, errors.New("need bitcoind conf to verify btc cred")
			}
			ok, err := verifyBtcSig(h, nodeId, sig, conf)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		default:
			return false, errors.New(
				fmt.Sprintf("unexpected id prefix: %c", nodeId[0]))
		}
	}

	return true, nil
}

func verifyOcSig(reqHash []byte, nodeId string, sig string) bool {
	nodeIdReader := strings.NewReader(nodeId)
	var x, y, r, s big.Int
	n, err := fmt.Fscanf(nodeIdReader, string(OC_ID_PREFIX)+"%x,%x", &x, &y)
	if err != nil {
		return false
	}
	n, err = nodeIdReader.Read(make([]byte, 1))
	if n != 0 || err != io.EOF {
		return false
	}

	sigReader := strings.NewReader(sig)
	n, err = fmt.Fscanf(sigReader, "%x,%x", &r, &s)
	if err != nil {
		return false
	}
	n, err = sigReader.Read(make([]byte, 1))
	if n != 0 || err != io.EOF {
		return false
	}

	curve := elliptic.P256()
	pub := ecdsa.PublicKey{
		Curve: curve,
		X:     &x,
		Y:     &y,
	}
	return ecdsa.Verify(&pub, reqHash, &r, &s)
}

func (bc *BtcCred) SignOcReq(req *msg.OcReq, conf *util.BitcoindConf) error {
	h, err := getReqSigDataHash(req)
	if err != nil {
		return err
	}
	hb64 := base64.StdEncoding.EncodeToString(h)

	msg, err := btcjson.NewSignMessageCmd(nil, bc.Addr, hb64)
	if err != nil {
		return fmt.Errorf("error while making cmd: %v", err.Error())
	}
	json, err := msg.MarshalJSON()
	if err != nil {
		return fmt.Errorf("error while marshaling: %v", err.Error())
	}
	resp, err := btcjson.RpcCommand(conf.User, conf.Password, conf.Server, json)
	if err != nil {
		return fmt.Errorf("error while making bitcoind JSON-RPC: %v", err.Error())
	}
	sig, ok := resp.Result.(string)
	if !ok {
		return errors.New("error during bitcoind JSON-RPC")
	}

	req.Id = append(req.Id, fmt.Sprintf(bc.Addr))
	req.Sig = append(req.Sig, fmt.Sprintf(sig))

	return nil
}

func verifyBtcSig(reqHash []byte, addr string, sig string, conf *util.BitcoindConf) (bool, error) {
	hb64 := base64.StdEncoding.EncodeToString(reqHash)

	msg, err := btcjson.NewVerifyMessageCmd(nil, addr, sig, hb64)
	if err != nil {
		return false, fmt.Errorf("error while making cmd: %v", err.Error())
	}
	json, err := msg.MarshalJSON()
	if err != nil {
		return false, fmt.Errorf("error while marshaling: %v", err.Error())
	}
	resp, err := btcjson.RpcCommand(conf.User, conf.Password, conf.Server, json)
	if err != nil {
		return false, fmt.Errorf(
			"error while making bitcoind JSON-RPC: %v", err.Error())
	}
	if resp.Error != nil {
		return false, resp.Error
	}
	verifyResult, ok := resp.Result.(bool)
	if !ok {
		return false, fmt.Errorf("error during bitcoind JSON-RPC: %v", err.Error())
	}

	return verifyResult, nil
}
