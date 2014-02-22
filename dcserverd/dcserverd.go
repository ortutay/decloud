package main

import (
	"strconv"
	"math/big"
	"regexp"
	"strings"
	"fmt"
	"os"
	"log"
	"github.com/droundy/goopt"
	"github.com/ortutay/decloud/conf"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/util"
	"github.com/ortutay/decloud/cred"
	"github.com/ortutay/decloud/node"
	"github.com/ortutay/decloud/services/calc"
	"github.com/ortutay/decloud/services/payment"
)

// var fTestNet = goopt.Flag([]string{"-t", "--test-net"}, []string{"--main-net"}, "Use testnet", "Use mainnet")
var fMinFee = goopt.String([]string{"--min-fee"}, "calc.calc=.01BTC", "")
var fMinCoins = goopt.String([]string{"--min-coins"}, "calc.calc=.1BTC", "")
var fMaxWork = goopt.String([]string{"--max-work"}, "calc.calc={\"bytes\": 1000, \"queries\": 100}", "")
var fPort = goopt.Int([]string{"-p", "--port"}, 9443, "")

func main() {
	goopt.Parse(nil)
	cmdArgs := make([]string, 0)
	for _, arg := range os.Args[1:] {
		if arg[0] != '-' {
			cmdArgs = append(cmdArgs, arg)
		}
	}
	fmt.Printf("cmd args: %v\n", cmdArgs)
	conf, err := makeConf(*fMinFee, *fMinCoins, *fMaxWork)
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Printf("running with conf: %v\n", conf)

	// TODO(ortutay): really, this should be flags to the binary so that we don't
	// spend people's coins without explicit permission
	bConf, err := util.LoadBitcoindConf("")
	if err != nil {
		log.Fatal(err.Error())
	}

	addr := fmt.Sprintf(":%v", *fPort)

	// TODO(ortutay): configure which services to run from command line args
	services := make(map[string]node.Handler)
	services[calc.SERVICE_NAME] = calc.CalcService{Conf: conf}
	services[payment.SERVICE_NAME] = &payment.PaymentService{BitcoindConf: bConf}
	mux := node.ServiceMux{
		Services: services,
	}
	s := node.Server{
		Cred: &cred.Cred{}, // TODO(ortutay): generate/fill in cred
		Conf: conf,
		Addr: addr,
		Handler: &mux,
	}
	err = s.ListenAndServe()
	if err != nil {
		log.Fatal(err.Error())
	}
}

func makeConf(minFeeFlag string, minCoinsFlag string, maxWorkFlag string) (*conf.Conf, error) {
	minFeeArgs := strings.Split(minFeeFlag, ";")
	minCoinsArgs := strings.Split(minCoinsFlag, ";")
	maxWorkArgs := strings.Split(maxWorkFlag, ";")
	policies := make([]conf.Policy, 0)

	// Parse min fees
 	for _, minFeeArg := range minFeeArgs {
		s := strings.Split(minFeeArg, "=")
		if len(s) != 2 {
			return nil, fmt.Errorf("could not parse: %v", minFeeArg)
		}
		psel, err := getSelector(s[0])
		if err != nil {
			return nil, err
		}
		pv, err := getPaymentValue(s[1])
		if err != nil {
			return nil, err
		}
		policy := conf.Policy{
			Selector: *psel,
			Cmd: conf.MIN_FEE,
			Args: []interface{}{pv},
		}
		policies = append(policies, policy)
	}

	// Parse min coins
 	for _, minCoinsArg := range minCoinsArgs {
		s := strings.Split(minCoinsArg, "=")
		if len(s) != 2 {
			return nil, fmt.Errorf("could not parse: %v", minCoinsArg)
		}
		psel, err := getSelector(s[0])
		if err != nil {
			return nil, err
		}
		pv, err := getPaymentValue(s[1])
		if err != nil {
			return nil, err
		}
		policy := conf.Policy{
			Selector: *psel,
			Cmd: conf.MIN_COINS,
			Args: []interface{}{pv},
		}
		policies = append(policies, policy)
	}

	// Parse max work
	for _, maxWorkArg := range maxWorkArgs {
		s := strings.SplitN(maxWorkArg, "=", 2)
		if len(s) != 2 {
			return nil, fmt.Errorf("could not parse: %v", maxWorkArg)
		}
		psel, err := getSelector(s[0])
		if err != nil {
			return nil, err
		}
		work, err := getWork(psel.Service, psel.Method, s[1])
		if err != nil {
			return nil, err
		}
		policy := conf.Policy{
			Selector: *psel,
			Cmd: conf.MAX_WORK,
			Args: []interface{}{work},
		}
		policies = append(policies, policy)
	}

	conf := conf.Conf{
		Policies: policies,
		BtcAddr: "",
	}
	return &conf, nil
}

func getSelector(sel string) (*conf.PolicySelector, error) {
	s := strings.Split(sel, ".")
	if len(s) != 2 {
		return nil, fmt.Errorf("could not parse: %v", sel)
	}
	service := s[0]
	method := s[1]

	supported := make(map[string]map[string]bool)

	supported[""] = make(map[string]bool)
	supported["calc"] = make(map[string]bool)

	supported[""][""] = true
	supported["calc"][""] = true
	supported["calc"]["calc"] = true
	supported["calc"]["quote"] = true

	srvSupported, srvOk := supported[service]
	if !srvOk {
		return nil, fmt.Errorf("unsupported service: '%v'", service)
	}
	_, methOk := srvSupported[method]
	if !methOk {
		return nil, fmt.Errorf("unsupported method: '%v'", method)
	}
	return &conf.PolicySelector{Service: service, Method: method}, nil
}

func getPaymentValue(pv string) (*msg.PaymentValue, error) {
	re := regexp.MustCompile("(?i)([0-9.]+) *(btc)")
	m := re.FindStringSubmatch(pv)
	if len(m) != 3 {
		return nil, fmt.Errorf("could not parse: %v", pv)
	}
	r := new(big.Rat)
	_, err := fmt.Sscan(m[1], r)
	if err != nil {
		return nil, fmt.Errorf("could not parse: %v", m[1])
	}
	r.Mul(r, big.NewRat(1e8, 1))
	if !r.IsInt() {
		return nil, fmt.Errorf("max precision is 8 decimal places (%v)", m[1])
	}
	intStr := r.RatString()
	satoshis, err := strconv.ParseInt(intStr, 10, 64)
	if err != nil {
		// unexpected, r.RatString() should always return valid integer string
		panic(err)
	}
	return &msg.PaymentValue{Amount: satoshis, Currency: msg.BTC}, nil
}

func getWork(service, method, w string) (interface{}, error) {
	switch service + "." + method {
	case "calc.calc":
		return calc.NewWork(w)
	default:
		return nil, fmt.Errorf("no support for work flag for %v.%v",
			service, method)
	}
}
