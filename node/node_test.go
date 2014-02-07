package node

import (
	"fmt"
	"github.com/ortutay/decloud/conf"
	"github.com/ortutay/decloud/cred"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/services/calc"
	"github.com/ortutay/decloud/services/info"
	"github.com/ortutay/decloud/util"
	"log"
	"net"
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

// TODO(ortutay): rm dupe code
func printBitcoindExpected() {
	println("Note: bitcoind daemon expected to be running")
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
		Policies: []conf.Policy{
			conf.Policy{
				Selector: conf.GLOBAL,
				Cmd:      conf.MIN_FEE,
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
	printBitcoindExpected()
	btcConf, err := util.LoadBitcoindConf("")
	if err != nil {
		t.Errorf("%v", err)
	}

	addr := ":9443"
	services := make(map[string]Handler)
	services[calc.SERVICE_NAME] = calc.CalcService{}
	services[info.SERVICE_NAME] = &info.InfoService{BitcoindConf: btcConf}
	mux := ServiceMux{
		Services: services,
	}
	s := Server{
		Cred:    &cred.Cred{},
		Addr:    addr,
		Handler: &mux,
		Policies: []conf.Policy{
			conf.Policy{
				Selector: conf.GLOBAL,
				Cmd:      conf.MIN_FEE,
				Args:     []interface{}{msg.PaymentValue{1, msg.BTC}},
			},
		},
	}
	listener, err := net.Listen("tcp", s.Addr)
	defer listener.Close()
	if err != nil {
		log.Fatal(err)
	}

	c, err := newClient()
	if err != nil {
		log.Fatal(err)
	}

	// Quote
	go s.Serve(listener)
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
	pv, err := msg.NewPaymentValue(string(resp.Body))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("resp: %v\npv: %v\n", resp, pv)

	// Get payment address
	go s.Serve(listener)
	payAddrReq := info.NewPaymentAddrReq(msg.BTC)
	fmt.Printf("req: %v\n", payAddrReq)
	resp, err = c.SendRequest(addr, payAddrReq)
	if err != nil {
		log.Fatal(err)
	}
	pa, err := msg.NewPaymentAddr(string(resp.Body))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("resp: %v\npa: %v\n", resp, pa)

	// Send request with defered payment
	calcReq.AttachDeferredPayment(pv)
	fmt.Printf("req with deferred payment: %v\n", calcReq)

	// TODO: submit request, and then fulfill deferred payment
}
