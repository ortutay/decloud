package msg

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

var _ = fmt.Printf

type BtcTxid string
type PaymentType string

const (
	NONE     PaymentType = "none"
	TXID                 = "txid"
	ATTACHED             = "attached"
	DEFER                = "defer"
)

type PaymentAddr struct {
	Currency Currency `json:"currency"`
	Addr     string   `json:"addr"`
}

func (pa *PaymentAddr) String() string {
	b, err := json.Marshal(pa)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func NewPaymentAddr(str string) (*PaymentAddr, error) {
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
	Amount   int64    `json:"amount"`
	Currency Currency `json:"currency"`
}

func (pv *PaymentValue) String() string {
	b, err := json.Marshal(pv)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func NewPaymentValue(str string) (*PaymentValue, error) {
	var pv PaymentValue
	err := json.Unmarshal([]byte(str), &pv)
	if err != nil {
		return nil, fmt.Errorf("couldn't create PaymentValue from %v", str)
	} else {
		return &pv, nil
	}
}

// TODO(ortutay): add types as appropriate
type OcReq struct {
	Id            []string        `json:"id,omitempty"`
	Sig           []string        `json:"sig,omitempty"`
	Nonce         string          `json:"nonce,omitempty"`
	Service       string          `json:"service"`
	Method        string          `json:"method"`
	Args          json.RawMessage `json:"args,omitempty"`
	PaymentType   PaymentType     `json:"paymentType,omitempty"`
	PaymentValue  *PaymentValue   `json:"paymentValue,omitempty"`
	PaymentTxn    string          `json:"paymentTxn,omitempty"`
	ContentLength int             `json:"contentLength,omitempty"`
	Body          []byte          `json:"-"`
}

func (r *OcReq) WriteSignablePortion(w io.Writer) error {
	w.Write([]byte(r.Nonce))
	w.Write([]byte(r.Service))
	w.Write([]byte(r.Method))
	// for _, arg := range r.Args {
	// 	w.Write([]byte(arg))
	// }
	var s = string(r.PaymentType)
	w.Write([]byte(s))
	w.Write([]byte(r.PaymentTxn))
	w.Write(r.Body)
	return nil
}

func (r *OcReq) Write(w io.Writer) error {
	return writeMsg(r, r.Body, w)
}

func ReadOcReq(r *bufio.Reader) (*OcReq, error) {
	jsonLine, err := r.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("error while reading JSON line: %v", err.Error())
	}
	var req OcReq
	err = json.Unmarshal(jsonLine, &req)
	if err != nil {
		return nil, fmt.Errorf("error while unmarshalling: %v", err.Error())
	}
	if req.ContentLength > 0 {
		req.Body = make([]byte, req.ContentLength)
		_, err := io.ReadFull(r, req.Body)
		if err != nil {
			return nil, fmt.Errorf("error while reading body: %v", err.Error())
		}
	}
	return &req, nil
}

func (r *OcReq) String() string {
	return msgString(r, r.Body)
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
	INVALID_ARGUMENTS   = CLIENT_ERROR + "/invalid-arguments"

	SERVER_ERROR = "server-error"

	REQUEST_DECLINED     = "request-declined"
	REFRESH_NONCE        = REQUEST_DECLINED + "/refresh-nonce"
	CURRENCY_UNSUPPORTED = REQUEST_DECLINED + "/currency-unsupported"
	PAYMENT_REQUIRED     = REQUEST_DECLINED + "/payment-required"
	PAYMENT_DECLINED     = REQUEST_DECLINED + "/payment-declined"
	INVALID_TXN          = PAYMENT_DECLINED + "/invalid-transaction"
	INVALID_TXID         = PAYMENT_DECLINED + "/invalid-txid"
	TOO_LOW              = PAYMENT_DECLINED + "/too-low"
	NO_DEFER             = PAYMENT_DECLINED + "/no-defer"
)

type OcResp struct {
	Id     []string     `json:"id,omitempty"`
	Sig    []string     `json:"sig,omitempty"`
	Nonce  string       `json:"nonce,omitempty"`
	Status OcRespStatus `json:"status,omitempty"`
	// TODO(ortutay): status code
	ContentLength int    `json:"contentLength,omitempty"`
	Body          []byte `json:"-"`
}

func NewRespOk(body []byte) *OcResp {
	resp := OcResp{
		Id:            []string{},
		Sig:           []string{},
		Nonce:         "", // TODO(ortutay)
		Status:        OK,
		ContentLength: len(body),
		Body:          body,
	}
	return &resp
}

func NewRespError(status OcRespStatus) *OcResp {
	if status == OK {
		panic("got status OK, but expected an error status")
	}
	resp := OcResp{
		Id:     []string{},
		Sig:    []string{},
		Nonce:  "", // TODO(ortutay)
		Status: status,
	}
	return &resp
}

func (r *OcResp) Write(w io.Writer) error {
	return writeMsg(r, r.Body, w)
}

func ReadOcResp(r *bufio.Reader) (*OcResp, error) {
	// TODO(ortutay): shared header that inclues ContentLength
	jsonLine, err := r.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("error while reading JSON line: %v", err.Error())
	}
	var resp OcResp
	err = json.Unmarshal(jsonLine, &resp)
	if err != nil {
		return nil, fmt.Errorf("error while unmarshalling: %v", err.Error())
	}
	if resp.ContentLength > 0 {
		resp.Body = make([]byte, resp.ContentLength)
		_, err := io.ReadFull(r, resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error while reading body: %v", err.Error())
		}
	}
	return &resp, nil
}

func (r *OcResp) String() string {
	return msgString(r, r.Body)
}

func writeMsg(v interface{}, body []byte, w io.Writer) error {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("Error while marshaling to json: %v", err.Error())
	}
	_, err = w.Write(b)
	if err != nil {
		return fmt.Errorf("Error while writing: %v", err.Error())
	}
	_, err = w.Write([]byte("\n"))
	if err != nil {
		return fmt.Errorf("Error while writing: %v", err.Error())
	}
	if len(body) > 0 {
		_, err = w.Write(body)
		if err != nil {
			return fmt.Errorf("Error while writing: %v", err.Error())
		}
	}
	return nil
}

func msgString(v interface{}, body []byte) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	var buf bytes.Buffer
	err = json.Indent(&buf, b, "", "  ")
	if err != nil {
		return ""
	}
	if len(body) > 0 {
		return fmt.Sprintf("%s\n%s", buf.String(), string(body))
	} else {
		return fmt.Sprintf("%s\n", buf.String())
	}
}
