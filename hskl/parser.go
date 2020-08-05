package hskl

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
)

/*
program : declarations

declarations : (variable_declaration | func_decl | type_def)*

type_def: TYPE ID type_ref
type_ref:  (LBRACKET RBRACKET)* (INT | STRING | struct_def)
struct_def : STRUCT LBRACE fields_def type_spec RBRACE
fields_def : (ID (COMMA ID)* COLON type_spec)*

variable_declaration : var_type_decl | var_assign_decl | empty
var_type_decl : VAR ID (COMMA ID)* COLON type_spec
var_assign_decl : ID ":="
		INT_CONST |
		STRING_CONST |
		LBRACKET RBRACKET INT LBRACKET (INT_CONST (COMMA, INT_CONST)*) RBRACKET |
		LBRACKET RBRACKET STRING LBRACKET (STRING_CONST (COMMA, STRING_CONST)*) RBRACKET

type_spec : INT | STRING |  ID | LBRACKET RBRACKET type_spec

func_decl: FUNC ID LPAREN formal_params RPAREN type_spec? code_block
code_block : LBRACE variable_declaration statement_list RBRACE

formal_params : ID (COMMA ID)* COLON type_spec (COMMA ID (COMMA ID)* COLON type_spec)*

statement_list : statement*
statement : (misc_stat |
			 func_call | condition_stat |
			 while_stat | break_stat) ";" | Empty

misc_stat: 	assign_statement | expr

break_stat : BREAK
condition_stat : IF expr
		code_block
		(ELIF expr code_block)*
		ELSE code_block

while_stat: while expr code_block

func_call : ID LPAREN (call_args) RPAREN
call_args : (expr | func_call) (COMMA (expr | func_call))* | Empty

assign_statement: var_ref ASSIGN expr
var_ref : ID (LBRACKET expr  RBRACKET | DOT ID)*
expr   : expr_comp ((AND | OR) comp)*
expr_comp   : expr ((GT | GTE | LT | LTE | EQ) expr)
expr_add   : term ((PLUS | MINUS) term)*
expr_mul   : factor ((MUL | DIV) factor)*
factor : (PLUS|MINUS|NOT) factor
		 | INTEGER
		 | STRING
		 | var_ref
		 | func_call
		 | new_op
		 | LPAREN expr RPAREN

new_op : NEW LPAREN type_spec RPAREN
*/

const (
	//symbol type
	symTypeVoid   = "void"
	symTypeInt    = "int"
	symTypeFloat  = "float"
	symTypeString = "string"
	symTypeArray  = "array"
	symTypeAny    = "any"
	symTypeStruct = "struct"

	entryFunc = "main"
)

type hskParser struct {
	lex       *hskLexer
	prevToken *Token
	curToken  *Token

	lookAhead []*Token
	lookPrevs []*Token
	pos_ah    int
	markers   []int
	lastError error
	tpMap     map[string]AstType
}

func (p *hskParser) getLastError() error {
	return p.lastError
}

func (p *hskParser) is_speculating() bool {
	return len(p.markers) > 0
}

func (p *hskParser) peekToken() *Token {
	next := p.pos_ah + 1
	if next < len(p.lookAhead) {
		return p.lookAhead[next]
	}

	//fill look ahead buf
	token := p.lex.getNextToken()
	p.lookAhead = append(p.lookAhead, token)
	return token
}

func (p *hskParser) mark_push() {
	//fmt.Printf("push prev: %s:%d, cur: %s:%d\n", p.prevToken.value, p.prevToken.line, p.curToken.value, p.curToken.line)
	p.lookPrevs = append(p.lookPrevs, p.prevToken)
	p.markers = append(p.markers, p.pos_ah)
}

func (p *hskParser) mark_pop() {
	mark := p.markers[len(p.markers)-1]
	p.markers = p.markers[0 : len(p.markers)-1]

	p.curToken = p.lookAhead[mark]
	p.pos_ah = mark

	p.prevToken = p.lookPrevs[len(p.lookPrevs)-1]
	p.lookPrevs = p.lookPrevs[0 : len(p.lookPrevs)-1]
	p.lastError = nil

	//fmt.Printf("recover prev: %s:%d, cur: %s:%d\n", p.prevToken.value, p.prevToken.line, p.curToken.value, p.curToken.line)
}

