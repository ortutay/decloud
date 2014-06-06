package proxy

import (
	"fmt"
	
	"github.com/ortutay/decloud/conf"
	"github.com/ortutay/decloud/msg"
)

const (
	SERVICE_NAME     = "proxy"
	PROXY_METHOD = "proxy"
)

func NewProxyReq(body string) *msg.OcReq {
	msg := msg.OcReq{
		ID:            "",
		Sig:           "",
		Coins:         []string{},
		CoinSigs:      []string{},
		Nonce:         "",
		Service:       SERVICE_NAME,
		Method:        PROXY_METHOD,
		Args:          []string{},
		PaymentType:   "",
		PaymentTxn:    "",
		ContentLength: 0,
		Body:          []byte(""),
	}
	if len(body) > 0 {
		msg.SetBody([]byte(body))
	}
	return &msg
}

type ProxyService struct {
	Conf *conf.Conf
}

func (ps *ProxyService) Handle(req *msg.OcReq) (*msg.OcResp, error) {
	println(fmt.Sprintf("proxy got request: %v", req))
	if req.Service != SERVICE_NAME {
		panic(fmt.Sprintf("unexpected service %s", req.Service))
	}

	methods := make(map[string]func(*msg.OcReq) (*msg.OcResp, error))
	methods[PROXY_METHOD] = ps.proxy

	if method, ok := methods[req.Method]; ok {
		return method(req)
	} else {
		return msg.NewRespError(msg.METHOD_UNSUPPORTED), nil
	}
}

func (ps *ProxyService) proxy(req *msg.OcReq) (*msg.OcResp, error) {
	resp := msg.NewRespOk([]byte("ok/placeholder"))
	return resp, nil
}
