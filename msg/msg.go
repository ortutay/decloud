package msg

import (
	"errors"
	"bytes"
	"fmt"
	"encoding/gob"
	"encoding/base64"
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

func (r *OcReq) Encode() ([]byte, error) {
	return encode(r)
}

func (r *OcReq) IsSigned() bool {
	return len(r.Sig) > 0
}

type OcRespStatus string
const (
	OK OcRespStatus = "ok"

	CLIENT_ERROR = "client-error"
	BAD_REQUEST = CLIENT_ERROR + "/bad-request"
	INVALID_SIGNATURE = CLIENT_ERROR + "/invalid-signature"
	SERVICE_UNSUPPORTED = CLIENT_ERROR + "/service-unsupported"
	METHOD_UNSUPPORTED = CLIENT_ERROR + "/method-unsupported"

	SERVER_ERROR = "server-error"

	REQUEST_DECLINED = "request-declined"
	REFRESH_NONCE = REQUEST_DECLINED + "/refresh-nonce"
	PAYMENT_DECLINED = REQUEST_DECLINED + "/payment-declined"
	TOO_LOW = PAYMENT_DECLINED + "/too-low"
	NO_DEFER = PAYMENT_DECLINED + "/no-defer"
)

type OcResp struct {
	NodeId []string
	Sig []string
	Nonce string
	Status OcRespStatus
	// TODO(ortutay): status code
	Body []byte
}

func NewRespOk(body []byte) *OcResp {
	resp := OcResp{
		NodeId: []string{},
		Sig: []string{},
		Nonce: "", // TODO(ortutay)
		Status: OK,
		Body: body,
	}
	return &resp
}

func NewRespError(status OcRespStatus) *OcResp {
	if status == OK {
		panic("got status OK, but expected an error status")
	}

	resp := OcResp{
		NodeId: []string{},
		Sig: []string{},
		Nonce: "", // TODO(ortutay)
		Status: status,
	}
	return &resp
}

func (r *OcResp) Encode() ([]byte, error) {
	return encode(r)
}

func encode(m interface{}) ([]byte, error) {
	// TODO(ortutay): for now, just doing gob->base64 to encode; will need
	// to figure out what to actually do
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(m)
	if err != nil {
		return []byte{}, errors.New("couldn't encode")
	}
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	return []byte(b64), nil
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
