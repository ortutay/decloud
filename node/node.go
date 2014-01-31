package node

import (
	"io"
	"fmt"
	"bufio"
	"oc/cred"
	"oc/msg"
	"net"
)


type Client struct {
	Cred *cred.Cred
}

func (c *Client) SendRequest(addr string, req *msg.OcReq) (*msg.OcResp, error) {
	if req.IsSigned() {
		// TODO(ortutay): not sure if this needs to be a panic; also may want to
		// rethink the structure around this
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
	fmt.Fprintf(conn, string(encoded) + "\n")

	b64resp, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("error while reading: %v", err.Error())
	}

	resp, err := msg.DecodeResp(b64resp)
	if err != nil {
		return nil, fmt.Errorf("error while decoding %v: %v", b64resp, err.Error())
	}

	return resp, nil
}

type Handler interface {
	Handle(*msg.OcReq) (*msg.OcResp, error)
}

type Server struct {
	Cred *cred.Cred
	Addr string
	Handler Handler
}


func (s *Server) ListenAndServe() error {
	fmt.Printf("listening on %s\n", s.Addr)
	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return fmt.Errorf("couldn't listen on %s: %s", s.Addr, err.Error())
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			println("error accepting: ", err.Error())
			continue
		}
		go (func(conn net.Conn) {
			req, err := readOcReq(conn)
			if err != nil {
				writeErrorResp(msg.BAD_REQUEST, conn)
			} else {
				resp, err := s.Handler.Handle(req)
				if err != nil {
					writeErrorResp(msg.SERVER_ERROR, conn)
				} else {
					writeResp(resp, conn)
				}
			}
			fmt.Fprintf(conn, "\n")
		})(conn)
	}
	return nil
}

func readOcReq(conn net.Conn) (*msg.OcReq, error) {
	b64, err := bufio.NewReader(conn).ReadString('\n')
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
