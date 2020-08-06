package main

import (
	"fmt"
	"hskl/hskl"
	"io/ioutil"
	"os"
)

func main() {
	if len(os.Args) <= 1 || len(os.Args[1]) == 0 {
		fmt.Printf("you should specify the source file\n")
		return
	}

	//fmt.Printf("args: %v\n", os.Args)

	body, _ := ioutil.ReadFile(os.Args[1])
	program := string(body)

	//fmt.Printf("%s", program)
	p := hskl.NewParser(program)
	analyzer := hskl.NewSemanticAnalyzer()

	pro := p.Program()
	err := analyzer.DoAnalyze(pro)
	if err != nil {
		fmt.Printf("analyze error: %v\n", err)
		return
	}

	interp := hskl.NewInterpreter()
	err = interp.DoInterpret(pro)

	if err != nil {
		fmt.Printf("interpret error: %v\n", err)
	}
}
