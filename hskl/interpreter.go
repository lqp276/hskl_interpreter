package hskl

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
)

const (
	Frame_Normal = iota + 1
	FrameRun_Return
	FrameRun_Break
)

type stackFrame struct {
	table   map[string]*vari
	level   int
	upLevel *stackFrame
	retVal  interface{}
	state   byte
	interp  *interpreter
}

func makeFrame(interp *interpreter, level int, upLevel *stackFrame) *stackFrame {
	tb := &stackFrame{level: level, upLevel: upLevel}
	tb.table = make(map[string]*vari)
	tb.interp = interp
	return tb
}

func (tb *stackFrame) insertVari(v *vari) {
	if tb.interp.debug {
		fmt.Printf("set variable: %v stack idx: %d\n", v, tb.level)
	}
	tb.table[v.name] = v
}

func (tb *stackFrame) lookup(name string, chained bool) *vari {
	if sym, ok := tb.table[name]; ok {
		return sym
	}

	if chained && tb.upLevel != nil {
		return tb.upLevel.lookup(name, chained)
	}

	return nil
}

type varClass interface {
	symName() string
	String() string
}

type vari struct {
	name  string
	type_ AstNode
	val   interface{}
}

func newIntVari(level int, ast *AstVarDecl) *vari {
	va := &vari{}
	va.name = ast.name
	va.type_ = ast.type_

	va.val = 0
	if len(ast.initVal) > 0 {
		va.val, _ = strconv.Atoi(ast.initVal)
	}

	return va
}

func newStringVari(level int, ast *AstVarDecl) *vari {
	va := &vari{}
	va.name = ast.name
	va.type_ = ast.type_

	va.val = ""
	if len(ast.initVal) > 0 {
		va.val = ast.initVal
	}

	return va
}

func newStructVari(level int, ast *AstVarDecl) *vari {
	va := &vari{}
	va.name = ast.name
	va.type_ = ast.type_

	mv := make(map[string]interface{})

	strctTp := va.type_.(*AstStructType)
	for _, field := range strctTp.fields {
		switch ftp := field.type_.(type) {
		case *AstPrimType:
			switch ftp.name {
			case symTypeInt:
				mv[field.name] = 0
				break

			case symTypeString:
				mv[field.name] = ""
				break
			}

			break

		default:
			mv[field.name] = nil
			break
		}
	}

	va.val = mv
	return va
}

func newArrayVari(level int, ast *AstVarDecl) *vari {
	va := &vari{}
	va.name = ast.name
	va.type_ = ast.type_

	arrTp := ast.type_.(*AstArrayType)

	switch elem := arrTp.elemType.(type) {
	case *AstPrimType:
		switch elem.name {
		case symTypeInt:
			valArr := []interface{}{}
			if ast.initArr != nil {
				for _, token := range ast.initArr {
					val, _ := strconv.Atoi(token.value)
					valArr = append(valArr, val)
				}
			}

			va.val = valArr
			break

		case symTypeString:
			valArr := []interface{}{}
			if ast.initArr != nil {
				for _, token := range ast.initArr {
					valArr = append(valArr, token.value)
				}
			}

			va.val = valArr
			break

		default:
			doPanic("newArrayVari error, unknown primitive elem type: %T", elem)
		}

	case *AstArrayType, *AstStructType:
		va.val = []interface{}{}
		break

	default:
		doPanic("newArrayVari error, unknown elem type: %T", elem)
	}

	return va
}

type interpreter struct {
	callStack []*stackFrame
	stackSize int
	curFrame  *stackFrame
	mainFunc  *AstFuncDecl
	debug     bool
}

func (interp *interpreter) pushStackFrame() *stackFrame {
	symTb := makeFrame(interp, interp.stackSize, interp.curFrame)
	interp.callStack = append(interp.callStack, symTb)
	interp.stackSize++
	interp.curFrame = symTb
	return interp.curFrame
}

func (interp *interpreter) popStackFrame() *stackFrame {
	popFrame := interp.curFrame
	interp.callStack = interp.callStack[:len(interp.callStack)-1]
	interp.stackSize--
	interp.curFrame = interp.callStack[len(interp.callStack)-1]
	interp.curFrame.state = popFrame.state
	return interp.curFrame
}

func (interp *interpreter) frameReturned() bool {
	return interp.curFrame.state == FrameRun_Return
}

func (interp *interpreter) frameBreaked() bool {
	return interp.curFrame.state == FrameRun_Break
}

