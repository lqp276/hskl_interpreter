package hskl

import (
	"fmt"
	"runtime/debug"

	"github.com/pkg/errors"
)

const ()

type symbolTable struct {
	table   map[string]symbolClass
	level   int
	upLevel *symbolTable
}

func newSymTable(level int, upLevel *symbolTable) *symbolTable {
	tb := &symbolTable{level: level, upLevel: upLevel}
	tb.table = make(map[string]symbolClass)
	tb.table[symTypeInt] = newBuiltinSymbol(symTypeInt)
	tb.table[symTypeString] = newBuiltinSymbol(symTypeString)
	return tb
}

func (tb *symbolTable) insertSymbol(sym symbolClass, debug bool) {
	if _, ok := tb.table[sym.symName()]; ok {
		doPanic("duplicate symbol: '%s'", sym.symName())
	}

	if debug {
		fmt.Printf("insert symbol: %v level: %d\n", sym, tb.level)
	}
	tb.table[sym.symName()] = sym
}

func (tb *symbolTable) lookup(name string, chained bool) symbolClass {
	if sym, ok := tb.table[name]; ok {
		return sym
	}

	if chained && tb.upLevel != nil {
		return tb.upLevel.lookup(name, chained)
	}

	return nil
}

type symbolClass interface {
	symName() string
	String() string
}

type symbol struct {
	name  string
	cate  string
	type_ AstType
	level int
}

type builtinSymbol struct {
	symbol
}

func (sym *builtinSymbol) String() string {
	return fmt.Sprintf("builtinSymbol{%s %s}", sym.name, sym.type_)
}

func (sym *builtinSymbol) symName() string {
	return sym.name
}

func newBuiltinSymbol(name string) *builtinSymbol {
	sym := &builtinSymbol{}
	sym.name = name
	sym.type_ = nil
	sym.level = 0
	return sym
}

type varSymbol struct {
	symbol
	ast *AstVarDecl
}

func (sym *varSymbol) String() string {
	ast := sym.ast
	return fmt.Sprintf("varSymbol{%s %s '%s' @line %d}", sym.name, ast.type_, ast.initVal, ast.line)
}

func (sym *varSymbol) symName() string {
	return sym.name
}

func newVarSymbol(name string, type_ AstType, lvl int, ast *AstVarDecl) *varSymbol {
	sym := &varSymbol{}
	sym.name = name
	if rtype, ok := type_.(*AstUndefType); ok {
		sym.type_ = rtype.resolved
	} else {
		sym.type_ = type_
	}
	sym.level = lvl
	sym.ast = ast
	return sym
}

type funcSymbol struct {
	symbol
	ast *AstFuncDecl
}

func (sym *funcSymbol) String() string {
	ast := sym.ast
	return fmt.Sprintf("funcSymbol{%s var decl: %d @line %d}", sym.name, len(ast.params), ast.line)
}

func (sym *funcSymbol) symName() string {
	return sym.name
}

func newFuncSymbol(name string, lvl int, ast *AstFuncDecl) *funcSymbol {
	sym := &funcSymbol{}
	sym.name = name
	sym.cate = "func"
	sym.level = lvl
	sym.ast = ast
	return sym
}

type semanticAnalyzer struct {
	symbolStack    []*symbolTable
	stackSize      int
	curSymbolTable *symbolTable
	firstPass      bool
	debug          bool
	brkStack       []bool
}

func (p *semanticAnalyzer) pushBrk() {
	p.brkStack = append(p.brkStack, true)
}

func (p *semanticAnalyzer) popBrk() {
	p.brkStack = p.brkStack[:len(p.brkStack)-1]
}

func (p *semanticAnalyzer) allowBrk() bool {
	return len(p.brkStack) > 0
}

func (se *semanticAnalyzer) pushSymbolTable() *symbolTable {
	symTb := newSymTable(se.stackSize, se.curSymbolTable)
	se.symbolStack = append(se.symbolStack, symTb)
	se.stackSize += 1
	se.curSymbolTable = symTb
	return se.curSymbolTable
}

func (se *semanticAnalyzer) popSymbolTable() *symbolTable {
	se.symbolStack = se.symbolStack[:len(se.symbolStack)-1]
	se.stackSize -= 1
	se.curSymbolTable = se.symbolStack[len(se.symbolStack)-1]
	return se.curSymbolTable
}

