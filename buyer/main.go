package main

import (
	"net"
	"bufio"
	"fmt"
	"oc/buyer/client"
	"oc/msg"
	"log"
)

func main() {
	println("buyer main")
	conn, err := net.Dial("tcp", "localhost:9443")
	if err != nil {
		log.Fatal("error while dialing: ", err)
	}

	args := []string{
		"1 2 +",
		"1 2 /",
		"3 4.5 + 7 - 8.123 *",
		"5 1 2 + 4 * + 3 -"}
	req := client.NewCalcReq(args)
	b64 := string(msg.EncodeReq(&req))
	fmt.Fprintf(conn, b64 + "\n")
	b64resp, err := bufio.NewReader(conn).ReadString('\n')
	resp, err := msg.DecodeResp(b64resp)
	println(fmt.Sprintf("got response: %v", string(resp.Body)))
}