func (interp *interpreter) visitProgram(program *AstProgram) {
	for _, decl := range program.decl_list {
		switch node := decl.(type) {
		case *AstVarDecl:
			interp.visitVarDecl(node)
			break

		case *AstFuncDecl:
			interp.visitFuncDecl(node)
			break

		case *AstTypeDef:
			break

		default:
			doPanic("unsupported ast type in program: %T", node)
		}
	}
}

func (interp *interpreter) visitVarDecl(node *AstVarDecl) {
	if sym := interp.curFrame.lookup(node.name, false); sym != nil {
		doPanic("error variable inited in level: %d, name: %s, type: %s, already exist: %s",
			interp.curFrame.level, node.name, node.type_, sym.name)
		return
	}

	//ok
	switch tp := node.type_.(type) {
	case *AstPrimType:
		switch tp.name {
		case symTypeInt:
			va := newIntVari(interp.curFrame.level, node)
			interp.curFrame.insertVari(va)
			break

		case symTypeString:
			va := newStringVari(interp.curFrame.level, node)
			interp.curFrame.insertVari(va)
			break

		case symTypeVoid:
			break

		default:
			doPanic("unknown AstPrimType when interpret: %s", node.type_)
		}
		break

	case *AstStructType:
		va := newStructVari(interp.curFrame.level, node)
		interp.curFrame.insertVari(va)
		break

	case *AstArrayType:
		va := newArrayVari(interp.curFrame.level, node)
		interp.curFrame.insertVari(va)
		break

	case *AstUndefType:
		ast := &AstVarDecl{}
		ast.initArr = node.initArr
		ast.initVal = node.initVal
		ast.line = node.line
		ast.name = node.name
		ast.type_ = tp.resolved
		interp.visitVarDecl(ast)
		break

	default:
		doPanic("unknown type when interpret: %s", node.type_)

	}
}

func (interp *interpreter) visitFuncDecl(node *AstFuncDecl) {
	if node.name == entryFunc {
		interp.mainFunc = node
	}
}

func (interp *interpreter) visitBuiltinPrint(node *AstFuncCall) interface{} {
	//lookup args
	val := interp.curFrame.lookup("format", false).val
	fmt.Printf("%v", val)
	return nil
}

func (interp *interpreter) visitBuiltinPrintn(node *AstFuncCall) interface{} {
	//lookup args
	val := interp.curFrame.lookup("format", false).val
	fmt.Printf("%v\n", val)
	return nil
}

func (interp *interpreter) visitBuiltinStr(node *AstFuncCall) interface{} {
	//lookup args
	val := interp.curFrame.lookup("val", false).val
	//fmt.Printf("builtin print: %v\n", val)
	return fmt.Sprintf("%v", val)
}

func (interp *interpreter) visitBuiltinInt(node *AstFuncCall) interface{} {
	//lookup args
	val := interp.curFrame.lookup("val", false).val
	switch tVal := val.(type) {
	case int:
		return tVal

	case string:
		iVal, _ := strconv.Atoi(tVal)
		return iVal

	default:
		doPanic("builtin int(): cant convert: %s", tVal)
		return nil
	}
}

func (interp *interpreter) visitBuiltinAppend(node *AstFuncCall) interface{} {
	vari := interp.curFrame.lookup("arr", false)

	if vari.val == nil {
		doPanic("hsk runtime error, nil reference: %s, line: %d", node.desc(), node.line)
		return nil
	}

	switch rType := vari.val.(type) {
	case []interface{}:
		elem := interp.curFrame.lookup("elem", false).val
		return append(rType, elem)

	case []int:
		elem := interp.curFrame.lookup("elem", false).val
		return append(rType, elem.(int))

	case []string:
		elem := interp.curFrame.lookup("elem", false).val
		return append(rType, elem.(string))

	default:
		doPanic("visit builtin append error, unknown array type: %T", rType)
		return nil
	}
}

func (interp *interpreter) visitBuiltinLen(node *AstFuncCall) int {
	vari := interp.curFrame.lookup("arr", false)
	arr := vari.val.([]interface{})
	return len(arr)
}

func (interp *interpreter) visitBuiltinFunc(node *AstFuncCall) interface{} {
	switch node.name {
	case Builtin_print:
		return interp.visitBuiltinPrint(node)

	case Builtin_printn:
		return interp.visitBuiltinPrintn(node)

	case Builtin_str:
		return interp.visitBuiltinStr(node)

	case Builtin_int:
		return interp.visitBuiltinInt(node)

	case Builtin_append:
		return interp.visitBuiltinAppend(node)

	case Builtin_len:
		return interp.visitBuiltinLen(node)

	default:
		doPanic("interpret built func failed, name: %s, call at line: %d", node.name, node.line)
		return nil
	}
}

