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
