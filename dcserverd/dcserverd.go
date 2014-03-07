package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/droundy/goopt"
	"github.com/ortutay/decloud/conf"
	"github.com/ortutay/decloud/cred"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/node"
	"github.com/ortutay/decloud/services/calc"
	"github.com/ortutay/decloud/services/payment"
	"github.com/ortutay/decloud/util"
)

// General flags
var fPort = goopt.Int([]string{"-p", "--port"}, 9443, "")
var fAppDir = goopt.String([]string{"--app-dir"}, "~/.decloud", "")
// var fTestNet = goopt.Flag([]string{"-t", "--test-net"}, []string{"--main-net"}, "Use testnet", "Use mainnet")

// Cross-service flags
var fMinFee = goopt.String([]string{"--min-fee"}, "calc.calc=.01BTC", "")
var fMinCoins = goopt.String([]string{"--min-coins"}, "calc.calc=.1BTC", "")
var fMaxWork = goopt.String([]string{"--max-work"}, "calc.calc={\"bytes\": 1000, \"queries\": 100}", "")

// Store service flags
var fStoreDir = goopt.String([]string{"--store.dir"}, "~/.decloud-store", "")
var fStoreMaxSpace = goopt.String([]string{"--store.max-space"}, "1GB", "")
var fStoreGbPricePerMo = goopt.String([]string{"--store.gb-price-per-mo"}, ".001BTC", "")

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
	conf.Setting("store.dir", *fStoreDir)
	conf.Setting("store.max-space", getSpace("", *fStoreMaxSpace))
	conf.Setting("store.gb-price-per-mo",
		getPaymentValue("", *fStoreGbPricePerMo))
	fmt.Printf("running with conf: %v\n", conf)

	util.SetAppDir(*fAppDir)
	ocCred, err := cred.NewOcCredLoadOrCreate("")
	if err != nil {
		log.Fatal(err.Error())
	}

	// TODO(ortutay): really, this should be flags to the binary so that we don't
	// spend people's coins without explicit intent
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
		Cred:    &cred.Cred{OcCred: *ocCred, Coins: []cred.BtcCred{}},
		Conf:    conf,
		Addr:    addr,
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
		policy, err := getPolicy(minFeeArg, conf.MIN_FEE, getPaymentValue)
		if err != nil {
			return nil, err
		}
		// TODO(ortutay): is there a better way to do this type conversion?
		pvp := policy.Args[0].(*msg.PaymentValue)
		policy.Args[0] = *pvp
		policies = append(policies, *policy)
	}

	// Parse min coins
	for _, minCoinsArg := range minCoinsArgs {
		policy, err := getPolicy(minCoinsArg, conf.MIN_COINS, getPaymentValue)
		if err != nil {
			return nil, err
		}
		// TODO(ortutay): is there a better way to do this type conversion?
		pvp := policy.Args[0].(*msg.PaymentValue)
		policy.Args[0] = *pvp
		policies = append(policies, *policy)
	}

	// Parse max work
	for _, maxWorkArg := range maxWorkArgs {
		policy, err := getPolicy(maxWorkArg, conf.MAX_WORK, getWork)
		if err != nil {
			return nil, err
		}
		// TODO(ortutay): I guess we have to do type conversion here also...
		// but can't easily...
		policies = append(policies, *policy)
	}

	conf := conf.Conf{
		Policies: policies,
		BtcAddr:  "",
	}
	return &conf, nil
}

func getPolicy(arg string, cmd conf.PolicyCmd, parse func(string, string) (interface{})) (*conf.Policy, error) {
	s := strings.Split(arg, "=")
	if len(s) != 2 {
		return nil, fmt.Errorf("could not parse: %v", arg)
	}
	psel := getSelector(s[0])
	pArg := parse(s[0], s[1])
	policy := conf.Policy{
		Selector: *psel,
		Cmd:      cmd,
		Args:     []interface{}{pArg},
	}
	return &policy, nil
}

func getSelector(sel string) *conf.PolicySelector {
	s := strings.Split(sel, ".")
	if len(s) != 2 {
		log.Fatalf("could not parse: %v", sel)
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
		log.Fatalf("unsupported service: '%v'", service)
	}
	_, methOk := srvSupported[method]
	if !methOk {
		log.Fatalf("unsupported method: '%v'", method)
	}
	return &conf.PolicySelector{Service: service, Method: method}
}

func getPaymentValue(srvMeth, pvStr string) interface{} {
	pv, err := msg.NewPaymentValueParseString(pvStr)
	if err != nil {
		log.Fatal(err)
	}
	return pv
}

func getWork(srvMeth, w string) interface{} {
	switch srvMeth {
	case "calc.calc":
		work, err := calc.NewWork(w)
		if err != nil {
			log.Fatal(err)
		}
		return work
	default:
		log.Fatal(fmt.Errorf("no support for work flag for %v", srvMeth))
	}
	return nil // unreachable
}

func getSpace(srvMeth, str string) interface{} {
	size, err := util.ByteSizeParseString(str)
	if err != nil {
		log.Fatal(err)
	}
	return size
}