func (p *hskParser) eat(ttype string) {
	if p.curToken.type_ == ttype {
		p.pos_ah++
		p.prevToken = p.curToken

		if p.pos_ah < len(p.lookAhead) {
			p.curToken = p.lookAhead[p.pos_ah]
		} else {
			p.curToken = p.lex.getNextToken()
			if p.is_speculating() {
				p.lookAhead = append(p.lookAhead, p.curToken)
			} else {
				p.pos_ah = 0
				p.lookAhead = p.lookAhead[0:1]
				p.lookAhead[0] = p.curToken
			}
		}

		if p.curToken == nil {
			p.panic(fmt.Sprintf("parse failed, last token: '%s', lexer error: %s", ttype, p.lex.getLastError()))
		}

	} else {
		p.panic(fmt.Sprintf("parse error, expect '%s', find: '%s:%s', line: %d", ttype, p.curToken.type_, p.curToken.value, p.curToken.line))
	}
}

func (p *hskParser) panic(format string, args ...interface{}) {
	desc := fmt.Sprintf(format, args...)
	p.lastError = errors.New(desc)
	panic(desc)
}

func (p *hskParser) eatSeperator() {
	if p.curToken.type_ == SEMI {
		p.eat(SEMI)
	} else {
		//check next token is in new line
		if p.prevToken != nil && p.prevToken.line == p.curToken.line {
			p.panic(fmt.Sprintf("missing seperator after '%s' line: %d", p.curToken.value, p.curToken.line))
		}
	}
}

//same as program
func (p *hskParser) Program() AstNode {
	return p.declarations()
}

//same as program
func (p *hskParser) declarations() AstNode {
	program := &AstProgram{}
	program.decl_list = []AstNode{}
	program.tpMap = p.tpMap
	//declarations : (variable_declaration | func_decl)*

	for p.curToken.type_ != EOF {
		if p.curToken.type_ == FUNC {
			p.eat(FUNC)
			ast := p.func_decl()
			if ast == nil {
				p.lastError = errors.Errorf("parse func_decl error: %s", p.lex.getLastError())
				break
			} else {
				program.decl_list = append(program.decl_list, ast)
			}
		} else if p.curToken.type_ == TYPE {
			ast := p.type_def()
			program.decl_list = append(program.decl_list, ast)
		} else {
			ast := p.variable_decl()
			p.eatSeperator()
			if ast == nil {
				p.lastError = errors.Errorf("parse variable_declaration error: %s", p.lex.getLastError())
				break
			} else {
				for _, val := range ast {
					program.decl_list = append(program.decl_list, val)
				}
			}
		}
	}

	if p.lastError != nil {
		return nil
	}
	return program
}

func (p *hskParser) type_def() *AstTypeDef {
	/*
		type_def: TYPE ID type_ref
		type_ref:  (LBRACKET RBRACKET)* (INT | ID | STRING | struct_def)
		struct_def : STRUCT LBRACE fields_def type_spec RBRACE
		fields_def : (ID (COMMA ID)* COLON type_spec)*
	*/

	p.eat(TYPE)
	p.eat(ID)

	name := p.prevToken.value
	ast := &AstTypeDef{}
	ast.name = name
	ast.impl = p.type_seek()

	if strct, ok := ast.impl.(*AstStructType); ok {
		strct.name = name
	}

	//fmt.Printf("type def name: %s, type: %s\n", ast.name, ast.impl.signature())
	if old, ok := p.tpMap[name]; ok {
		if ast.impl.astType() != AST_TP_UNDEF_TYPE && old.astType() == AST_TP_UNDEF_TYPE {
			//resolve it
			oldUndef := old.(*AstUndefType)
			if oldUndef.name == ast.name && oldUndef.resolved == nil {
				oldUndef.resolved = ast.impl
				return ast
			}
		}

		p.panic("duplicate type define, name: %s, type: %s, old type: %s", name, ast.impl.signature(), old.signature())
		return nil
	} else {
		p.tpMap[name] = ast.impl
	}

	return ast
}

