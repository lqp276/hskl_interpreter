package hskl

import (
	"fmt"
	"runtime/debug"

	"github.com/pkg/errors"
)

const (
	AST_Program = iota + 1
	AST_INT_CONST
	AST_STRING_CONST
	AST_VarDecl
	AST_FuncDecl
	AST_BuiltinFunc
	AST_Assign
	AST_BinOp
	AST_NewOp
	AST_UnaryOp
	AST_Noop
	AST_CODE_BLOCK
	AST_VAR_REF
	AST_DOT_REF
	AST_INDEXED_REF
	AST_CONDITION_BLOCK
	AST_WHILE
	AST_BREAK

	//data type
	AST_TP_PRIMITIVE
	AST_TP_ARRAY
	AST_TP_STRUCT
	AST_TP_TYPE_DEF
	AST_TP_TYPE_REF
	AST_TP_UNDEF_TYPE
)

var verbPanic bool

func doPanic(format string, args ...interface{}) {
	desc := fmt.Sprintf(format, args...)
	if verbPanic {
		stack := string(debug.Stack())
		desc += "\nstacktrace: " + stack
	}
	panic(errors.New(desc))
}

type interpError struct {
	msg string
}

func (iterr *interpError) Error() string {
	return iterr.msg
}

func interpPanic(format string, args ...interface{}) {
	desc := fmt.Sprintf(format, args...)
	panic(&interpError{desc})
}

type astVisitor interface {
	visitProgram(node *AstProgram)
	visitVarDecl(node *AstVarDecl)
	visitFuncDecl(node *AstFuncDecl)
	visitAssign(node *AstAssgin)
	visitBinOP(node *AstBinOP) interface{}
	visitUnaryOP(node *AstUnaryOP) interface{}
	visitVarRef(node *AstVarNameRef) interface{}
}

type AstNode interface {
	astType() int
	String() string
	desc() string
}

type AstBase struct {
	seq int
}

type AstProgram struct {
	AstBase
	tpMap     map[string]AstType
	decl_list []AstNode
}

func (ast *AstProgram) astType() int {
	return AST_Program
}

func (ast *AstProgram) String() string {
	return fmt.Sprintf("AstProgram")
}

func (ast *AstProgram) desc() string {
	return fmt.Sprintf("not impl")
}

type AstVarDecl struct {
	AstBase
	name    string
	type_   AstType
	initVal string
	initArr []*Token
	line    int
}

func (ast *AstVarDecl) astType() int {
	return AST_VarDecl
}

func (ast *AstVarDecl) String() string {
	return fmt.Sprintf("AstVarDecl")
}

func (ast *AstVarDecl) desc() string {
	return fmt.Sprintf("var %s:%s", ast.name, ast.type_.signature())
}

type AstFuncDecl struct {
	AstBase
	name       string
	retType    AstType
	params     []*AstVarDecl
	va_param   *string
	block      *AstCodeBlock
	builtin    bool
	line       int
	fixRetType func(fn *AstFuncCall) AstNode
}

func (ast *AstFuncDecl) astType() int {
	return AST_FuncDecl
}

func (ast *AstFuncDecl) String() string {
	return fmt.Sprintf("AstFuncDecl")
}

func (ast *AstFuncDecl) desc() string {
	return fmt.Sprintf("func: %s", ast.name)
}

type AstFuncCall struct {
	AstBase
	name string
	args []AstNode
	line int
	ast  *AstFuncDecl
}

func (ast *AstFuncCall) astType() int {
	return AST_FuncDecl
}

func (ast *AstFuncCall) String() string {
	return fmt.Sprintf("AstFuncCall")
}

func (ast *AstFuncCall) desc() string {
	return fmt.Sprintf("%s()", ast.name)
}

type AstCodeBlock struct {
	AstBase
	vars      []*AstVarDecl
	stat_list []AstNode
}

func (ast *AstCodeBlock) astType() int {
	return AST_CODE_BLOCK
}

func (ast *AstCodeBlock) String() string {
	return fmt.Sprintf("AstCompoundStat")
}

func (ast *AstCodeBlock) desc() string {
	return fmt.Sprintf("not impl")
}