func (se *semanticAnalyzer) visitProgram(program *AstProgram) {
	se.resolveTypes(program)

	for _, decl := range program.decl_list {
		switch node := decl.(type) {
		case *AstVarDecl:
			se.visitVarDecl(node)
			break

		case *AstFuncDecl:
			se.visitFuncDecl(node)
			break

		case *AstTypeDef:
			break

		default:
			doPanic("unsupported ast type in program: %T", node)
		}
	}

	se.firstPass = false
	for _, decl := range program.decl_list {
		switch node := decl.(type) {
		case *AstFuncDecl:
			se.visitFuncDecl(node)
			break
		}
	}
}

func (se *semanticAnalyzer) visitVarDecl(node *AstVarDecl) {
	if sym := se.curSymbolTable.lookup(node.name, false); sym != nil {
		doPanic("error var symbol defined in level: %d, name: %s, type: %s, already exist: %s",
			se.curSymbolTable.level, node.name, node.type_, sym.symName())
		return
	}

	if primTp, ok := node.type_.(*AstPrimType); ok {
		if tp := se.curSymbolTable.lookup(primTp.name, true); tp == nil {
			doPanic("variable type not defined in level: %d, name: %s", se.curSymbolTable.level, node.type_)
			return
		}
	}

	//ok
	sym := newVarSymbol(node.name, node.type_, se.curSymbolTable.level, node)
	se.curSymbolTable.insertSymbol(sym, se.debug)
}

func (se *semanticAnalyzer) visitFuncDecl(node *AstFuncDecl) {
	//fmt.Printf("visit func decl: %s, line: %d\n", node.name, node.line)

	if sym := se.curSymbolTable.lookup(node.name, false); se.firstPass && sym != nil {
		doPanic("error func symbol defined in level: %d, name: %s, already exist: %s",
			se.curSymbolTable.level, node.name, sym.symName())
		return
	}

	se.pushSymbolTable()
	if !se.firstPass {
		for _, varDecl := range node.params {
			se.visitVarDecl(varDecl)
		}

		ret := se.visitCodeBlock(node.block).(AstType)
		if ret.signature() != node.retType.signature() {
			doPanic("return type not match in func: %s, want: %s, actual: %s",
				node.name, node.retType, ret)
		}
	}

	se.popSymbolTable()
	if se.firstPass {
		funcSym := newFuncSymbol(node.name, se.curSymbolTable.level, node)
		se.curSymbolTable.insertSymbol(funcSym, se.debug)
	}
}

func (se *semanticAnalyzer) visitCodeBlock(node *AstCodeBlock) interface{} {
	for _, varDecl := range node.vars {
		se.visitVarDecl(varDecl)
	}

	var ret AstType
	ret = &AstPrimType{name: symTypeVoid}

eval_loop:
	for _, ast := range node.stat_list {
		switch stat := ast.(type) {
		case *AstAssgin, *AstBinOP, *AstUnaryOP, *AstIntConst, *AstVarNameRef:
			se.visitAst(ast)
			break

		case *AstFuncCall:
			se.visitFuncCall(stat)
			break

		case *AstBreak:
			if !se.allowBrk() {
				doPanic("break has not exit point")
			}
			break

		case *AstReturn:
			ret = se.visitReturn(stat).(AstType)
			break eval_loop

		case *AstCodeBlock:
			se.pushSymbolTable()
			ret = se.visitCodeBlock(stat).(AstType)
			se.popSymbolTable()
			break

		case *AstConditionBlock:
			ret = se.visitConditionBlock(stat).(AstType)
			break

		case *AstWhileBlock:
			se.pushBrk()
			ret = se.visitWhileBlock(stat).(AstType)
			se.popBrk()
			break

		case *AstNoopStat:
			break

		default:
			doPanic("error ast in func block: %T", ast)
		}
	}

	return ret
}

func (se *semanticAnalyzer) visitConditionBlock(node *AstConditionBlock) interface{} {
	var normRet AstType
	normRet = &AstPrimType{name: symTypeVoid}
	realRet := normRet

	se.visitAst(node.cond)
	se.pushSymbolTable()
	realRet = se.visitCodeBlock(node.block).(AstType)
	se.popSymbolTable()

	if node.altCondBlock != nil {
		ret := se.visitConditionBlock(node.altCondBlock).(AstType)
		if ret.signature() != "V" && ret.signature() != realRet.signature() {
			doPanic("return diffrent type, ret1: %v, ret2: %v", realRet, ret)
			return nil
		}
	}

	if node.altBlock != nil {
		se.pushSymbolTable()
		ret := se.visitCodeBlock(node.altBlock).(AstType)
		se.popSymbolTable()

		if ret.signature() != "V" && ret.signature() != realRet.signature() {
			doPanic("return diffrent type, ret1: %v, ret2: %v", realRet, ret)
			return nil
		}
	}

	return realRet
}

