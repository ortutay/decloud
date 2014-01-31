package node

import (
	"oc/cred"
	"oc/msg"
	"oc/services/calc"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	addr := ":9443"
	handler := calc.CalcService{}
	s := Server{
		Cred:    &cred.Cred{},
		Addr:    addr,
		Handler: handler,
	}
	go (func() {
		err := s.ListenAndServe()
		if err != nil {
			t.Errorf(err.Error())
		}
	})()

	ocCred, err := cred.NewOcCred()
	if err != nil {
		t.Errorf("%v", err)
	}
	c := Client{
		Cred: &cred.Cred{
			Signers: []cred.Signer{ocCred},
		},
	}
	req := calc.NewCalcReq([]string{"1 2 +"})
	resp, err := c.SendRequest(addr, req)
	if err != nil {
		t.Errorf(err.Error())
	}
	if resp.Status != msg.OK {
		t.Errorf("expected OK, but got: %v\n", resp.Status)
	}
}