type AstAssgin struct {
	AstBase
	dst  AstNode
	expr AstNode
	line int
}

func (ast *AstAssgin) astType() int {
	return AST_Assign
}

func (ast *AstAssgin) String() string {
	return fmt.Sprintf("AstAssgin")
}

func (ast *AstAssgin) desc() string {
	return fmt.Sprintf("%s = %s", ast.dst.desc(), ast.expr.desc())
}

type AstBinOP struct {
	AstBase
	op    string
	left  AstNode
	right AstNode
	line  int
}

func (ast *AstBinOP) astType() int {
	return AST_BinOp
}

func (ast *AstBinOP) String() string {
	return fmt.Sprintf("AstBinOP")
}

func (ast *AstBinOP) desc() string {
	return fmt.Sprintf("(%s) %s (%s)", ast.left.desc(), ast.op, ast.right.desc())
}

type AstNewOP struct {
	AstBase
	opType AstType
	line   int
}

func (ast *AstNewOP) astType() int {
	return AST_NewOp
}

func (ast *AstNewOP) String() string {
	return fmt.Sprintf("AstNewOP")
}

func (ast *AstNewOP) desc() string {
	return fmt.Sprintf("new: %s", ast.opType.desc())
}

type AstUnaryOP struct {
	AstBase
	op   string
	dst  AstNode
	line int
}

func (ast *AstUnaryOP) astType() int {
	return AST_UnaryOp
}

func (ast *AstUnaryOP) String() string {
	return fmt.Sprintf("AstUnaryOP")
}

func (ast *AstUnaryOP) desc() string {
	return fmt.Sprintf("%s%s", ast.op, ast.dst.desc())
}

type AstReturn struct {
	AstBase
	expr AstNode
	line int
}

func (ast *AstReturn) astType() int {
	return AST_UnaryOp
}

func (ast *AstReturn) String() string {
	return fmt.Sprintf("AstReturn")
}

func (ast *AstReturn) desc() string {
	return fmt.Sprintf("return: %s", ast.expr.desc())
}

type AstNoopStat struct {
}

func (ast *AstNoopStat) astType() int {
	return AST_Noop
}

func (ast *AstNoopStat) String() string {
	return fmt.Sprintf("AstNoopStat")
}

func (ast *AstNoopStat) desc() string {
	return fmt.Sprintf("not impl")
}

type AstIntConst struct {
	AstBase
	value int
	line  int
}

func (ast *AstIntConst) astType() int {
	return AST_STRING_CONST
}

func (ast *AstIntConst) String() string {
	return fmt.Sprintf("AstIntConst")
}

func (ast *AstIntConst) desc() string {
	return fmt.Sprintf("int const: %d", ast.value)
}

type AstStringConst struct {
	AstBase
	value string
	line  int
}

func (ast *AstStringConst) astType() int {
	return AST_STRING_CONST
}

func (ast *AstStringConst) String() string {
	return fmt.Sprintf("AstStringConst")
}

func (ast *AstStringConst) desc() string {
	return fmt.Sprintf("string const: %s", ast.value)
}

type AstVarNameRef struct {
	AstBase
	name string
	line int
}

func (ast *AstVarNameRef) astType() int {
	return AST_VAR_REF
}

func (ast *AstVarNameRef) String() string {
	return fmt.Sprintf("AstVarRef")
}

func (ast *AstVarNameRef) desc() string {
	return fmt.Sprintf("%s", ast.name)
}

type AstDotRef struct {
	AstBase
	host AstNode
	name string
	line int
}

func (ast *AstDotRef) astType() int {
	return AST_DOT_REF
}

func (ast *AstDotRef) String() string {
	return fmt.Sprintf("AstDotRef")
}

func (ast *AstDotRef) desc() string {
	return fmt.Sprintf("%s.%s", ast.host.desc(), ast.name)
}

type AstIndexedRef struct {
	AstBase
	host  AstNode
	index AstNode
	line  int
}

func (ast *AstIndexedRef) astType() int {
	return AST_INDEXED_REF
}

func (ast *AstIndexedRef) String() string {
	return fmt.Sprintf("AstIndexedRef")
}

