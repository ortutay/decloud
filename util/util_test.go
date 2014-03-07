package util

import (
	"fmt"
	"testing"
)

func TestParseByteSize(t *testing.T) {
	fmt.Printf("x\n")
	s, err := ByteSizeParseString("1MB")
	if err != nil {
		t.Fatal(err)
	}
	if s.Int() != int(1e6) {
		t.Fatalf("%v", s.Int(), int(1e6))
	}

	s, err = ByteSizeParseString(".5GB")
	if err != nil {
		t.Fatal(err)
	}
	if s.Int() != int(500e6) {
		t.Fatalf("%v", s.Int())
	}
}