func (se *semanticAnalyzer) visitWhileBlock(node *AstWhileBlock) interface{} {
	se.visitAst(node.cond)
	se.pushSymbolTable()
	ret := se.visitCodeBlock(node.block)
	se.popSymbolTable()

	return ret
}

func isAnyType(ast AstNode) bool {
	if tp, ok := ast.(*AstPrimType); ok {
		if tp.name == symTypeAny {
			return true
		}
	}

	return false
}

func isCompatiWithPrimType(primTp *AstPrimType, has interface{}) bool {
	if primTp.name == symTypeAny {
		return true
	}

	return false
}

func isTypeCompatiable(want string, has interface{}) bool {
	if has == nil {
		return false
	}

	strHas, ok := has.(string)
	if !ok {
		return false
	}

	p1 := newSigParser(want)
	p2 := newSigParser(strHas)

	for {
		s1 := p1.getNextElem()
		s2 := p2.getNextElem()

		if s1 == nil && s2 == nil {
			return true
		}

		if s1 == nil {
			return false
		}

		if s1.tp == symTypeAny {
			return true
		}

		//require exact match
		if s1.tp != s2.tp {
			return false
		}

		//same type, check value
		if s1.value != s2.value {
			return false
		}
	}

	return false
}

func (se *semanticAnalyzer) visitFuncCall(node *AstFuncCall) interface{} {
	sym := se.curSymbolTable.lookup(node.name, true)
	if sym == nil {
		doPanic("undefined func: %s, line: %d", node.name, node.line)
		return nil
	}

	fdef, ok := sym.(*funcSymbol)
	if !ok {
		doPanic("error func call, target is not func decl: %s, line: %d, dst: %T", node.name, node.line, sym)
		return nil
	}

	node.ast = fdef.ast

	argLen := len(node.args)
	paramLen := len(node.ast.params)
	if node.ast.va_param == nil && argLen != paramLen {
		doPanic("error func call, param count not match, need: %d, actual: %d, func: %s, line: %d",
			paramLen, argLen, node.name, node.line)
		return nil
	}

	for idx, ast := range node.args {
		get := se.visitAst(ast)
		want := node.ast.params[idx].type_
		if isTypeCompatiable(want.signature(), get) {
			doPanic("error func call, arg type not match, idx: %d, need: %s, actual: %s, func: %s, line: %d",
				idx, want, get, node.name, node.line)
			return nil
		}
	}

	if fdef.ast.fixRetType == nil {
		return fdef.ast.retType
	}
	retTp := se.visitAst(fdef.ast.fixRetType(node))
	return retTp
}

func (se *semanticAnalyzer) visitAssign(node *AstAssgin) interface{} {
	// name := ""
	// sym := se.curSymbolTable.lookup(name, true)
	// if sym == nil {
	// 	doPanic("undefined var: %s, line: %d", name, node.line)
	// 	return nil
	// }

	var dstType AstType
	switch node.dst.(type) {
	case *AstIndexedRef, *AstVarNameRef, *AstDotRef:
		dstType = se.visitAst(node.dst).(AstType)
		break

	default:
		doPanic("error ast in func block: %T", node.dst)
	}

	var ret AstType
	switch node.expr.(type) {
	case *AstBinOP, *AstUnaryOP, *AstNewOP,
		*AstIntConst, *AstStringConst,
		*AstIndexedRef, *AstDotRef, *AstVarNameRef,
		*AstFuncCall:
		ret = se.visitAst(node.expr).(AstType)
		break

	default:
		doPanic("error ast in func block: %T", node.expr)
	}

	lhs := dstType.signature()
	rhs := ret.signature()

	//fmt.Printf("match assign left: %s, right: %s\n", lhs, rhs)
	if lhs != rhs {
		doPanic("assign with diffirent type, lhs: %s, rhs: %s, line: %d", lhs, rhs, node.line)
	}

	return nil
}

func (se *semanticAnalyzer) visitReturn(node *AstReturn) interface{} {
	var ret AstType
	ret = &AstPrimType{name: symTypeVoid}
	if node.expr != nil {
		ret = se.visitAst(node.expr).(AstType)
	}

	return ret
}