func (ast *AstIndexedRef) desc() string {
	return fmt.Sprintf("%s[%s]", ast.host.desc(), ast.index.desc())
}

type AstConditionBlock struct {
	AstBase

	first bool
	cond  AstNode
	block *AstCodeBlock

	altCondBlock *AstConditionBlock
	altBlock     *AstCodeBlock

	outterFunc *AstFuncDecl
}

func (ast *AstConditionBlock) astType() int {
	return AST_CONDITION_BLOCK
}

func (ast *AstConditionBlock) String() string {
	return fmt.Sprintf("AstVarRef")
}

func (ast *AstConditionBlock) desc() string {
	return fmt.Sprintf("if")
}

type AstWhileBlock struct {
	AstBase
	cond  AstNode
	block *AstCodeBlock
}

func (ast *AstWhileBlock) astType() int {
	return AST_WHILE
}

func (ast *AstWhileBlock) String() string {
	return fmt.Sprintf("AstWhileBlock")
}

func (ast *AstWhileBlock) desc() string {
	return fmt.Sprintf("while")
}

type AstBreak struct {
	AstBase
}

func (ast *AstBreak) astType() int {
	return AST_BREAK
}

func (ast *AstBreak) String() string {
	return fmt.Sprintf("AstBreak")
}

func (ast *AstBreak) desc() string {
	return fmt.Sprintf("break")
}

type AstTypeDef struct {
	AstBase
	name string
	impl AstType
}

func (ast *AstTypeDef) astType() int {
	return AST_TP_TYPE_DEF
}

func (ast *AstTypeDef) String() string {
	return fmt.Sprintf("AstTypeDef")
}

func (ast *AstTypeDef) desc() string {
	return fmt.Sprintf("not impl")
}

type AstType interface {
	AstNode
	signature() string
}

type AstPrimType struct {
	name string
}

func (ast *AstPrimType) astType() int {
	return AST_TP_PRIMITIVE
}

func (ast *AstPrimType) String() string {
	return fmt.Sprintf("AstPrimType: " + ast.name)
}

func (ast *AstPrimType) signature() string {
	switch ast.name {
	case symTypeAny:
		return "*"

	case symTypeVoid:
		return "V"

	case symTypeInt:
		return "I"

	case symTypeString:
		return "S"

	default:
		doPanic("unknown primitive type: " + ast.name)
	}
	return "-"
}

func (ast *AstPrimType) desc() string {
	return ast.name
}

func newPrimType(name string) *AstPrimType {
	return &AstPrimType{name: name}
}

type AstArrayType struct {
	elemType AstType
}

func (ast *AstArrayType) astType() int {
	return AST_TP_ARRAY
}

func (ast *AstArrayType) String() string {
	return fmt.Sprintf("AstArrayType")
}

func (ast *AstArrayType) signature() string {
	return "[" + ast.elemType.signature()
}

func (ast *AstArrayType) desc() string {
	return "[" + ast.elemType.desc()
}

type AstStructType struct {
	name   string
	fields []*AstVarDecl
}

func (ast *AstStructType) astType() int {
	return AST_TP_STRUCT
}

func (ast *AstStructType) String() string {
	return fmt.Sprintf("AstStructType: " + ast.name)
}

func (ast *AstStructType) signature() string {
	return "s" + ast.name + ";"
}

func (ast *AstStructType) desc() string {
	return "struct: " + ast.name
}

type AstUndefType struct {
	name     string
	resolved AstType
}

func (ast *AstUndefType) astType() int {
	return AST_TP_UNDEF_TYPE
}

func (ast *AstUndefType) String() string {
	return fmt.Sprintf("AstUndefType: " + ast.name)
}

func (ast *AstUndefType) signature() string {
	if ast.resolved != nil {
		return ast.resolved.signature()
	} else {
		return "?" + ast.name + ";"
	}
}

func (ast *AstUndefType) desc() string {
	if ast.resolved != nil {
		return ast.resolved.desc()
	} else {
		return "undefine: " + ast.name
	}
}

func realType(tp AstType) AstType {
	if rtype, ok := tp.(*AstUndefType); ok {
		return rtype.resolved
	}

	return tp
}
