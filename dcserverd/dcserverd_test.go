package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ortutay/decloud/msg"
)

func TestMakeConf(t *testing.T) {
	minFeeFlag := "calc.calc=.01BTC"
	minCoinsFlag := ".=.1BTC"
	maxWorkFlag := "calc.calc={\"bytes\": 1000, \"queries\": 100}"
	conf, err := makeConf(minFeeFlag, minCoinsFlag, maxWorkFlag)
	if err != nil {
		t.Fatalf(err.Error())
	}
	// TODO(ortutay): implement real comparison
	expectedStr := `&{[{{calc calc} min-fee [{1000000 BTC}]} {{ } min-coins [{10000000 BTC}]} {{calc calc} max-work [{"queries":100,"bytes":1000}]}] }`
	if expectedStr != fmt.Sprintf("%v", conf) {
		t.Fatalf("unexpected conf: %v != %v", expectedStr, conf)
	}
}

func TestGetSelector(t *testing.T) {
	psel, err := getSelector("calc.calc")
	if err != nil {
		t.Fatalf(err.Error())
	}
	if psel.Service != "calc" || psel.Method != "calc" {
		t.FailNow()
	}
}

func TestGetSelectorWildcardMethod(t *testing.T) {
	psel, err := getSelector("calc.")
	if err != nil {
		t.Fatalf(err.Error())
	}
	if psel.Service != "calc" || psel.Method != "" {
		t.FailNow()
	}
}

func TestGetSelectorUnsupportedService(t *testing.T) {
	_, err := getSelector("unsupported.unsupportedmethod")
	if err == nil {
		t.FailNow()
	}
}

func TestGetPaymentValue(t *testing.T) {
	pvI, err := getPaymentValue("", ".1BTC")
	if err != nil {
		t.Fatalf(err.Error())
	}
	pv := pvI.(*msg.PaymentValue)
	if msg.BTC != pv.Currency {
		t.Fatalf("expected %v, got %v", msg.BTC, pv.Currency)
	}
	if 1e7 != pv.Amount {
		t.Fatalf("expected %v, got %v", 1e7, pv.Amount)
	}
}

func TestGetPaymentValueAlternateFormat(t *testing.T) {
	pvI, err := getPaymentValue("", "2.1 BTC")
	if err != nil {
		t.Fatalf(err.Error())
	}
	pv := pvI.(*msg.PaymentValue)
	if msg.BTC != pv.Currency {
		t.Fatalf("expected %v, got %v", msg.BTC, pv.Currency)
	}
	if 2.1e8 != pv.Amount {
		t.Fatalf("expected %v, got %v", 2.1e8, pv.Amount)
	}
}

func TestGetPaymentValueOverMaxPrecision(t *testing.T) {
	_, err := getPaymentValue("", ".123456789BTC")
	if err == nil {
		t.FailNow()
	}
	if !strings.HasPrefix(err.Error(), "max precision is 8") {
		t.FailNow()
	}
}
