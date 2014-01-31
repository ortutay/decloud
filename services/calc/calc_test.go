package calc

import (
	"fmt"
	"log"
	"oc/buyer/client"
	"oc/seller/calc"
	"testing"
)

var _ = fmt.Printf

func TestHandleCalculate_Simple(t *testing.T) {
	req := client.NewCalcReq([]string{"1 2 +"})
	resp, err := calc.HandleCalculate(&req)
	if err != nil {
		log.Fatal(err)
	}
	body := string(resp.Body)
	if body != "3" {
		t.Errorf("got %v, expected %v\n", body, 3)
	}
}

func TestHandleCalculate_Complex(t *testing.T) {
	args := []string{
		"1 2 +",
		"1 2 /",
		"3 4.5 + 7 - 8.123 *",
		"5 1 2 + 4 * + 3 -"}
	req := client.NewCalcReq(args)
	resp, err := calc.HandleCalculate(&req)
	if err != nil {
		log.Fatal(err)
	}
	body := string(resp.Body)
	exp := "3 0.5 4.0615 14"
	if body != exp {
		t.Errorf("got %v, expected %v\n", body, exp)
	}
}

func TestHandleCalculate_InvalidExpr(t *testing.T) {
	req := client.NewCalcReq([]string{"1 2 + 3"})
	_, err := calc.HandleCalculate(&req)
	if err == nil || err.Error() != "invalid expression" {
		log.Fatal(err)
	}
}
