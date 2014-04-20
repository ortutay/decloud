package node

import (
	"bufio"
	"fmt"
	"net"

	"github.com/conformal/btcjson"
	"github.com/ortutay/decloud/conf"
	"github.com/ortutay/decloud/cred"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/peer"
	"github.com/ortutay/decloud/util"
)

const SERVER_PAYMENT_MIN_CONF = 0
const SERVER_MAX_BALANCE = 1e6 // 1 BTC

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

func (c *Client) BtcSignRequest(min, max int64, req *msg.OcReq) error {
	return nil
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
	BtcConf *util.BitcoindConf
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

		p, err := peer.NewPeerFromReq(req, s.BtcConf)
		if err != nil {
			msg.NewRespError(msg.SERVER_ERROR).Write(conn)
		}

		// TODO(ortutay): more configuration options around allowed balance
		balance, err := p.Balance(SERVER_PAYMENT_MIN_CONF, s.BtcConf)
		if err != nil {
			msg.NewRespError(msg.SERVER_ERROR).Write(conn)
		}
		fmt.Printf("balance: %v\n", balance)
		if (balance.Currency != msg.BTC) {
			panic("TODO: support other currencies")
		}
		if balance.Amount > SERVER_MAX_BALANCE {
			addr, err := p.PaymentAddr(-1, s.BtcConf)
			if err != nil {
				msg.NewRespError(msg.SERVER_ERROR).Write(conn)
			}
			body := fmt.Sprintf("Balance due. Please pay %v %v to %v\n",
				util.S2B(balance.Amount - SERVER_MAX_BALANCE),
				balance.Currency.String(),
				addr)
			msg.NewRespErrorWithBody(msg.PLEASE_PAY, []byte(body)).Write(conn)
		}

		if ok, status := s.isAllowedByPolicy(p, req); !ok {
			if status == msg.OK {
				panic("expected error status")
			}
			fmt.Printf("not allowed: %v %v\n", ok, status)
			msg.NewRespError(status).Write(conn)
			return
		}

		fmt.Printf("passing off to handler...\n")
		resp, err := s.Handler.Handle(req)
		if err != nil {
			fmt.Printf("server error: %v\n", err)
			msg.NewRespError(msg.SERVER_ERROR).Write(conn)
		} else {
			fmt.Printf("sending response: %v\n", resp)
			resp.Write(conn)
		}
		return
	})(conn)
	return nil
}

func (s *Server) isAllowedByPolicy(p *peer.Peer, req *msg.OcReq) (bool, msg.OcRespStatus) {
	fmt.Printf("is allowed? %v\n", s)

	paidPv, err := p.AmountPaid(SERVER_PAYMENT_MIN_CONF, s.BtcConf)
	if err != nil {
		return false, msg.SERVER_ERROR
	}
	consumedPv, err := p.AmountConsumed()
	if err != nil {
		return false, msg.SERVER_ERROR
	}
	fmt.Printf("paid: %v, consumed: %v\n", paidPv, consumedPv)

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