func (p *hskParser) type_seek() AstType {
	//type_ref:  (LBRACKET RBRACKET)* (INT | ID | STRING | struct_def)

	if p.curToken.type_ == LBRACKET {
		p.eat(LBRACKET)
		p.eat(RBRACKET)
		ast := &AstArrayType{}
		ast.elemType = p.type_seek()
		return ast
	} else if p.curToken.type_ == TYPE_INT {
		p.eat(TYPE_INT)
		return p.tpMap[symTypeInt]
	} else if p.curToken.type_ == TYPE_STRING {
		p.eat(TYPE_STRING)
		return p.tpMap[symTypeString]
	} else if p.curToken.type_ == STRUCT {
		ast := p.struct_def()
		return ast
	} else {
		p.eat(ID)
		id := p.prevToken.value
		return &AstUndefType{name: id}
	}
}

func (p *hskParser) struct_def() *AstStructType {
	p.eat(STRUCT)
	p.eat(LBRACE)

	ast := &AstStructType{}
	for p.curToken.type_ != RBRACE {
		ast.fields = append(ast.fields, p.field_decl()...)
	}

	fmap := make(map[string]*AstVarDecl)

	for _, field := range ast.fields {
		if of, ok := fmap[field.name]; ok {
			doPanic("duplicate filed name: %s with type: %s, line: %d, prev type: %s, line: %d",
				field.name, field.type_.signature(), field.line, of.type_.signature(), of.line)
			break
		}
	}

	p.eat(RBRACE)
	return ast
}

func (p *hskParser) field_decl() []*AstVarDecl {
	decVars := []*AstVarDecl{}

	names := []string{}
	names = append(names, p.curToken.value)

	line := p.curToken.line
	p.eat(ID)

	for p.curToken.type_ == COMMA {
		p.eat(COMMA)
		if p.curToken.type_ == ID {
			names = append(names, p.curToken.value)
			p.eat(ID)
		} else {
			break
		}
	}

	p.eat(COLON)
	varTp := p.type_spec()

	for _, name := range names {
		decVars = append(decVars, &AstVarDecl{name: name, type_: varTp, line: line})
	}

	return decVars
}

func (p *hskParser) variable_decl() []*AstVarDecl {
	decls := []*AstVarDecl{}
	//variable_declaration : var_type_decl | var_assign_decl | empty
	if p.curToken.type_ == EOF {
		return decls
	}

	if p.curToken.type_ == VAR {
		decls = append(decls, p.var_type_decl()...)
	} else {
		decls = append(decls, p.var_assign_decl())
	}

	return decls
}

func (p *hskParser) type_spec() AstType {
	//type_spec : INT | STRING |  ID | LBRACKET RBRACKET type_spec

	if p.curToken.type_ == TYPE_INT {
		p.eat(TYPE_INT)
		return p.tpMap[symTypeInt]
	} else if p.curToken.type_ == TYPE_STRING {
		p.eat(TYPE_STRING)
		return p.tpMap[symTypeString]
	} else if p.curToken.type_ == TYPE_ANY {
		p.eat(TYPE_ANY)
		return p.tpMap[symTypeAny]
	} else if p.curToken.type_ == ID {
		p.eat(ID)
		tp := p.tpMap[p.prevToken.value]
		if tp != nil {
			return tp
		}
		ast := &AstUndefType{}
		ast.name = p.prevToken.value

		p.tpMap[ast.name] = ast
		return ast
	} else if p.curToken.type_ == LBRACKET {
		p.eat(LBRACKET)
		p.eat(RBRACKET)
		ast := &AstArrayType{}
		ast.elemType = p.type_spec()
		return ast
	} else {
		p.panic("error type spec: %s", p.curToken.value)
		return nil
	}
}

