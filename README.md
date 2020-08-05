# hskl language interpreter
this is a simple interpreter I build just for fun, the implemented language is called 'hskl'

## how to play
```
//use 'hello world' source file under data directory:
go run hskl.go ./data/helloworld.hskl

//caculate fibonacci sequence
go run hskl.go ./data/fibonacci.hskl
```
## features
* builtin data type: int string, array
* arithmetic operator: + - * /
* logic operator: && || ! < <= > >=
* user defined struct
* use defined function 