func (interp *interpreter) visitFuncCall(node *AstFuncCall) interface{} {
	if node.ast.builtin {
		return interp.visitBuiltinFunc(node)
	}

	ret := interp.visitCodeBlock(node.ast.block)
	return ret
}

func (interp *interpreter) visitCodeBlockVars(node *AstCodeBlock) {
	//clear old first
	//interp.curFrame.table = map[string]*vari{}
	for _, varDecl := range node.vars {
		interp.visitVarDecl(varDecl)
	}
}

func (interp *interpreter) visitCodeBlockStatement(node *AstCodeBlock) interface{} {
	var ret interface{}
	ret = nil

eval_loop:
	for _, ast := range node.stat_list {
		switch stat := ast.(type) {
		case *AstAssgin, *AstBinOP, *AstUnaryOP, *AstIntConst, *AstVarNameRef, *AstFuncCall:
			interp.visitAst(ast)
			break

		case *AstReturn:
			ret = interp.visitReturn(stat)
			interp.curFrame.state = FrameRun_Return
			break eval_loop

		case *AstCodeBlock:
			interp.pushStackFrame()
			ret = interp.visitCodeBlock(stat)
			interp.popStackFrame()
			if interp.frameReturned() {
				return ret
			}
			break

		case *AstConditionBlock:
			ret = interp.visitConditionBlock(stat)
			if interp.frameReturned() || interp.frameBreaked() {
				return ret
			}
			break

		case *AstWhileBlock:
			ret = interp.visitWhileBlock(stat)
			if interp.frameReturned() {
				return ret
			}
			break

		case *AstBreak:
			interp.curFrame.state = FrameRun_Break
			return nil

		case *AstNoopStat:
			break

		default:
			doPanic("error ast in func block: %T", ast)
		}
	}

	return ret
}

func (interp *interpreter) visitCodeBlock(node *AstCodeBlock) interface{} {
	interp.visitCodeBlockVars(node)
	return interp.visitCodeBlockStatement(node)
}

func (interp *interpreter) conditionOk(val interface{}) bool {
	if val == nil {
		doPanic("conditionOk recv nil value")
		return false
	}

	switch tp := val.(type) {
	case int:
		if tp == 0 {
			return false
		} else {
			return true
		}
		break

	case string:
		if len(tp) > 0 {
			return true
		} else {
			return false
		}

		break

	default:
		doPanic("conditionOk recv unknown type: %T:%v", val, val)
		return false
	}

	return false
}

func (interp *interpreter) visitConditionBlock(node *AstConditionBlock) interface{} {
	cond := interp.conditionOk(interp.visitAst(node.cond))
	if cond {
		interp.pushStackFrame()
		realRet := interp.visitCodeBlock(node.block)
		interp.popStackFrame()

		return realRet
	} else {
		if node.altCondBlock != nil {
			return interp.visitConditionBlock(node.altCondBlock)
		} else if node.altBlock != nil {
			interp.pushStackFrame()
			realRet := interp.visitCodeBlock(node.altBlock)
			interp.popStackFrame()

			return realRet
		}
	}

	return nil
}

func (interp *interpreter) visitWhileBlock(node *AstWhileBlock) interface{} {
	var ret interface{}
	for interp.conditionOk(interp.visitAst(node.cond)) {

		interp.pushStackFrame()
		interp.visitCodeBlockVars(node.block)
		ret = interp.visitCodeBlockStatement(node.block)
		interp.popStackFrame()

		if interp.frameReturned() {
			break
		}

		if interp.frameBreaked() {
			//eat break state
			interp.curFrame.state = Frame_Normal
			break
		}
	}
	return ret
}

func (interp *interpreter) setAstVal(dst AstNode, val interface{}) {
	switch rTp := dst.(type) {
	case *AstIndexedRef:
		if ret := interp.visitAst(rTp.host); ret != nil {
			arr := ret.([]interface{})
			idx := interp.visitAst(rTp.index).(int)
			arr[idx] = val
		} else {
			interpPanic("hskl runtime error, nil reference: %s, line: %d", rTp.host.desc(), rTp.line)
		}
		break

	case *AstVarNameRef:
		sym := interp.curFrame.lookup(rTp.name, true)
		if sym == nil {
			doPanic("error in varRef, symbol not found: %s", rTp.name)
		}
		sym.val = val
		break

	case *AstDotRef:
		if ret := interp.visitAst(rTp.host); ret != nil {
			mv := ret.(map[string]interface{})
			mv[rTp.name] = val
		} else {
			interpPanic("hskl runtime error, nil reference: %s, line: %d", rTp.host.desc(), rTp.line)
		}

		break

	default:
		doPanic("error ast in setAstVal: %T", dst)
	}
}