func (p *hskParser) var_type_decl() []*AstVarDecl {
	decVars := []*AstVarDecl{}
	if p.curToken.type_ != VAR {
		return decVars
	}
	p.eat(VAR)

	names := []string{}
	names = append(names, p.curToken.value)

	line := p.curToken.line
	p.eat(ID)

	for p.curToken.type_ == COMMA {
		p.eat(COMMA)
		if p.curToken.type_ == ID {
			names = append(names, p.curToken.value)
			p.eat(ID)
		} else {
			break
		}
	}

	p.eat(COLON)
	varTp := p.type_spec()

	for _, name := range names {
		decVars = append(decVars, &AstVarDecl{name: name, type_: varTp, line: line})
	}

	return decVars
}

func (p *hskParser) var_assign_decl() *AstVarDecl {
	/*
		var_assign_decl : ID ":="
			INT_CONST |
			STRING_CONST |
			LBRACKET RBRACKET INT LBRACKET (INT_CONST (COMMA, INT_CONST)*) RBRACKET |
			LBRACKET RBRACKET STRING LBRACKET (STRING_CONST (COMMA, STRING_CONST)*) RBRACKET
	*/
	id := p.curToken
	line := p.curToken.line
	p.eat(ID)
	p.eat(DEC_ASSIGN)

	initVal := p.curToken

	if p.curToken.type_ == INT_CONST {
		p.eat(INT_CONST)
		astNode := &AstVarDecl{name: id.value, initVal: initVal.value, type_: &AstPrimType{name: symTypeInt}, line: line}
		return astNode
	} else if p.curToken.type_ == STRING_CONST {
		p.eat(STRING_CONST)
		astNode := &AstVarDecl{name: id.value, initVal: initVal.value, type_: &AstPrimType{name: symTypeString}, line: line}
		return astNode
	} else {
		//array
		p.eat(LBRACKET)
		p.eat(RBRACKET)

		astNode := &AstVarDecl{name: id.value, line: line}
		if p.curToken.type_ == TYPE_INT {
			astNode.type_ = &AstArrayType{elemType: newPrimType(symTypeInt)}
			p.eat(TYPE_INT)
			p.eat(LBRACE)

			if p.curToken.type_ == INT_CONST {
				astNode.initArr = append(astNode.initArr, p.curToken)
				p.eat(INT_CONST)
			}

			for p.curToken.type_ == COMMA {
				p.eat(COMMA)
				astNode.initArr = append(astNode.initArr, p.curToken)
				p.eat(INT_CONST)
			}
		} else {
			//string
			astNode.type_ = &AstArrayType{elemType: newPrimType(symTypeString)}
			p.eat(TYPE_STRING)
			p.eat(LBRACE)

			if p.curToken.type_ == STRING_CONST {
				astNode.initArr = append(astNode.initArr, p.curToken)
				p.eat(STRING_CONST)
			}

			for p.curToken.type_ == COMMA {
				p.eat(COMMA)
				astNode.initArr = append(astNode.initArr, p.curToken)
				p.eat(STRING_CONST)
			}
		}
		p.eat(RBRACE)
		return astNode
	}
}

func (p *hskParser) func_decl() *AstFuncDecl {
	/*
		func_decl: "func" ID "(" formal_params ")" type_spec? "{"
			variable_declaration
			statement_list
		"}"
	*/

	ast := &AstFuncDecl{name: p.curToken.value}
	ast.line = p.curToken.line
	p.eat(ID)
	p.eat(LPAREN)
	ast.params = p.formal_params()
	p.eat(RPAREN)
	if p.curToken.type_ != LBRACE {
		ast.retType = p.type_spec()
	} else {
		ast.retType = newPrimType(symTypeVoid)
	}
	ast.block = p.code_block()
	return ast
}

func (p *hskParser) code_block() *AstCodeBlock {
	p.eat(LBRACE)
	ast := &AstCodeBlock{}
	ast.vars = []*AstVarDecl{}
	for p.curToken.type_ == VAR ||
		(p.curToken.type_ == ID && p.peekToken().type_ == DEC_ASSIGN) {
		ast.vars = append(ast.vars, p.variable_decl()...)
	}

	ast.stat_list = p.statement_list()
	p.eat(RBRACE)
	return ast
}

