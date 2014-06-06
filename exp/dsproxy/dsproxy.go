package main

import (
	"fmt"
	"strings"
	"log"

	"github.com/droundy/goopt"
	"github.com/ortutay/decloud/util"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/node"
	"github.com/ortutay/decloud/conf"
	"github.com/ortutay/decloud/cred"
)

var fPort = goopt.Int([]string{"-p", "--port"}, 9443, "")
var fAppDir = goopt.String([]string{"--app-dir"}, "~/.decloud", "")
var fMaxBalance = goopt.String([]string{"--max-balance"}, ".1BTC", "")

var fMinCoins = goopt.String([]string{"--min-coins"}, "calc.calc=.1BTC", "")

func main() {
	goopt.Parse(nil)
	util.SetAppDir(*fAppDir)

	ocCred, err := cred.NewOcCredLoadOrCreate("")
	util.Ferr(err)

	config, err := makeConf(*fMinCoins)
	util.Ferr(err)

	config.AddPolicy(&conf.Policy{
		Selector: conf.PolicySelector{},
		Cmd:      conf.MAX_BALANCE,
		Args:     []interface{}{getPaymentValue("", *fMaxBalance)},
	})

	// TODO(ortutay): This should be flags to the binary so that we don't risk
	// spending people's coins without explicit intent. The server, though,
	// should not be sending coins.
	bConf, err := util.LoadBitcoindConf("")
	util.Ferr(err)

	addr := fmt.Sprintf(":%v", *fPort)

	services := make(map[string]node.Handler)
	mux := node.ServiceMux{
		Services: services,
	}

	wakers := []node.PeriodicWaker{}

	s := node.Server{
		Cred: &cred.Cred{
			OcCred:  *ocCred,
			BtcConf: bConf,
			Coins:   []cred.BtcCred{},
		},
		BtcConf: bConf,
		Conf:    config,
		Addr:    addr,
		Handler: &mux,
		PeriodicWakers: wakers,
	}

	fmt.Printf("server: %v\n", s)
}

func makeConf(minCoinsFlag string) (*conf.Conf, error) {
	minCoinsArgs := strings.Split(minCoinsFlag, ";")
	policies := make([]conf.Policy, 0)

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

	conf := conf.Conf{
		Policies: policies,
		BtcAddr:  "",
	}
	return &conf, nil
}

func getPolicy(arg string, cmd conf.PolicyCmd, parse func(string, string) interface{}) (*conf.Policy, error) {
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