func (se *semanticAnalyzer) visitBinOP(node *AstBinOP) interface{} {
	var lhs AstType
	var rhs AstType
	switch node.left.(type) {
	case *AstBinOP, *AstUnaryOP, *AstDotRef, *AstIndexedRef,
		*AstStringConst, *AstIntConst, *AstVarNameRef, *AstFuncCall:
		lhs = se.visitAst(node.left).(AstType)
		break

	default:
		doPanic("error in binop left, unknown ast type: %s, line: %d", node.left, node.line)
	}

	switch node.right.(type) {
	case *AstBinOP, *AstUnaryOP, *AstDotRef, *AstIndexedRef,
		*AstStringConst, *AstIntConst, *AstVarNameRef, *AstFuncCall:
		rhs = se.visitAst(node.right).(AstType)
		break

	default:
		doPanic("error in binop right, unknown ast type: %s, line: %d", node.right, node.line)
	}

	if lhs.signature() != rhs.signature() {
		if lhs.signature() == "S" && node.op == PLUS {
			//insert str call
			strAst := &AstFuncCall{}
			strAst.args = append(strAst.args, node.right)
			strAst.name = Builtin_str
			strAst.line = node.line

			node.right = strAst
			return se.visitBinOP(node)
		}

		doPanic("assign with incompatiable type, lhs: %s, rhs: %s, line: %d", lhs, rhs, node.line)
		return nil
	}

	sp := newSigParser(lhs.signature())
	first := sp.getNextElem()
	if first == nil {
		doPanic("get signature elem error, lhs: %s, rhs: %s, line: %d", lhs, rhs, node.line)
		return nil
	}

	switch first.tp {
	case symTypeInt:
		break

	case symTypeVoid, symTypeAny, symTypeArray, symTypeStruct:
		doPanic("error binop on type: %s, lhs: %s, rhs: %s, line: %d", first.tp, lhs, rhs, node.line)
		break

	case symTypeString:
		if node.op != PLUS {
			doPanic("string type only allow add, lhs: %s, rhs: %s, line: %d", lhs, rhs, node.line)
		}
		break

	default:
		doPanic("error binop on type: %s, lhs: %s, rhs: %s, line: %d", first.tp, lhs, rhs, node.line)
	}

	return lhs
}

func (se *semanticAnalyzer) visitUnaryOP(node *AstUnaryOP) interface{} {
	var rhs AstType
	switch node.dst.(type) {
	case *AstBinOP, *AstUnaryOP, *AstIntConst, *AstVarNameRef:
		rhs = se.visitAst(node.dst).(AstType)
		break

	default:
		doPanic("error in unaryop dst, unknown ast type: %s, line: %d", node.dst, node.line)
	}

	return rhs
}

func (se *semanticAnalyzer) visitNewOP(node *AstNewOP) interface{} {
	tp := realType(node.opType)
	switch rtp := tp.(type) {
	case *AstPrimType:
		doPanic("error type in new operator: %s, line: %d", rtp.desc(), node.line)
		break
	}

	return tp
}

func (se *semanticAnalyzer) visitIntConst(node *AstIntConst) interface{} {
	return &AstPrimType{name: symTypeInt}
}

func (se *semanticAnalyzer) visitStringConst(node *AstStringConst) interface{} {
	return &AstPrimType{name: symTypeString}
}

func (se *semanticAnalyzer) visitIndexedRef(node *AstIndexedRef) interface{} {
	idxTp := se.visitAst(node.index)
	primTp, ok := idxTp.(*AstPrimType)
	if !ok || primTp.name != symTypeInt {
		doPanic("error in indexedRef: %s, index should be int, actual: %s",
			node.host, primTp)
		return nil
	}

	//check host
	hostTp := se.visitAst(node.host)
	arrTp, ok := hostTp.(*AstArrayType)
	if !ok {
		doPanic("error in indexedRef: %s, host should be array, actual: %s",
			node.host, arrTp)
		return nil
	}

	return arrTp.elemType
}

func (se *semanticAnalyzer) visitDotRef(node *AstDotRef) interface{} {
	hType := se.visitAst(node.host)
	strctTp, ok := hType.(*AstStructType)
	if !ok {
		doPanic("error in dotRef: %s, host should be struct, actual: %s",
			node.host, hType)
		return nil
	}

	//check host
	for _, field := range strctTp.fields {
		if field.name == node.name {
			return realType(field.type_)
		}
	}

	doPanic("visit dotRef error, struct %s has no field: %s", strctTp.name, node.name)
	return nil
}

