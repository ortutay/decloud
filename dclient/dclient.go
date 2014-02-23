package main

import (
	"fmt"
	"github.com/droundy/goopt"
	"github.com/ortutay/decloud/cred"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/node"
	"github.com/ortutay/decloud/services/calc"
	"github.com/ortutay/decloud/util"
	"log"
	"os"
	"strings"
)

var fAddr = goopt.String([]string{"-a", "--addr"}, "", "Remote host address")
var fAppDir = goopt.String([]string{"--app-dir"}, "~/.decloud", "")

func main() {
	goopt.Parse(nil)
	util.SetAppDir(*fAppDir)

	ocCred, err := cred.NewOcCredLoadOrCreate("")
	if err != nil {
		log.Fatal(err.Error())
	}

	bConf, err := util.LoadBitcoindConf("")
	if err != nil {
		log.Fatal(err.Error())
	}

	c := node.Client{
		BitcoindConf: bConf,
		Cred: cred.Cred{
			Signers: []cred.Signer{ocCred},
		},
	}
	fmt.Printf("client %v\n", c)

	cmdArgs := make([]string, 0)
	for _, arg := range os.Args[1:] {
		if arg[0] != '-' {
			cmdArgs = append(cmdArgs, arg)
		}
	}
	fmt.Printf("cmd args: %v\n", cmdArgs)

	var req *msg.OcReq
	switch cmdArgs[0] {
	case "quote":
		qReq, err := makeQuoteReq(cmdArgs[1:])
		req = qReq
		if err != nil {
			log.Fatal(err.Error())
		}
	default:
		fmt.Printf("unrecognized command: %v", cmdArgs)
	}

	fmt.Printf("Sending request:\n%v\n\n", req.String())
	fmt.Printf("addr %v\n", *fAddr)
	resp, err := c.SendRequest(*fAddr, req)
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Printf("Got response:\n%v\n", resp.String())
}

func makeQuoteReq(args []string) (*msg.OcReq, error) {
	req, err := makeReq(args)
	if err != nil {
		return nil, fmt.Errorf("couldn't make request to quote: %v", err.Error())
	}

	switch args[0] {
	case "calc.calc":
		return calc.NewQuoteReqFromReq(req)
	default:
		return nil, fmt.Errorf("cannot make quote request for: %v", args[0])
	}
}

func makeReq(args []string) (*msg.OcReq, error) {
	s := strings.Split(args[0], ".")
	if len(s) != 2 {
		return nil, fmt.Errorf("expected server.method, but got: %v", args[0])
	}
	req := msg.OcReq{
		Id:            []string{},
		Sig:           []string{},
		Nonce:         "",
		Service:       s[0],
		Method:        s[1],
		Args:          []byte("[\"" + strings.Join(args[1:], "\", \"") + "\"]"),
		PaymentType:   "",
		PaymentTxn:    "",
		ContentLength: 0,
		Body:          []byte(""),
	}
	return &req, nil
}
