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
	println("send req")
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
		Conf: conf.Conf{
			Policies: []conf.Policy{
				conf.Policy{
					Selector: conf.PolicySelector{},
					Cmd:      conf.MIN_FEE,
					Args:     []interface{}{msg.PaymentValue{1, msg.BTC}},
				},
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
	services[calc.SERVICE_NAME] = calc.CalcService{
		Conf: conf.Conf{
			Policies: []conf.Policy{
				conf.Policy{
					Selector: conf.PolicySelector{
						Service: calc.SERVICE_NAME,
						Method:  calc.CALCULATE_METHOD,
					},
					Cmd:  conf.MIN_FEE,
					Args: []interface{}{msg.PaymentValue{2, msg.BTC}},
				},
			},
		},
	}
	services[info.SERVICE_NAME] = &info.InfoService{BitcoindConf: btcConf}
	mux := ServiceMux{
		Services: services,
	}
	s := Server{
		Cred:    &cred.Cred{},
		Addr:    addr,
		Handler: &mux,
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
	fmt.Printf("quote\n")
	calcReq := calc.NewCalcReq([]string{"1 2 +"})
	work, err := calc.Measure(calcReq)
	if err != nil {
		log.Fatal(err)
	}
	quoteReq := calc.NewQuoteReq(work)
	fmt.Printf("quote req: %v\n", quoteReq)
	resp, err := c.SendRequest(addr, quoteReq)
	if err != nil {
		log.Fatal(err)
	}
	pv, err := msg.NewPaymentValue(string(resp.Body))
	fmt.Printf("resp: %v\nbody: %v\n", resp, string(resp.Body))
	if err != nil {
		log.Fatal(err)
	}

	// Get payment address
	go s.Serve(listener)
	fmt.Printf("get payment addr\n")
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

	// Send low payment
	go s.Serve(listener)
	fmt.Printf("send req with deferred payment")
	sendLowPv := msg.PaymentValue(*pv)
	sendLowPv.Amount -= 1
	calcReqLowPv := msg.OcReq(*calcReq)
	calcReqLowPv.AttachDeferredPayment(&sendLowPv)
	resp, err = c.SendRequest(addr, &calcReqLowPv)
	if resp.Status != msg.TOO_LOW {
		log.Fatalf("expected status %v, got %v", msg.TOO_LOW, resp.Status)
	}

	// Send requested payment
	go s.Serve(listener)
	fmt.Printf("send req with deferred payment")
	sendPv := msg.PaymentValue(*pv)
	calcReq.AttachDeferredPayment(&sendPv)
	resp, err = c.SendRequest(addr, calcReq)
	if resp.Status != msg.OK {
		log.Fatalf("expected status %v, got %v", msg.OK, resp.Status)
	}
}
