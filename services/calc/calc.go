package calc

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/ortutay/decloud/btc"
	"github.com/ortutay/decloud/conf"
	"github.com/ortutay/decloud/msg"
)

var _ = fmt.Printf

const (
	SERVICE_NAME     = "calc"
	CALCULATE_METHOD = "calc"
	QUOTE_METHOD     = "quote"
)

type Work struct {
	Queries int `json:"queries"`
	Bytes   int `json:"bytes"`
}

func NewQuoteReqFromReq(orig *msg.OcReq) (*msg.OcReq, error) {
	work, err := Measure(orig)
	if err != nil {
		return nil, fmt.Errorf("couldn't measure work: %v", err.Error())
	}
	req := NewQuoteReq(work)
	return req, nil
}

func NewQuoteReq(work *Work) *msg.OcReq {
	workJson, err := json.Marshal(work)
	if err != nil {
		panic(err)
	}
	msg := msg.OcReq{
		ID:            "",
		Sig:           "",
		Coins:         []string{},
		CoinSigs:      []string{},
		Nonce:         "",
		Service:       SERVICE_NAME,
		Method:        QUOTE_METHOD,
		Args:          []string{CALCULATE_METHOD, string(workJson)},
		PaymentType:   "",
		PaymentTxn:    "",
		ContentLength: 0,
		Body:          []byte(""),
	}
	return &msg
}

func NewCalcReq(queries []string) *msg.OcReq {
	msg := msg.OcReq{
		ID:            "",
		Sig:           "",
		Coins:         []string{},
		CoinSigs:      []string{},
		Nonce:         "",
		Service:       SERVICE_NAME,
		Method:        CALCULATE_METHOD,
		Args:          queries,
		PaymentType:   "",
		PaymentTxn:    "",
		ContentLength: 0,
		Body:          []byte(""),
	}
	return &msg
}

