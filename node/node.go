package node

import (
	"bufio"
	"fmt"
	"net"
	"sort"

	"github.com/conformal/btcjson"
	"github.com/ortutay/decloud/conf"
	"github.com/ortutay/decloud/cred"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/util"
)

type Client struct {
	BtcConf *util.BitcoindConf
	Cred    cred.Cred
}

func (c *Client) SignAndSend(addr string, req *msg.OcReq) (*msg.OcResp, error) {
	err := c.SignRequest(req)
	if err != nil {
		return nil, err
	}
	return c.SendRequest(addr, req)
}

type AddressBalance struct {
	Address string
	Amount int64
}

type ByAmount []AddressBalance
func (a ByAmount) Len() int { return len(a) }
func (a ByAmount) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByAmount) Less(i, j int) bool { return a[i].Amount < a[j].Amount }

func (c *Client) BtcSignRequest(min, max int64, req *msg.OcReq) error {
	cmd, err := btcjson.NewListUnspentCmd("")
	if err != nil {
		return fmt.Errorf("error while making cmd: %v", err.Error())
	}
	resp, err := util.SendBtcRpc(cmd, c.BtcConf)
	if err != nil {
		return fmt.Errorf("error while making cmd: %v", err.Error())
	}
	if resp.Error != nil {
		return fmt.Errorf("error during bitcoind JSON-RPC: %v", resp.Error)
	}
	addrs := make(map[string]*AddressBalance)
	unspent := resp.Result.([]btcjson.ListUnSpentResult)
	for _, u := range unspent {
		if _, ok := addrs[u.Address]; !ok {
			addrs[u.Address] = &AddressBalance{
				Address: u.Address,
				Amount: 0,
			}
		}
		ab := addrs[u.Address]
		ab.Amount += util.B2S(u.Amount)
	}
	addrsList := make([]AddressBalance, len(addrs))
	i := 0
	for _, v := range addrs {
		addrsList[i] = *v
		i++
	}
	sort.Sort(ByAmount(addrsList))
	var use *[]AddressBalance
	for iter := 1; iter <= 5; iter++ {
		use, err = inputsInRange(&addrsList, min, max, iter, len(addrsList) - 1)
		if use != nil {
			break
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func inputsInRange(unspent *[]AddressBalance, min, max int64, iter int, right int) (*[]AddressBalance, error) {
	if iter == 0 {
		return nil, fmt.Errorf("couldn't find matching inputs")
	}
	// Assume list is already sorted
	for i := right; i >= 0; i-- {
		u := (*unspent)[i]
		amt := u.Amount
		if amt > max {
			continue;
		}
		if iter == 1 {
			if amt >= min {
				return &[]AddressBalance{u}, nil
			}
		} else {
			r, _ := inputsInRange(unspent, min - amt, max - amt, iter - 1, i - 1)
			if r != nil {
				result := append(*r, u)
				return &result, nil
			}
		}
	}
	return nil, fmt.Errorf("couldn't find matching inputs")
}

func (c *Client) SignRequest(req *msg.OcReq) error {
	// TODO(ortutay): add nonce support
	err := c.Cred.SignOcReq(req, c.BtcConf)
	if err != nil {
		return fmt.Errorf("error while signing: %v", err.Error())
	}
	return nil
}

func (c *Client) SendRequest(addr string, req *msg.OcReq) (*msg.OcResp, error) {
	if req.Nonce != "" {
		// TODO(ortutay): add nonce support
		panic("expected no nonce")
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("error while dialing: %v", err.Error())
	}

	err = req.Write(conn)
	if err != nil {
		return nil, fmt.Errorf("error while writing to conn: %v", err.Error())
	}

	resp, err := msg.ReadOcResp(bufio.NewReader(conn))
	if err != nil {
		return nil, fmt.Errorf("error while reading: %v", err.Error())
	}

	return resp, nil
}

func (c *Client) SendBtcPayment(payVal *msg.PaymentValue, payAddr *msg.PaymentAddr) (msg.BtcTxid, error) {
	if payVal.Currency != msg.BTC || payAddr.Currency != msg.BTC {
		panic("unexpected currency: " + payVal.Currency + " " + payAddr.Currency)
	}
	cmd, err := btcjson.NewSendToAddressCmd("", payAddr.Addr, payVal.Amount)
	if err != nil {
		return "", fmt.Errorf("error while making cmd: %v", err.Error())
	}
	resp, err := util.SendBtcRpc(cmd, c.BtcConf)
	if err != nil {
		return "", fmt.Errorf("error while making cmd: %v", err.Error())
	}
	txid, ok := resp.Result.(string)
	if !ok {
		return "", fmt.Errorf("error during bitcoind JSON-RPC: %v", resp)
	}
	return msg.BtcTxid(txid), nil
}

type Handler interface {
	Handle(*msg.OcReq) (*msg.OcResp, error)
}

type ServiceMux struct {
	Services map[string]Handler
}

func (sm *ServiceMux) Handle(req *msg.OcReq) (*msg.OcResp, error) {
	if service, ok := sm.Services[req.Service]; ok {
		return service.Handle(req)
	} else {
		return msg.NewRespError(msg.SERVICE_UNSUPPORTED), nil
	}
}

type Server struct {
	Cred    *cred.Cred
	Addr    string
	Conf    *conf.Conf
	Handler Handler
}

func (s *Server) ListenAndServe() error {
	fmt.Printf("listening on %s\n", s.Addr)
	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return fmt.Errorf("couldn't listen on %s: %s", s.Addr, err.Error())
	}
	defer listener.Close()
	for {
		err := s.Serve(listener)
		if err != nil {
			fmt.Printf("error accepting: %v", err)
			continue
		}
	}
	return nil
}

func (s *Server) Serve(listener net.Listener) error {
	conn, err := listener.Accept()
	if err != nil {
		return err
	}
	go (func(conn net.Conn) {
		println("get req")
		req, err := msg.ReadOcReq(bufio.NewReader(conn))
		defer conn.Close()
		defer fmt.Fprintf(conn, "\n")
		if err != nil {
			msg.NewRespError(msg.BAD_REQUEST).Write(conn)
			return
		}

		fmt.Printf("Got request: %v\n", req)

		// TODO(ortutay): implement additional request validation
		// - validate sigs
		// - check nonce
		// - check service available
		// - check method available

		if ok, status := s.isAllowedByPolicy(req); !ok {
			if status == msg.OK {
				panic("expected error status")
			}
			fmt.Printf("not allowed: %v %v\n", ok, status)
			msg.NewRespError(status).Write(conn)
			return
		}

		resp, err := s.Handler.Handle(req)
		if err != nil {
			msg.NewRespError(msg.SERVER_ERROR).Write(conn)
		} else {
			fmt.Printf("sending response: %v\n", resp)
			resp.Write(conn)
		}
		return
	})(conn)
	return nil
}

func (s *Server) isAllowedByPolicy(req *msg.OcReq) (bool, msg.OcRespStatus) {
	fmt.Printf("is allowed? %v\n", s)
	// policies := s.Conf.MatchingPolicies(req.Service, req.Method)
	// for _, policy := range policies {
	// 	fmt.Printf("check against policy: %v\n", policy)
	// 	switch policy.Cmd {
	// 	case conf.ALLOW:
	// 		continue
	// 	case conf.DENY:
	// 		return false, msg.ACCESS_DENIED
	// 	case conf.MIN_FEE:
	// 		min := policy.Args[0].(msg.PaymentValue)
	// 		fmt.Printf("min fee: %v, pt: %v\n", min, req.PaymentType)
	// 		if req.PaymentType != msg.ATTACHED {
	// 			return false, msg.PAYMENT_REQUIRED
	// 		}

	// 		// TODO(ortutay): implement
	// 		return false, msg.SERVER_ERROR
	// 	}
	// }
	return true, msg.OK
}
