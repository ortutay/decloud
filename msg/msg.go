package msg

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

var _ = fmt.Printf

type PaymentType string

const (
	NONE     PaymentType = "none"
	ATTACHED             = "attached"
	DEFER                = "defer"
)

type PaymentAddr struct {
	Currency Currency
	Addr     string
}

func (pa *PaymentAddr) ToString() string {
	// TODO(ortutay): figure out real wire format
	b, err := json.Marshal(pa)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func NewPaymentAddr(str string) (*PaymentAddr, error) {
	// TODO(ortutay): figure out real wire format
	var pa PaymentAddr
	err := json.Unmarshal([]byte(str), &pa)
	if err != nil {
		return nil, fmt.Errorf("couldn't create PaymentAddr from %v", str)
	} else {
		return &pa, nil
	}
}

type Currency string

const (
	BTC Currency = "BTC"
	USD          = "USD"
)

type PaymentValue struct {
	Amount   float64
	Currency Currency
}

func (pv *PaymentValue) ToString() string {
	// TODO(ortutay): figure out real wire format
	b, err := json.Marshal(pv)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func NewPaymentValue(str string) (*PaymentValue, error) {
	// TODO(ortutay): figure out real wire format
	var pv PaymentValue
	err := json.Unmarshal([]byte(str), &pv)
	if err != nil {
		return nil, fmt.Errorf("couldn't create PaymentValue from %v", str)
	} else {
		return &pv, nil
	}
}

// For now, all fields are string or []byte.
// TODO(ortutay): add types for these fields
type OcReq struct {
	NodeId       []string
	Sig          []string
	Nonce        string
	Service      string
	Method       string
	Args         []string
	PaymentType  PaymentType
	PaymentValue *PaymentValue
	PaymentTxn   string
	Body         []byte
}

func (r *OcReq) WriteSignablePortion(w io.Writer) error {
	w.Write([]byte(r.Nonce))
	w.Write([]byte(r.Service))
	w.Write([]byte(r.Method))
	for _, arg := range r.Args {
		w.Write([]byte(arg))
	}
	var s = string(r.PaymentType)
	w.Write([]byte(s))
	w.Write([]byte(r.PaymentTxn))
	w.Write(r.Body)
	return nil
}

// TODO(ortutay): WriteEncoded(w io.Writer)
func (r *OcReq) Encode() ([]byte, error) {
	return encode(r)
}

func (r *OcReq) IsSigned() bool {
	return len(r.Sig) > 0
}

func (r *OcReq) AttachDeferredPayment(pv *PaymentValue) {
	if r.PaymentType != "" || r.PaymentTxn != "" || r.PaymentValue != nil {
		panic("expected request with no payment")
	}
	pvCopy := PaymentValue(*pv)
	r.PaymentType = DEFER
	r.PaymentValue = &pvCopy
}

type OcRespStatus string

const (
	OK OcRespStatus = "ok"

	ACCESS_DENIED = "access-denied"

	CLIENT_ERROR        = "client-error"
	BAD_REQUEST         = CLIENT_ERROR + "/bad-request"
	INVALID_SIGNATURE   = CLIENT_ERROR + "/invalid-signature"
	SERVICE_UNSUPPORTED = CLIENT_ERROR + "/service-unsupported"
	METHOD_UNSUPPORTED  = CLIENT_ERROR + "/method-unsupported"

	SERVER_ERROR = "server-error"

	REQUEST_DECLINED     = "request-declined"
	REFRESH_NONCE        = REQUEST_DECLINED + "/refresh-nonce"
	CURRENCY_UNSUPPORTED = REQUEST_DECLINED + "/currency-unsupported"
	PAYMENT_REQUIRED     = REQUEST_DECLINED + "/payment-required"
	PAYMENT_DECLINED     = REQUEST_DECLINED + "/payment-declined"
	TOO_LOW              = PAYMENT_DECLINED + "/too-low"
	NO_DEFER             = PAYMENT_DECLINED + "/no-defer"
)

type OcResp struct {
	NodeId []string
	Sig    []string
	Nonce  string
	Status OcRespStatus
	// TODO(ortutay): status code
	Body []byte
}

func NewRespOk(body []byte) *OcResp {
	resp := OcResp{
		NodeId: []string{},
		Sig:    []string{},
		Nonce:  "", // TODO(ortutay)
		Status: OK,
		Body:   body,
	}
	return &resp
}

func NewRespError(status OcRespStatus) *OcResp {
	if status == OK {
		panic("got status OK, but expected an error status")
	}

	resp := OcResp{
		NodeId: []string{},
		Sig:    []string{},
		Nonce:  "", // TODO(ortutay)
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