func (w *Work) String() string {
	b, err := json.Marshal(w)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func NewWork(str string) (*Work, error) {
	var work Work
	err := json.Unmarshal([]byte(str), &work)
	if err != nil {
		return nil, fmt.Errorf("couldn't create calc.Work from %v (%v)",
			str, err.Error())
	} else {
		return &work, nil
	}
}

// TODO(ortutay): standard quotable units

func Measure(req *msg.OcReq) (*Work, error) {
	if req.Service != SERVICE_NAME {
		panic(fmt.Sprintf("unexpected service %s", req.Service))
	}
	if req.Method != CALCULATE_METHOD {
		return nil, fmt.Errorf("can only measure work for %s method, got %s",
			CALCULATE_METHOD, req.Method)
	}
	var work Work
	for _, q := range req.Args {
		work.Bytes += len(q)
		work.Queries++
	}
	return &work, nil
}

type CalcService struct {
	Conf *conf.Conf
}

func (cs CalcService) paymentForWork(work *Work, method string) (*msg.PaymentValue, error) {
	if cs.Conf == nil {
		return &msg.PaymentValue{Amount: 0, Currency: msg.BTC}, nil
	}
	matching := cs.Conf.MatchingPolicies(SERVICE_NAME, method)
	minFees := make([]*conf.Policy, 0)
	for _, policy := range matching {
		if policy.Cmd == conf.MIN_FEE {
			minFees = append(minFees, policy)
		}
	}
	if len(minFees) > 1 {
		log.Printf("more than 1 min fee for calc.%v (got %v)", method, minFees)
		return nil, errors.New("more than 1 min fee")
	}
	var pv msg.PaymentValue
	if len(minFees) == 0 {
		pv = msg.PaymentValue{Amount: 0, Currency: "BTC"}
	} else {
		pv = minFees[0].Args[0].(msg.PaymentValue)
	}
	return &pv, nil
}

func (cs CalcService) Handle(req *msg.OcReq) (*msg.OcResp, error) {
	println(fmt.Sprintf("calc got request: %v", req))
	if req.Service != SERVICE_NAME {
		panic(fmt.Sprintf("unexpected service %s", req.Service))
	}

	methods := make(map[string]func(*msg.OcReq) (*msg.OcResp, error))
	methods[CALCULATE_METHOD] = cs.calculate
	methods[QUOTE_METHOD] = cs.quote

	if method, ok := methods[req.Method]; ok {
		return method(req)
	} else {
		return msg.NewRespError(msg.METHOD_UNSUPPORTED), nil
	}
}

func (cs CalcService) info(req *msg.OcReq) (*msg.OcResp, error) {
	return nil, nil
}

func (cs CalcService) quote(req *msg.OcReq) (*msg.OcResp, error) {
	reqMethod := req.Args[0]
	var reqWork Work
	err := json.Unmarshal([]byte(req.Args[1]), &reqWork)
	if err != nil {
		return msg.NewRespError(msg.INVALID_ARGUMENTS), nil
	}
	if reqMethod != CALCULATE_METHOD {
		return msg.NewRespError(msg.INVALID_ARGUMENTS), nil
	}

	pv, err := cs.paymentForWork(&reqWork, reqMethod)
	if err != nil {
		log.Printf("server error: %v", err.Error())
		return msg.NewRespError(msg.SERVER_ERROR), nil
	}
	resp := msg.NewRespOk([]byte(pv.String()))
	return resp, nil
}

func (cs CalcService) methods(req *msg.OcReq) (*msg.OcResp, error) {
	return nil, nil
}

func (cs CalcService) calculate(req *msg.OcReq) (*msg.OcResp, error) {
	// TODO(ortutay): pull out payment verifcation logic
	work, err := Measure(req)
	if err != nil {
		log.Printf("server error: %v", err.Error())
		return msg.NewRespError(msg.SERVER_ERROR), nil
	}
	pv, err := cs.paymentForWork(work, CALCULATE_METHOD)
	if err != nil {
		log.Printf("server error: %v", err.Error())
		return msg.NewRespError(msg.SERVER_ERROR), nil
	}
	var submitTxn string
	if pv.Amount != 0 {
		fmt.Printf("want payment: %v, got payment: %v\n", pv, req.PaymentValue)
		if req.PaymentType == msg.NONE || req.PaymentValue == nil {
			return msg.NewRespError(msg.PAYMENT_REQUIRED), nil
		}
		if req.PaymentValue.Currency != pv.Currency {
			return msg.NewRespError(msg.CURRENCY_UNSUPPORTED), nil
		}
		if req.PaymentValue.Amount < pv.Amount {
			return msg.NewRespError(msg.TOO_LOW), nil
		}

		switch req.PaymentType {
		case msg.DEFER:
			// TODO(ortutay): check if we accept deferred payment for the request
			// return msg.NewRespError(msg.NO_DEFER), nil
		case msg.ATTACHED:
			if !btc.TxnIsValid(req.PaymentTxn, req.PaymentValue) {
				return msg.NewRespError(msg.INVALID_TXN), nil
			}
			submitTxn = req.PaymentTxn
		}
	}

	var results []string
	for _, q := range req.Args {
		tokens := strings.Split(q, " ")
		var stack []float64
		for _, token := range tokens {
			switch token {
			case "+":
				r := stack[len(stack)-1] + stack[len(stack)-2]
				stack = stack[0 : len(stack)-2]
				stack = append(stack, r)
			case "-":
				r := stack[len(stack)-2] - stack[len(stack)-1]
				stack = stack[0 : len(stack)-2]
				stack = append(stack, r)
			case "/":
				r := stack[len(stack)-2] / stack[len(stack)-1]
				stack = stack[0 : len(stack)-2]
				stack = append(stack, r)
			case "*":
				r := stack[len(stack)-1] * stack[len(stack)-2]
				stack = stack[0 : len(stack)-2]
				stack = append(stack, r)
			default:
				f, err := strconv.ParseFloat(token, 54)
				stack = append(stack, f)
				if err != nil {
					return nil, errors.New("didn't understand \"" + token + "\"")
				}
			}
		}
		if len(stack) != 1 {
			return nil, errors.New("invalid expression")
		}
		results = append(results, fmt.Sprintf("%v", stack[0]))
	}
	resp := msg.NewRespOk([]byte(strings.Join(results, " ")))
	if submitTxn != "" {
		_ = btc.SubmitTxn(submitTxn)
	}
	return resp, nil
}
