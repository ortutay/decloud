package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"io/ioutil"

	"github.com/droundy/goopt"
	"github.com/ortutay/decloud/cred"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/node"
	"github.com/ortutay/decloud/rep"
	"github.com/ortutay/decloud/services/calc"
	"github.com/ortutay/decloud/services/payment"
	"github.com/ortutay/decloud/util"

	"github.com/andrew-d/go-termutil"
)

// General flags
var fAddr = goopt.String([]string{"-a", "--addr"}, "", "Remote host address")
var fAppDir = goopt.String([]string{"--app-dir"}, "~/.decloud", "")
var fCoinsLower = goopt.String([]string{"--coins-lower"}, "0btc", "")
var fCoinsUpper = goopt.String([]string{"--coins-upper"}, "10btc", "")
var fVerbosity = goopt.Int([]string{"-v", "--verbosity"}, 0, "")

// var fTestNet = goopt.Flag([]string{"-t", "--test-net"}, []string{"--main-net"}, "Use testnet", "Use mainnet")

// Cross-service flags
var fDefer = goopt.String([]string{"--defer"}, "", "Promise deferred payment")

// Store service flags
var fStoreFile = goopt.String([]string{"--store.file"}, "", "File to store")
var fStoreFor = goopt.String([]string{"--store.for"}, "1h", "How long to store")
var fStoreGbPricePerMo = goopt.String([]string{"--store.gb-price-per-mo"}, ".001BTC", "")

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

	pvLower, err := msg.NewPaymentValueParseString(*fCoinsLower)
	if err != nil {
		log.Fatal(err)
	}
	pvUpper, err := msg.NewPaymentValueParseString(*fCoinsUpper)
	if err != nil {
		log.Fatal(err)
	}
	coins, err := cred.GetBtcCredInRange(pvLower.Amount, pvUpper.Amount, bConf)
	if err != nil {
		log.Fatal(err.Error())
	}

	c := node.Client{
		BtcConf: bConf,
		Cred: cred.Cred{
			OcCred:  *ocCred,
			BtcConf: bConf,
			Coins:   *coins,
		},
	}

	var body []byte
	if !termutil.Isatty(os.Stdin.Fd()) {
		var err error
		body, err = ioutil.ReadAll(os.Stdin)
		util.Ferr(err)
	}

	cmdArgs := make([]string, 0)
	for _, arg := range os.Args[1:] {
		if arg[0] != '-' {
			cmdArgs = append(cmdArgs, arg)
		}
	}
	if len(cmdArgs) == 0 {
		// TODO(ortutay): print usage info
		return
	}

	var req *msg.OcReq
	switch cmdArgs[0] {
	case "quote":
		qReq, err := makeQuoteReq(cmdArgs[1:])
		if err != nil {
			log.Fatal(err.Error())
		}
		resp := sendRequest(&c, qReq)
		if resp.Status == msg.OK {
			var pv msg.PaymentValue
			err := json.Unmarshal(resp.Body, &pv)
			if err != nil {
				panic(err)
			}
			fmt.Printf("\n%v%v\n", util.S2B(pv.Amount), pv.Currency)
		}
	case "call":
		req, err = makeReq(cmdArgs[1:], body)
		if err != nil {
			log.Fatal(err.Error())
		}
		resp := sendRequest(&c, req)
		switch fmt.Sprintf("%v.%v", req.Service, req.Method) {
		case "payment.balance": {
			var br payment.BalanceResponse
			err := json.Unmarshal(resp.Body, &br)
			if err != nil {
				log.Fatalf("malformed response")
			}
			fmt.Printf("\nServer reports balance of %v%v (max allowed is %v%v)\n",
				util.S2B(br.Balance.Amount), br.Balance.Currency,
				util.S2B(br.MaxBalance.Amount), br.MaxBalance.Currency)
		}
		}
	case "pay":
		payBtc(&c, cmdArgs)
	case "listrep":
		sel := rep.Record{}
		if len(cmdArgs) > 1 {
			selJson := cmdArgs[1]
			err := json.Unmarshal([]byte(selJson), &sel)
			if err != nil {
				log.Fatal(err.Error())
			}
			fmt.Printf("sel json: %v %v\n", selJson, sel)
		}
		err := rep.PrettyPrint(&sel)
		if err != nil {
			log.Fatal(err.Error())
		}
	default:
		fmt.Printf("unrecognized command: %v", cmdArgs)
	}
}

func sendRequest(c *node.Client, req *msg.OcReq) *msg.OcResp {
	// Parse/attach payments
	if *fDefer != "" {
		pv, err := msg.NewPaymentValueParseString(*fDefer)
		if err != nil {
			log.Fatal(err.Error())
		}
		req.AttachDeferredPayment(pv)
	}

	// TODO(ortutay): handle non-deferred payments
	err := c.SignRequest(req)
	if err != nil {
		log.Fatal(err.Error())
	}
	if *fVerbosity > 0 {
		fmt.Printf("sending request to %v\n%v\n\n", *fAddr, req.String())
	}
	resp, err := c.SendRequest(*fAddr, req)
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Printf("%v\n", resp.String())
	if resp.Status == msg.PLEASE_PAY {
		var pr msg.PaymentRequest
		err := json.Unmarshal(resp.Body, &pr)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Server is requesting payment: %v%v to %v\n",
			util.S2B(pr.Amount), pr.Currency, pr.Addr)
	}
	return resp
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
	req, err := makeReq(args, nil)
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

func makeReq(args []string, body []byte) (*msg.OcReq, error) {
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
	}
	if body != nil {
		req.SetBody(body)
	}
	return &req, nil
}