func (se *semanticAnalyzer) visitVarRef(node *AstVarNameRef) interface{} {
	sym := se.curSymbolTable.lookup(node.name, true)
	if sym == nil {
		doPanic("error in varRef, symbol not found: %s", node.name)
		return nil
	}

	if varSym, ok := sym.(*varSymbol); ok {
		return realType(varSym.type_)
	} else {
		doPanic("error in varRef, name: %s, line: %d, %T", node.name, node.line, sym)
		return nil
	}
}

func (se *semanticAnalyzer) resolveTypes(pro *AstProgram) {
	//add user defined type to type table

	for {
		fixArr := []*AstUndefType{}
		for k, v := range pro.tpMap {
			switch node := v.(type) {
			case *AstPrimType, *AstArrayType, *AstStructType:
				//fmt.Printf("skip resolve type: %s\n", node)
				break

			case *AstUndefType:
				if node.resolved != nil {
					continue
				}

				if rs, ok := pro.tpMap[node.name]; ok {
					if rs.astType() != AST_TP_UNDEF_TYPE {
						node.resolved = rs
						//fmt.Printf("-->resolve type: %s -> %s, ref by: %s\n", node.name, rs, k)
						fixArr = append(fixArr, node)
					} else {
						if undefTp := rs.(*AstUndefType); undefTp.resolved != nil {
							//fmt.Printf("==>resolve type: %s -> %s, ref by: %s\n", node.name, rs, k)
							node.resolved = undefTp.resolved
							fixArr = append(fixArr, node)
						}
					}
				} else {
					doPanic("undefined type, name: %s", node.name)
				}
				break

			default:
				doPanic("unsupported ast type in type map: %T, name: %s", node, k)
			}
		}

		if len(fixArr) == 0 {
			break
		}
	}

	for k, v := range pro.tpMap {
		switch node := v.(type) {
		case *AstUndefType:
			if node.resolved == nil {
				doPanic("unresolved type, name: %s, ref by: %s", node.name, k)
			}

			//fmt.Printf("indirect collected type: %s -> %s\n", k, node.resolved)
			break

		default:
			//fmt.Printf("collected type: [%s] %s\n", k, node)
		}
	}
}

func (se *semanticAnalyzer) DoAnalyze(root AstNode) (result error) {
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			desc := r.(error).Error() + "\n" + stack
			result = errors.New(desc)
		}
	}()

	switch node := root.(type) {
	case *AstProgram:
		se.visitProgram(node)
		break

	default:
		return errors.Errorf("root ast type should be program, actual recv: %T", root)
	}

	main := se.curSymbolTable.lookup(entryFunc, false)
	if main == nil {
		return fmt.Errorf("main func is not defined")
	}

	if funcDecl, ok := main.(*funcSymbol); ok {
		if len(funcDecl.ast.params) > 0 {
			return errors.Errorf("func 'main' has params count > 0")
		}
	} else {
		return errors.Errorf("'main' is not func symbol, actual type: %T", main)
	}

	return nil
}

func (se *semanticAnalyzer) visitAst(ast AstNode) interface{} {
	switch statement := ast.(type) {
	case *AstVarDecl:
		se.visitVarDecl(statement)
		break

	case *AstFuncDecl:
		se.visitFuncDecl(statement)
		break
	case *AstAssgin:
		return se.visitAssign(statement)

	case *AstBinOP:
		return se.visitBinOP(statement)

	case *AstUnaryOP:
		return se.visitUnaryOP(statement)

	case *AstStringConst:
		return se.visitStringConst(statement)

	case *AstIntConst:
		return se.visitIntConst(statement)

	case *AstVarNameRef:
		return se.visitVarRef(statement)

	case *AstFuncCall:
		return se.visitFuncCall(statement)

	case *AstIndexedRef:
		return se.visitIndexedRef(statement)

	case *AstDotRef:
		return se.visitDotRef(statement)

	case *AstNewOP:
		return se.visitNewOP(statement)

	default:
		doPanic("unknown ast type: %T", ast)
		break
	}

	return nil
}

func NewSemanticAnalyzer() *semanticAnalyzer {
	se := &semanticAnalyzer{}

	symTb := newSymTable(0, nil)
	se.symbolStack = []*symbolTable{symTb}
	se.stackSize = 1
	se.curSymbolTable = se.symbolStack[0]
	se.brkStack = []bool{}

	for _, bfc := range getBuiltinFunc() {
		symFunc := newFuncSymbol(bfc.name, 0, bfc)
		se.curSymbolTable.insertSymbol(symFunc, se.debug)
	}

	se.firstPass = true
	return se
}