func (interp *interpreter) visitAssign(node *AstAssgin) interface{} {
	var ret interface{}
	ret = interp.visitAst(node.expr)

	interp.setAstVal(node.dst, ret)
	return ret
}

func (interp *interpreter) visitBinOP(node *AstBinOP) interface{} {
	var lhs interface{}
	var rhs interface{}

	switch node.left.(type) {
	case *AstBinOP, *AstUnaryOP, *AstStringConst, *AstIntConst, *AstVarNameRef, *AstFuncCall:
		lhs = interp.visitAst(node.left)
		break

	default:
		doPanic("error in binop left, unknown ast type: %s, line: %d", node.left, node.line)
	}

	switch node.right.(type) {
	case *AstBinOP, *AstUnaryOP, *AstStringConst, *AstIntConst, *AstVarNameRef, *AstFuncCall:
		rhs = interp.visitAst(node.right)
		break

	default:
		doPanic("error in binop right, unknown ast type: %s, line: %d", node.right, node.line)
	}

	if s1, ok := lhs.(string); ok {
		//must be string add
		s2 := rhs.(string)
		return s1 + s2
	}

	lhv := lhs.(int)
	rhv := rhs.(int)

	//fmt.Printf("test: %d %s %d\n", lhv, node.op, rhv)
	switch node.op {
	case PLUS:
		return lhv + rhv

	case MINUS:
		return lhv - rhv

	case MUL:
		return lhv * rhv

	case DIV:
		if rhv != 0 {
			return lhv / rhv
		} else {
			interpPanic("div by zero: %s, line: %d", node.desc(), node.line)
		}

	case AND:
		if lhv == 0 {
			return lhv
		} else {
			return rhv
		}

	case OR:
		if lhv != 0 {
			return lhv
		} else {
			return rhv
		}

	case NOT:
		if lhv == 0 {
			return 1
		} else {
			return 0
		}

	case EQU:
		if lhv == rhv {
			return 1
		} else {
			return 0
		}

	case NEQ:
		if lhv != rhv {
			return 1
		} else {
			return 0
		}

	case LT:
		if lhv < rhv {
			return 1
		} else {
			return 0
		}
	case LTE:
		if lhv <= rhv {
			return 1
		} else {
			return 0
		}
	case GT:
		if lhv > rhv {
			return 1
		} else {
			return 0
		}
	case GTE:
		if lhv >= rhv {
			return 1
		} else {
			return 0
		}
	}

	return 0
}

func (interp *interpreter) visitUnaryOP(node *AstUnaryOP) interface{} {
	var rhs interface{}
	switch node.dst.(type) {
	case *AstBinOP, *AstUnaryOP, *AstIntConst, *AstVarNameRef:
		rhs = interp.visitAst(node.dst)
		break

	default:
		doPanic("error in unaryop dst, unknown ast type: %s, line: %d", node.dst, node.line)
	}

	intVal := rhs.(int)
	switch node.op {
	case PLUS:
		return rhs

	case MINUS:
		return -intVal

	case NOT:
		if intVal == 0 {
			return 1
		} else {
			return 0
		}

	default:
		doPanic("unknown unary operator: %s, line: %d", node.op, node.line)
	}

	return nil
}

func (se *interpreter) visitNewOP(node *AstNewOP) interface{} {
	switch tp := realType(node.opType).(type) {
	case *AstStructType:
		return make(map[string]interface{})

	case *AstArrayType:
		return []interface{}{}

	default:
		doPanic("error type when interpret new op: %s", tp.desc())
	}

	return nil
}

func (interp *interpreter) visitReturn(node *AstReturn) interface{} {
	if node.expr != nil {
		return interp.visitAst(node.expr)
	}

	return nil
}

func (interp *interpreter) visitIntConst(node *AstIntConst) interface{} {
	return node.value
}

func (interp *interpreter) visitStringConst(node *AstStringConst) interface{} {
	return node.value
}

