package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/droundy/goopt"
	"github.com/ortutay/decloud/cred"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/node"
	"github.com/ortutay/decloud/services/calc"
	"github.com/ortutay/decloud/util"
)

var fAddr = goopt.String([]string{"-a", "--addr"}, "", "Remote host address")
var fAppDir = goopt.String([]string{"--app-dir"}, "~/.decloud", "")
var fDefer = goopt.String([]string{"--defer"}, "", "Included deferred payment")

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
		BtcConf: bConf,
		Cred: cred.Cred{
			OcCred: *ocCred,
			Coins:  []cred.BtcCred{},
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
		if err != nil {
			log.Fatal(err.Error())
		}
		sendRequest(&c, qReq)
	case "call":
		req, err = makeReq(cmdArgs[1:])
		if err != nil {
			log.Fatal(err.Error())
		}
		sendRequest(&c, req)
	case "pay":
		payBtc(&c, cmdArgs)
	default:
		fmt.Printf("unrecognized command: %v", cmdArgs)
	}
}

func sendRequest(c *node.Client, req *msg.OcReq) {
	// Parse/attach payments
	if *fDefer != "" {
		pv, err := msg.NewPaymentValueParseString(*fDefer)
		if err != nil {
			log.Fatal(err.Error())
		}
		req.AttachDeferredPayment(pv)
	}

	// TODO(ortutay): handle non-deferred payments

	fmt.Printf("Sending request:\n%v\n\n", req.String())
	fmt.Printf("addr %v\n", *fAddr)
	resp, err := c.SendRequest(*fAddr, req)
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Printf("Got response:\n%v\n", resp.String())
}

func payBtc(c *node.Client, cmdArgs []string) {
	amt := cmdArgs[1]
	addr := cmdArgs[2]
	pv, err := msg.NewPaymentValueParseString(amt)
	if err != nil {
		log.Fatal(err.Error())
	}
	// TODO(ortutay): validate bitcoin address
	pa := msg.PaymentAddr{
		Currency: msg.BTC,
		Addr:     addr,
	}
	txid, err := c.SendBtcPayment(pv, &pa)
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Printf("sent payment, txid: %v\n", txid)
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
	reqArgs := args[1:]
	req := msg.OcReq{
		ID:            "",
		Sig:           "",
		Nonce:         "",
		Service:       s[0],
		Method:        s[1],
		Args:          reqArgs,
		PaymentType:   "",
		PaymentTxn:    "",
		ContentLength: 0,
		Body:          []byte(""),
	}
	return &req, nil
}