func (p *hskParser) formal_params() []*AstVarDecl {
	//ID (COMMA ID)* COLON type_spec (COMMA ID (COMMA ID)* COLON type_spec)*
	decVars := []*AstVarDecl{}
	if p.curToken.type_ != ID {
		return decVars
	}

	names := []string{}
	names = append(names, p.curToken.value)

	line := p.curToken.line
	p.eat(ID)

	for p.curToken.type_ == COMMA {
		p.eat(COMMA)
		if p.curToken.type_ == ID {
			names = append(names, p.curToken.value)
			p.eat(ID)
		} else {
			break
		}
	}

	p.eat(COLON)
	tpVal := p.type_spec()

	for _, name := range names {
		decVars = append(decVars, &AstVarDecl{name: name, type_: tpVal, line: line})
	}

	for p.curToken.type_ == COMMA {
		p.eat(COMMA)
		decVars = append(decVars, p.formal_params()...)
	}

	return decVars
}

func (p *hskParser) statement_list() []AstNode {
	list := []AstNode{}
	if p.curToken.type_ == RBRACE {
		return list
	}

	list = append(list, p.statement())

	for p.curToken.type_ != RBRACE && p.curToken.type_ != EOF {
		stat := p.statement()
		if stat.astType() != AST_Noop {
			list = append(list, stat)
		}
		p.eatSeperator()
	}

	return list
}

func (p *hskParser) statement() AstNode {
	var ast AstNode
	ast = &AstNoopStat{}
	if p.curToken.type_ == RETURN {
		ast = p.return_stat()
	} else if p.curToken.type_ == IF {
		ast = p.condition_stat()
	} else if p.curToken.type_ == WHILE {
		ast = p.while_stat()
	} else if p.curToken.type_ == BREAK {
		ast = p.break_stat()
	} else {
		ast = p.misc_stat()
	}

	return ast
}

func (p *hskParser) condition_stat() AstNode {
	/*
		condition_stat : IF expr
				code_block
				(ELIF expr code_block)*
				ELSE code_block
	*/

	p.eat(IF)
	topAst := &AstConditionBlock{}
	topAst.cond = p.expr()
	topAst.first = true
	topAst.block = p.code_block()

	curAst := topAst
	for p.curToken.type_ == ELIF {
		p.eat(ELIF)
		ast := &AstConditionBlock{}
		ast.cond = p.expr()
		ast.block = p.code_block()

		curAst.altCondBlock = ast
		curAst = ast
	}

	if p.curToken.type_ == ELSE {
		p.eat(ELSE)
		curAst.altBlock = p.code_block()
	}

	return topAst
}

func (p *hskParser) while_stat() AstNode {
	//while_stat: while expr code_block
	ast := &AstWhileBlock{}
	p.eat(WHILE)
	ast.cond = p.expr()
	ast.block = p.code_block()
	return ast
}

func (p *hskParser) break_stat() AstNode {
	//while_stat: while expr code_block
	p.eat(BREAK)
	ast := &AstBreak{}
	return ast
}

func (p *hskParser) return_stat() AstNode {
	p.eat(RETURN)

	ast := &AstReturn{}

	switch p.curToken.type_ {
	case ID, INT_CONST, STRING_CONST:
		ast.expr = p.expr()
		break

	default:
		ast.expr = nil
	}

	return ast
}

func (p *hskParser) func_call() AstNode {
	/*
		func_call : ID LPAREN (call_args) RPAREN
		call_args : (expr | func_call) (COMMA (expr | func_call))* | Empty
	*/

	p.eat(p.curToken.type_)
	ast := &AstFuncCall{}
	ast.name = builtFuncMap(p.prevToken.value)
	ast.line = p.prevToken.line

	p.eat(LPAREN)
	ast.args = p.call_args()
	p.eat(RPAREN)
	return ast
}

func (p *hskParser) new_op() *AstNewOP {
	//new_op : NEW LPAREN type_spec RPAREN
	ast := &AstNewOP{}
	ast.line = p.curToken.line

	p.eat(NEW)
	p.eat(LPAREN)
	ast.opType = p.type_spec()
	p.eat(RPAREN)

	return ast
}