func (interp *interpreter) visitIndexedRef(node *AstIndexedRef) interface{} {
	idxTp := interp.visitAst(node.index)
	primTp, ok := idxTp.(int)
	if !ok {
		interpPanic("hskl runtime error, nil reference: %s, line: %d", node.host.desc(), node.line)
		return nil
	}

	//check host
	hostTp := interp.visitAst(node.host)
	if hostTp == nil {
		interpPanic("hskl runtime error, nil array reference, line: %d", node.line)
		return nil
	}

	arrTp, ok := hostTp.([]interface{})
	if !ok {
		doPanic("error in indexedRef: %s, host should be array, actual: %T",
			node.host, arrTp)
		return nil
	}

	if primTp < len(arrTp) {
		return arrTp[primTp]
	}
	doPanic("error in indexedRef: %v, index out of bound, index: %d, arr len: %d",
		primTp, primTp, len(arrTp))
	return nil
}

func (interp *interpreter) visitDotRef(node *AstDotRef) interface{} {
	hType := interp.visitAst(node.host) //should return map
	if hType == nil {
		interpPanic("hskl runtime error, nil reference: %s, line: %d", node.host.desc(), node.line)
		return nil
	}

	mv, ok := hType.(map[string]interface{})
	if !ok {
		doPanic("error in dotRef: %s, host should be map[string]interface{}, actual: %T",
			node.host, mv)
		return nil
	}

	//check host
	return mv[node.name]
}

func (interp *interpreter) visitVarRef(node *AstVarNameRef) interface{} {
	sym := interp.curFrame.lookup(node.name, true)
	if sym == nil {
		doPanic("error in varRef, symbol not found: %s", node.name)
		return nil
	}

	return sym.val
}

func (interp *interpreter) DoInterpret(root AstNode) (result error) {
	defer func() {
		if r := recover(); r != nil {
			//stack := string(debug.Stack())
			//desc := r.(error).Error() + "\n" + stack
			if rte, ok := r.(*interpError); ok {
				result = errors.New(rte.msg)
			} else {
				result = errors.New(r.(error).Error())
			}
		}
	}()

	switch node := root.(type) {
	case *AstProgram:
		interp.visitProgram(node)
		break

	default:
		return errors.Errorf("root ast type should be program, actual recv: %T", root)
	}

	call := &AstFuncCall{}
	call.ast = interp.mainFunc
	call.name = entryFunc
	interp.pushStackFrame()
	interp.visitFuncCall(call)
	interp.popStackFrame()
	interp.curFrame.state = Frame_Normal
	return nil
}

func (interp *interpreter) visitAst(ast AstNode) interface{} {
	//fmt.Printf("visit ast: %T\n", ast)
	switch statement := ast.(type) {
	case *AstVarDecl:
		interp.visitVarDecl(statement)
		break

	case *AstFuncDecl:
		interp.visitFuncDecl(statement)
		break
	case *AstAssgin:
		return interp.visitAssign(statement)

	case *AstBinOP:
		return interp.visitBinOP(statement)

	case *AstUnaryOP:
		return interp.visitUnaryOP(statement)

	case *AstNewOP:
		return interp.visitNewOP(statement)

	case *AstReturn:
		return interp.visitReturn(statement)

	case *AstIntConst:
		return interp.visitIntConst(statement)

	case *AstStringConst:
		return interp.visitStringConst(statement)

	case *AstVarNameRef:
		return interp.visitVarRef(statement)

	case *AstIndexedRef:
		return interp.visitIndexedRef(statement)

	case *AstDotRef:
		return interp.visitDotRef(statement)

	case *AstFuncCall:
		//prepare for args
		args := []interface{}{}
		for _, argExp := range statement.args {
			arg := interp.visitAst(argExp)
			args = append(args, arg)
		}

		interp.pushStackFrame()
		for idx, param := range statement.ast.params {
			interp.curFrame.insertVari(&vari{name: param.name, type_: param.type_, val: args[idx]})
		}

		ret := interp.visitFuncCall(statement)
		//return and break not cross func boundary
		interp.popStackFrame().state = Frame_Normal
		return ret

	default:
		doPanic("unknown ast when interpret: %T", ast)
		break
	}

	return nil
}

func NewInterpreter() *interpreter {
	inter := &interpreter{}

	symTb := makeFrame(inter, 0, nil)
	inter.callStack = []*stackFrame{symTb}
	inter.stackSize = 1
	inter.curFrame = inter.callStack[0]
	return inter
}
