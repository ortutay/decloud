package calc

import (
	"errors"
	"fmt"
	"oc/msg"
	"strconv"
	"strings"
	"encoding/json"
)

var _ = fmt.Printf

const (
	CALC = "calc"
	CALCULATE = "calculate"
	QUOTE = "quote"
)

func NewQuoteReq(work *Work) *msg.OcReq {
	workStr := work.ToString()
	msg := msg.OcReq{
		NodeId:   []string{},
		Sig:     []string{},
		Nonce:    "",
		Service:   CALC,
		Method:   QUOTE,
		Args:    []string{workStr},
		PaymentType: "",
		PaymentTxn: "",
		Body:    []byte(""),
	}
	return &msg
}

func NewCalcReq(queries []string) *msg.OcReq {
	msg := msg.OcReq{
		NodeId:   []string{},
		Sig:     []string{},
		Nonce:    "",
		Service:   CALC,
		Method:   CALCULATE,
		Args:    queries,
		PaymentType: "",
		PaymentTxn: "",
		Body:    []byte(""),
	}
	return &msg
}

type Work struct {
	NumQueries int
	NumBytes int
}
func (w *Work) ToString() string {
	// TODO(ortutay): figure out real wire format
	b, err := json.Marshal(w)
	if err != nil {
		panic(err)
	}
	return string(b)
}
func FromString(str string) (*Work, error) {
	// TODO(ortutay): figure out real wire format
	var work Work
	err := json.Unmarshal([]byte(str), &work)
	if err != nil {
	 	return nil, fmt.Errorf("couldn't create calc.Work from %v", str)
	} else {
		return &work, nil
	}
}

// TODO(ortutay): standard quotable units

func Measure(req *msg.OcReq) (*Work, error) {
	if req.Service != CALC {
		return nil, fmt.Errorf("expected %s service, got %s", CALC, req.Service)
	}
	if req.Method != CALCULATE {
		return nil, fmt.Errorf("can only measure work for %s method, got %s",
			CALCULATE, req.Method)
	}
	var work Work
	for _, q := range req.Args {
		work.NumBytes += len(q)
		work.NumQueries++
	}
	return &work, nil
}

type CalcService struct {
}

func (cs CalcService) Handle(req *msg.OcReq) (*msg.OcResp, error) {
	println(fmt.Sprintf("calc got request: %v", req))

	methods := make(map[string]func(*msg.OcReq) (*msg.OcResp, error))
	methods[CALCULATE] = cs.Calculate
	methods[QUOTE] = cs.Quote

	if method, ok := methods[req.Method]; ok {
		return method(req)
	} else {
		return msg.NewRespError(msg.METHOD_UNSUPPORTED), nil
	}
}

func (cs CalcService) Info(req *msg.OcReq) (*msg.OcResp, error) {
	return nil, nil
}

func (cs CalcService) Quote(req *msg.OcReq) (*msg.OcResp, error) {
	pv := msg.PaymentValue{Amount: .01, Currency: "BTC"}
	resp := msg.NewRespOk([]byte(pv.ToString()))
	return resp, nil
}

func (cs CalcService) Methods(req *msg.OcReq) (*msg.OcResp, error) {
	return nil, nil
}

func (cs CalcService) Calculate(req *msg.OcReq) (*msg.OcResp, error) {
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
	return resp, nil
}
