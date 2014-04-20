package store

import (
	"github.com/ortutay/decloud/conf"
	"github.com/ortutay/decloud/msg"
)

const (

	SERVICE_NAME = "store"

	// TODO(ortutay): quote is not really a method of the service; may want to
	// put it in a separate package
	// [method] [method args...]
	QUOTE_METHOD = "quote"

	// [blob-id] [size] [time]
	ALLOC_METHOD = "alloc"

	// [blob-id] [block-indexes] body: [block-data]
	PUT_METHOD = "put"

	// [blob-id] [block-indexs]
	GET_METHOD = "get"

	// [blob-id] [block-indexs] [salt]
	HASH_METHOD = "hash"
)

const BYTES_PER_BLOCK = 4096

type WorkPut struct {
	Blocks  int `json:"blocks"`
	Seconds int `json:"seconds"`
}

type WorkGet struct {
	Blocks int `json:"blocks"`
}

type WorkHash struct {
	Blocks int `json:"blocks"`
}

func MeasurePut(req *msg.OcReq) (*WorkPut, error) {
	return nil, nil
}

func MeasureGet(req *msg.OcReq) (*WorkGet, error) {
	return nil, nil
}

func MeasureHash(req *msg.OcReq) (*WorkHash, error) {
	return nil, nil
}

func (wp *WorkPut) Quote() *msg.PaymentValue {
	return nil
}

func (wp *WorkGet) Quote() *msg.PaymentValue {
	return nil
}

func (wp *WorkHash) Quote() *msg.PaymentValue {
	return nil
}

func NewPutReq() *msg.OcReq {
	return nil
}

func NewGetReq() *msg.OcReq {
	return nil
}

func NewHashReq() msg.OcReq {
	return nil
}

func Init() {
	quote.Register("store", "put", MeasurePut, QuotePut)
}

type StoreService struct {
	Conf *conf.Conf
}

func (ss *StoreService) Handle(req *msg.OcReq) (*msg.OcResp, error) {
	methods := make(map[string]func(*msg.OcReq) (*msg.OcResp, error))
	return nil, nil
}

func (ss *StoreService) quote(req *msg.OcReq) (*msg.OcResp, error) {
	return nil, nil
}

func (ss *StoreService) alloc(req *msg.OcReq) (*msg.OcResp, error) {
	return nil, nil
}

func (ss *StoreService) put(req *msg.OcReq) (*msg.OcResp, error) {
	return nil, nil
}

func (ss *StoreService) diff(req *msg.OcReq) (*msg.OcResp, error) {
	return nil, nil
}

func (ss *StoreService) get(req *msg.OcReq) (*msg.OcResp, error) {
	return nil, nil
}

func (ss *StoreService) hash(req *msg.OcReq) (*msg.OcResp, error) {
	return nil, nil
}
