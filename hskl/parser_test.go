package hskl

import (
	"fmt"
	"io/ioutil"
	"testing"
)

func TestParser(t *testing.T) {
	body, _ := ioutil.ReadFile("../data/hskl/test.hskl")
	program := string(body)

	p := NewParser(program)

	fmt.Printf("parse result: %T, lastErr: %s\n", p.declarations(), p.lastError)
}