func (p *hskParser) call_args() []AstNode {
	/*
		func_call : ID LPAREN (call_args) RPAREN
		call_args : call_arg (COMMA call_arg)* | Empty
	*/

	args := []AstNode{}
	if p.curToken.type_ == RPAREN {
		return args
	}

	args = append(args, p.call_arg())
	for p.curToken.type_ == COMMA {
		p.eat(COMMA)
		args = append(args, p.call_arg())
	}

	return args
}

func (p *hskParser) call_arg() AstNode {
	/*
		func_call : ID LPAREN (call_args) RPAREN
		call_arg : expr | func_call
	*/

	if p.curToken.type_ == ID && p.peekToken().type_ == LPAREN {
		//func call
		return p.func_call()
	} else {
		//expr
		return p.expr()
	}
}

func (p *hskParser) spec_assign_stat() (ok bool) {
	ok = true
	p.mark_push()

	defer func() {
		p.mark_pop()

		if r := recover(); r != nil {
			//fmt.Printf("spect assign stat failed: %s\n", r)
			ok = false
		}
	}()

	p.assign_statement()
	return ok
}

func (p *hskParser) spec_expr() (ok bool) {
	ok = true
	p.mark_push()

	defer func() {
		p.mark_pop()

		if r := recover(); r != nil {
			//fmt.Printf("spect expr stat failed: %s\n", r)
			ok = false
		}
	}()

	p.expr()
	return ok
}

func (p *hskParser) misc_stat() AstNode {
	//misc_stat: 	assign_statement | expr
	if p.spec_assign_stat() {
		//fmt.Printf("get assign stat, cur token: %s, line: %d\n", p.curToken.value, p.curToken.line)
		return p.assign_statement()
	} else if p.spec_expr() {
		//fmt.Printf("get expr stat, cur token: %s, line: %d\n", p.curToken.value, p.curToken.line)
		return p.expr()
	}

	doPanic("unexpected token: %s:%d after token: %s:%d", p.curToken.value, p.curToken.line,
		p.prevToken.value, p.prevToken.line)
	return nil
}

func (p *hskParser) assign_statement() AstNode {
	//assign_statement: var_ref ASSIGN expr
	line := p.curToken.line
	lhs := p.var_ref()
	p.eat(ASSIGN)
	expr := p.expr()

	ast := &AstAssgin{dst: lhs, expr: expr, line: line}
	return ast
}

/*
expr   : expr_and (OR expr_and)*
expr_and   : expr_equ (AND expr_equ)*
expr_equ   : expr_comp ((EQU | NEQ) expr_comp)
expr_comp   : expr_add ((GT | GTE | LT | LTE) expr_add)
expr_add   : term ((PLUS | MINUS) term)*
expr_mul   : factor ((MUL | DIV) factor)*
factor : (PLUS | MINUS | NOT) factor | INT_CONST | LPAREN expr RPAREN
*/

func (p *hskParser) expr() AstNode {
	node := p.expr_and()

	for p.curToken.type_ == OR {
		token := p.curToken
		p.eat(p.curToken.type_)
		node = &AstBinOP{op: token.type_, left: node, right: p.expr_and(), line: token.line}
	}

	return node
}

func (p *hskParser) expr_and() AstNode {
	node := p.expr_equ()

	for p.curToken.type_ == AND {
		token := p.curToken
		p.eat(p.curToken.type_)
		node = &AstBinOP{op: token.type_, left: node, right: p.expr_equ(), line: token.line}
	}

	return node
}

func (p *hskParser) expr_equ() AstNode {
	node := p.expr_comp()
	for p.curToken.type_ == EQU || p.curToken.type_ == NEQ {
		token := p.curToken
		p.eat(p.curToken.type_)
		node = &AstBinOP{op: token.type_, left: node, right: p.expr_comp(), line: token.line}
	}
	return node
}

func (p *hskParser) expr_comp() AstNode {
	node := p.expr_add()
	for p.curToken.type_ == GT || p.curToken.type_ == GTE ||
		p.curToken.type_ == LT || p.curToken.type_ == LTE {
		token := p.curToken
		p.eat(p.curToken.type_)
		node = &AstBinOP{op: token.type_, left: node, right: p.expr_add(), line: token.line}
	}
	return node
}

