package hskl

import (
	"io/ioutil"
	"testing"
)

func TestDotify(t *testing.T) {
	body, _ := ioutil.ReadFile("../data/hskl/test.hskl")
	program := string(body)
	p := NewParser(program)
	pro := p.Program()

	dotify := newDotifier()
	err := dotify.gendot(pro)
	if err != nil {
		t.Error(err)
	}
}
