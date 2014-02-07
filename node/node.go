package node

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"github.com/ortutay/decloud/cred"
	"github.com/ortutay/decloud/msg"
)

type Client struct {
	Cred *cred.Cred
}

func (c *Client) SendRequest(addr string, req *msg.OcReq) (*msg.OcResp, error) {
	if req.IsSigned() {
		// TODO(ortutay): not sure if this needs to be a panic; also may want to
		// rethink the general structure
		panic(fmt.Sprintf("expected to sign the request: %v", req.Sig))
	}
	if req.Nonce != "" {
		// TODO(ortutay): same as above
		panic("expected no nonce")
	}

	// TODO(ortutay): add nonce

	err := c.Cred.SignOcReq(req)
	if err != nil {
		return nil, fmt.Errorf("error while signing: %v", err.Error())
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("error while dialing: %v", err.Error())
	}

	encoded, err := req.Encode()
	if err != nil {
		return nil, fmt.Errorf("error while encoding: %v", err.Error())
	}

	_, err = fmt.Fprintf(conn, string(encoded)+"\n")
	if err != nil {
		return nil, fmt.Errorf("error while writing to conn: %v", err.Error())
	}

	b64resp, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("error while reading: %v", err.Error())
	}
	println("got resp")

	resp, err := msg.DecodeResp(b64resp)
	if err != nil {
		return nil, fmt.Errorf("error while decoding %v: %v", b64resp, err.Error())
	}

	return resp, nil
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
	Cred     *cred.Cred
	Addr     string
	Policies []Policy
	Handler  Handler
}

type PolicyCmd string

const (
	ALLOW   PolicyCmd = "allow"
	DENY              = "deny"
	MIN_FEE           = "min-fee"
	// TODO(ortutay): add rate-limit
	// TODO(ortutay): additional policy commands
)

// TODO(ortutay): implement real selectors; PolicySelector is just a placeholder
type PolicySelector string

const (
	GLOBAL PolicySelector = "global"
)

type Policy struct {
	Selector PolicySelector
	Cmd      PolicyCmd
	Args     []interface{}
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
		req, err := readOcReq(conn)
		defer conn.Close()
		defer fmt.Fprintf(conn, "\n")
		if err != nil {
			writeErrorResp(msg.BAD_REQUEST, conn)
			return
		}

		// TODO(ortutay): implement additional request validation
		// - validate sigs
		// - check nonce
		// - check service available
		// - check method available

		if ok, status := s.IsAllowedByPolicy(req); !ok {
			fmt.Printf("not allowed: %v %v\n", ok, status)
			writeErrorResp(status, conn)
			return
		}

		resp, err := s.Handler.Handle(req)
		if err != nil {
			writeErrorResp(msg.SERVER_ERROR, conn)
		} else {
			writeResp(resp, conn)
		}
		return
	})(conn)
	return nil
}

func (s *Server) IsAllowedByPolicy(req *msg.OcReq) (bool, msg.OcRespStatus) {
	fmt.Printf("is allowed? %v\n", s)
	for _, policy := range s.Policies {
		fmt.Printf("check against policy: %v\n", policy)
		switch policy.Cmd {
		case ALLOW:
			continue
		case DENY:
			return false, msg.ACCESS_DENIED
		case MIN_FEE:
			min := policy.Args[0].(msg.PaymentValue)
			fmt.Printf("min fee: %v, pt: %v\n", min, req.PaymentType)
			if req.PaymentType != msg.ATTACHED {
				return false, msg.PAYMENT_REQUIRED
			}

			// TODO(ortutay): implement
			return false, msg.SERVER_ERROR
		}
	}
	return true, msg.OK
}

func readOcReq(conn net.Conn) (*msg.OcReq, error) {
	println("read req")
	b64, err := bufio.NewReader(conn).ReadString('\n')
	fmt.Printf("did read req %v", b64)
	if err != nil {
		return nil, fmt.Errorf("could not read request")
	}
	req, err := msg.DecodeReq(b64)
	if err != nil {
		return nil, fmt.Errorf("could not parse request")
	}
	return req, nil
}

func writeErrorResp(status msg.OcRespStatus, w io.Writer) {
	if status == msg.OK {
		panic("got status OK, but expected an error status")
	}

	resp := msg.NewRespError(status)
	writeResp(resp, w)
}

func writeResp(resp *msg.OcResp, w io.Writer) {
	encoded, err := resp.Encode()
	if err != nil {
		fmt.Printf("couldn't encode %v: %v\n", resp, err.Error())
		return
	}

	_, err = w.Write(encoded)
	if err != nil {
		fmt.Printf("couldn't write encoded response %v: %v\n", encoded, err.Error())
		return
	}
}
