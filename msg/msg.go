package msg

import (
	"bytes"
	"fmt"
	"log"
	"encoding/gob"
	"encoding/base64"
	"oc/util"
	"crypto/sha256"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/elliptic"
	"math/big"
)
var _ = fmt.Printf

// For now, all fields are string or []byte.
// TODO(ortutay): add types for these fields
type OcReq struct {
	NodeId []string
	Sig []string
	Nonce string
	Service string
	Method string
	Args []string
	PaymentType string
	PaymentTxn string
	Body []byte
}

type OcResp struct {
	NodeId []string
	Sig []string
	Nonce string
	Status string
	// TODO(ortutay): status code
	Body []byte
}

func NewRespOk(body []byte) *OcResp {
	resp := OcResp{
		NodeId: []string{},
		Sig: []string{},
		Nonce: "TODO",
		Status: "ok",
		Body: body,
	}
	return &resp
}

func EncodeReq(req *OcReq) []byte {
	return encode(req)
}

func EncodeResp(resp *OcResp) []byte {
	return encode(resp)
}

func encode(m interface{}) []byte {
	// TODO(ortutay): for now, just doing gob->base64 to encode; will need
	// to figure out what to actually do
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(m)
	if err != nil {
		log.Fatal("couldn't encode", m)
	}
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	return []byte(b64)
}

func DecodeReq(b64 string) (*OcReq, error) {
	var req OcReq
	err := decode(b64, &req)
	return &req, err
}

func DecodeResp(b64 string) (*OcResp, error) {
	var resp OcResp
	err := decode(b64, &resp)
	return &resp, err
}

func decode(b64 string, d interface{}) error {
	buf, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return err
	}
	dec := gob.NewDecoder(bytes.NewBuffer(buf))
	err = dec.Decode(d)
	if err != nil {
		return err
	}
	return nil
}

// functions related to node ID/signing; may want these in a different package.
// probably packaged under the concept of a node identity/pseudonym object that
// encompases the assorted private keys, signing, and verification
const (
	PRIVATE_KEY_FILENAME = "nodeid-priv"
	NODE_ID_RAND_NUM_BYTES = 256
	SIG_RAND_NUM_BYTES = 256
)

func MakeNodeId(filename string) error {
	if filename == "" { filename = PRIVATE_KEY_FILENAME }
	b := make([]byte, NODE_ID_RAND_NUM_BYTES)
	_, err := rand.Read(b)
	if err != nil { return err }

	curve := elliptic.P256()
	priv, err := ecdsa.GenerateKey(curve, bytes.NewReader(b))
	if err != nil { return err }

	err = StoreNodePrivateKey(filename, priv)
	if err != nil { return err }

	return nil
}

func StoreNodePrivateKey(filename string, priv *ecdsa.PrivateKey)  error {
	d := fmt.Sprintf("%x\n", priv.D)
	err := util.StoreAppData(filename, []byte(d), 0600)
	if err != nil { return err }
	return nil
}

func GetNodePrivateKey(filename string) (*ecdsa.PrivateKey, error) {
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
	return &priv, nil
}

func getReqSigData(req *OcReq) []byte {
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
	return buf.Bytes()
}

func SignOcReq(req *OcReq) error {
	bytes := getReqSigData(req)

	h := sha256.New()
	h.Write(bytes)
	hash := make([]byte, h.Size())
	println(fmt.Sprintf("hash init: %v", hash))
	hash = h.Sum(hash)

	randBytes := make([]byte, SIG_RAND_NUM_BYTES)
	_, err := rand.Read(randBytes)
	if err != nil {return err }

	// r, s, err := ecdsa.Sign(randBytes, )
	// if err != nil {return err }

	return nil
}

func SignOcResp(resp *OcResp) error {
	return nil
}

func VerifyOcReq(req *OcReq) bool {
	return false
}

func VerifyOcResp(resp *OcResp) bool {
	return false
}
