package sellerd

import (
	"net"
	"os"
	"bufio"
	"oc/msg"
	"oc/seller/calc"
	"fmt"
)
var _ = fmt.Printf

const (
	BUF_LEN = 1024
)

func Listen() {
	println("starting server")
	listener, err := net.Listen("tcp", "localhost:9443")
	if err != nil {	
	println("error listening: ", err.Error())
		os.Exit(1)
	}

	for {
		println("listening for new connection...")
		conn, err := listener.Accept()
		if err != nil {
			println("error accepting: ", err.Error())
			return
		}
		go handle(conn)
	}
}


func handle(conn net.Conn) {
	b64, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		println("error reading: ", err.Error())
		writeError(conn, "bad request")
		return
	}
	req, err := msg.DecodeReq(b64)
	if err != nil {
		println("error reading: ", err.Error())
		writeError(conn, "bad request")
		return
	}

	switch req.Service {
	case "calc":
		resp, err := calc.Handle(req)
		if err != nil {
			writeError(conn, err.Error())
		}
		writeResp(conn, resp)
	default:
		writeError(conn, "unhandled service: " + req.Service)
	}
	
	_, err = conn.Write([]byte("resp\n"))
	// send resp
	if err != nil {
		println("error when sending reply: ", err.Error())
	} else {
		println("reply sent")
	}
}

func writeResp(conn net.Conn, resp *msg.OcResp) {
	_, err := conn.Write([]byte(msg.EncodeResp(resp)))
	_, err = conn.Write([]byte("\n"))
	if err != nil {
		println("error when sending reply: ", err.Error())
	} else {
		println("reply sent")
	}
}

func writeError(conn net.Conn, errMsg string) {
	// TODO(ortutay): write real OcResp error
	_, _ = conn.Write([]byte("error: " + errMsg + "\n"))
}
