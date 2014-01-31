package calc

import (
	"strconv"
	"strings"
	"oc/msg"
	"fmt"
	"errors"
)
var _ = fmt.Printf

func NewCalcReq(queries []string) *msg.OcReq {
	msg := msg.OcReq{
		NodeId: []string{},
		Sig: []string{},
		Nonce: "",
		Service: "calc",
		Method: "calculate",
		Args: queries,
		PaymentType: "",
		PaymentTxn: "",
		Body: []byte(""),
	}
	return &msg
}

type CalcService struct {
}

func (cs CalcService) Handle(req *msg.OcReq) (*msg.OcResp, error) {
	println(fmt.Sprintf("calc got request: %v", req))
	switch req.Method {
	case "calculate":
		return cs.HandleCalculate(req)
	default:
		return nil, errors.New("unhandled method")
	}
	return nil, nil
}

func (cs CalcService) HandleCalculate(req *msg.OcReq) (*msg.OcResp, error) {
	var results []string
	for _, q := range req.Args {
		tokens := strings.Split(q, " ")
		var stack []float64
		for _, token := range tokens {
			switch token {
			case "+":
				r := stack[len(stack)-1] + stack[len(stack)-2]
				stack = stack[0:len(stack)-2]
				stack = append(stack, r)
			case "-":
				r := stack[len(stack)-2] - stack[len(stack)-1]
				stack = stack[0:len(stack)-2]
				stack = append(stack, r)
			case "/":
				r := stack[len(stack)-2] / stack[len(stack)-1]
				stack = stack[0:len(stack)-2]
				stack = append(stack, r)
			case "*":
				r := stack[len(stack)-1] * stack[len(stack)-2]
				stack = stack[0:len(stack)-2]
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
