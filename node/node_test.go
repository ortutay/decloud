package node

import (
	"fmt"
	"log"
	"net"
	"github.com/ortutay/decloud/cred"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/services/calc"
	"testing"
)

var _ = fmt.Printf

func newClient() (*Client, error) {
	ocCred, err := cred.NewOcCred()
	if err != nil {
		return nil, err
	}
	c := Client{
		Cred: &cred.Cred{
			Signers: []cred.Signer{ocCred},
		},
	}
	return &c, nil
}

func TestRoundTrip(t *testing.T) {
	addr := ":9443"
	handler := calc.CalcService{}
	s := Server{
		Cred:    &cred.Cred{},
		Addr:    addr,
		Handler: handler,
	}
	listener, err := net.Listen("tcp", s.Addr)
	defer listener.Close()
	if err != nil {
		log.Fatal(err)
	}
	go s.Serve(listener)

	c, err := newClient()
	if err != nil {
		log.Fatal(err)
	}

	req := calc.NewCalcReq([]string{"1 2 +"})
	resp, err := c.SendRequest(addr, req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("resp: %v\nbody: %v\n", resp, string(resp.Body))

	if resp.Status != msg.OK {
		t.Errorf("expected OK, but got: %v\n", resp.Status)
	}
}

func TestPaymentRequired(t *testing.T) {
	addr := ":9443"
	handler := calc.CalcService{}
	s := Server{
		Cred:    &cred.Cred{},
		Addr:    addr,
		Handler: handler,
		Policies: []Policy{
			Policy{
				Selector: GLOBAL,
				Cmd:      MIN_FEE,
				Args:     []interface{}{msg.PaymentValue{1, msg.BTC}},
			},
		},
	}
	listener, err := net.Listen("tcp", s.Addr)
	defer listener.Close()
	if err != nil {
		log.Fatal(err)
	}
	go s.Serve(listener)

	c, err := newClient()
	if err != nil {
		log.Fatal(err)
	}
	req := calc.NewCalcReq([]string{"1 2 +"})
	resp, err := c.SendRequest(addr, req)
	if resp.Status != msg.PAYMENT_REQUIRED {
		t.Errorf("expected PAYMENT_REQUIRED, but got: %v\n", resp.Status)
	}
}

func TestPaymentRoundTrip(t *testing.T) {
	addr := ":9443"
	handler := calc.CalcService{}
	s := Server{
		Cred:    &cred.Cred{},
		Addr:    addr,
		Handler: handler,
	}
	listener, err := net.Listen("tcp", s.Addr)
	defer listener.Close()
	if err != nil {
		log.Fatal(err)
	}
	go s.Serve(listener)

	c, err := newClient()
	if err != nil {
		log.Fatal(err)
	}
	calcReq := calc.NewCalcReq([]string{"1 2 +"})
	work, err := calc.Measure(calcReq)
	if err != nil {
		log.Fatal(err)
	}
	quoteReq := calc.NewQuoteReq(work)
	resp, err := c.SendRequest(addr, quoteReq)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("resp: %v\nbody: %v\n", resp, string(resp.Body))
}
