package msg

import (
	"fmt"
	"log"
	"encoding/gob"
	"encoding/base64"
	"bytes"
	"oc/util"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/elliptic"
)
var _ = fmt.Printf

// For simplicity, all fields are string or []byte
type OcReq struct {
	NodeId string
	Nonce string
	Sig string
	Service string
	Method string
	Args []string
	PaymentType string
	PaymentTxn string
	Body []byte
}

type OcResp struct {
	NodeId string
	Sig string
	Status string
	// TODO(ortutay): status code
	Body []byte
}

func NewRespOk(body []byte) *OcResp {
	resp := OcResp{
		NodeId: "TODO",
		Sig: "TODO",
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

// functions related to node ID/signing; may want these in a different package
func MakeNodeId(destPath string) error {
	b := make([]byte, 256)
	_, err := rand.Read(b)
	if err != nil {
		return err
	}
	curve := elliptic.P256()
	priv, err := ecdsa.GenerateKey(curve, bytes.NewReader(b))
	if err != nil {
		return err
	}
	h := fmt.Sprintf("%x\n", priv.D)
	if destPath == "" { destPath = "nodeid-priv" }
	err = util.StoreAppData(destPath, []byte(h), 0600)
	if err != nil {
		return err
	}
	pub := elliptic.Marshal(curve, priv.PublicKey.X, priv.PublicKey.Y)
	println(fmt.Sprintf("%v\n", pub))
	return nil
}