func (p *hskParser) expr_add() AstNode {
	node := p.expr_mul()
	for p.curToken.type_ == PLUS || p.curToken.type_ == MINUS {
		token := p.curToken
		p.eat(p.curToken.type_)
		node = &AstBinOP{op: token.type_, left: node, right: p.expr_mul(), line: token.line}
	}
	return node
}

func (p *hskParser) expr_mul() AstNode {
	node := p.factor()
	for p.curToken.type_ == MUL || p.curToken.type_ == DIV {
		token := p.curToken
		p.eat(p.curToken.type_)
		node = &AstBinOP{op: token.type_, left: node, right: p.factor(), line: token.line}
	}
	return node
}

func (p *hskParser) factor() AstNode {
	/*
		factor : (PLUS|MINUS|NOT) factor
				| INTEGER
				| STRING
				| var_ref
				| func_call
				| new_op
				| LPAREN expr RPAREN

		new_op : NEW LPAREN type_spec RPAREN
	*/
	if p.curToken.type_ == PLUS || p.curToken.type_ == MINUS || p.curToken.type_ == NOT {
		ast := &AstUnaryOP{}
		ast.op = p.curToken.type_
		ast.line = p.curToken.line
		p.eat(p.curToken.type_)

		ast.dst = p.factor()
		return ast
	} else if p.curToken.type_ == INT_CONST {
		p.eat(INT_CONST)
		val, _ := strconv.Atoi(p.prevToken.value)
		ast := &AstIntConst{value: val}
		return ast
	} else if p.curToken.type_ == STRING_CONST {
		p.eat(STRING_CONST)
		ast := &AstStringConst{value: p.prevToken.value}
		return ast
	} else if p.curToken.type_ == LPAREN {
		p.eat(LPAREN)
		ast := p.expr()
		p.eat(RPAREN)
		return ast
	} else if p.curToken.type_ == ID {
		if p.peekToken().type_ == LPAREN {
			ast := p.func_call()
			return ast
		} else {
			ast := p.var_ref()
			return ast
		}
	} else if p.curToken.type_ == TYPE_INT && p.peekToken().type_ == LPAREN {
		ast := p.func_call()
		return ast
	} else if p.curToken.type_ == NEW && p.peekToken().type_ == LPAREN {
		ast := p.new_op()
		return ast
	} else {
		msg := fmt.Sprintf("parse factor failed, cur token: '%s', line: %d", p.curToken.value, p.curToken.line)
		//fmt.Println(msg)
		p.panic(msg)
		return nil
	}
}

func (p *hskParser) var_ref() AstNode {
	//var_ref : ID (LBRACKET expr  RBRACKET | DOT ID)*
	var ast AstNode
	ast = &AstVarNameRef{name: p.curToken.value, line: p.curToken.line}
	p.eat(ID)

loop:
	for {
		switch p.curToken.type_ {
		case LBRACKET:
			top := &AstIndexedRef{}
			top.line = p.curToken.line
			top.host = ast
			p.eat(LBRACKET)
			top.index = p.expr()
			p.eat(RBRACKET)
			ast = top
			break

		case DOT:
			top := &AstDotRef{}
			top.line = p.curToken.line
			top.host = ast
			p.eat(DOT)
			top.name = p.curToken.value
			p.eat(ID)
			ast = top
			break

		default:
			break loop
		}
	}

	return ast
}

func NewParser(text string) *hskParser {
	p := &hskParser{}
	p.lex = newLexer(text)

	p.curToken = p.lex.getNextToken()
	p.lookAhead = append(p.lookAhead, p.curToken)
	for p.curToken.type_ == LF {
		p.eat(LF)
	}

	p.tpMap = make(map[string]AstType)
	arr := []string{symTypeAny, symTypeInt, symTypeString, symTypeVoid}
	for _, val := range arr {
		p.tpMap[val] = &AstPrimType{name: val}
	}

	return p
}
