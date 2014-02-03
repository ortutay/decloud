package calc

import (
	"errors"
	"fmt"
	"oc/msg"
	"strconv"
	"strings"
)

var _ = fmt.Printf

const (
	CALC = "calc"
	CALCULATE = "calculate"
)

func NewQuoteReq(work *Work) *msg.OcReq {
	return nil
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
	NumBytes int
	NumQueries int
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

func (cs *CalcService) Handle(req *msg.OcReq) (*msg.OcResp, error) {
	println(fmt.Sprintf("calc got request: %v", req))

	methods := make(map[string]func(*msg.OcReq) (*msg.OcResp, error))
	methods[CALCULATE] = cs.Calculate

	if method, ok := methods[req.Method]; ok {
		return method(req)
	} else {
		return nil, errors.New("unhandled method")
	}
}

func (cs *CalcService) Info(req *msg.OcReq) (*msg.OcResp, error) {
	return nil, nil
}

func (cs *CalcService) Quote(req *msg.OcReq) (*msg.OcResp, error) {
	return nil, nil
}

func (cs *CalcService) Methods(req *msg.OcReq) (*msg.OcResp, error) {
	return nil, nil
}

func (cs *CalcService) Calculate(req *msg.OcReq) (*msg.OcResp, error) {
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
