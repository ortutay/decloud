package calc

import (
	"fmt"
	"log"
	"testing"
)

var _ = fmt.Printf

func TestCalculate_Simple(t *testing.T) {
	cs := CalcService{}
	req := NewCalcReq([]string{"1 2 +"})
	resp, err := cs.Calculate(req)
	if err != nil {
		log.Fatal(err)
	}
	body := string(resp.Body)
	if body != "3" {
		t.Errorf("got %v, expected %v\n", body, 3)
	}
}

func TestCalculate_Complex(t *testing.T) {
	cs := CalcService{}
	args := []string{
		"1 2 +",
		"1 2 /",
		"3 4.5 + 7 - 8.123 *",
		"5 1 2 + 4 * + 3 -"}
	req := NewCalcReq(args)
	resp, err := cs.Calculate(req)
	if err != nil {
		log.Fatal(err)
	}
	body := string(resp.Body)
	exp := "3 0.5 4.0615 14"
	if body != exp {
		t.Errorf("got %v, expected %v\n", body, exp)
	}
}

func TestCalculate_InvalidExpr(t *testing.T) {
	cs := CalcService{}
	req := NewCalcReq([]string{"1 2 + 3"})
	_, err := cs.Calculate(req)
	if err == nil || err.Error() != "invalid expression" {
		log.Fatal(err)
	}
}
