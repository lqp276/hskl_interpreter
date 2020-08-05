package hskl

import (
	"fmt"

	"github.com/pkg/errors"
)

type outStream interface {
	addLine(format string, args ...interface{})
}

type printStream struct {
	str string
}

func (ps *printStream) addLine(format string, args ...interface{}) {
	desc := fmt.Sprintf(format+"\n", args...)
	fmt.Printf(desc)
}

type astDotifier struct {
	count   int
	ps      outStream
	lastSeq int
}

func (se *astDotifier) newNodeSeq() int {
	se.count++
	return se.count
}

func (se *astDotifier) visitProgram(program *AstProgram) {
	mySeq := se.newNodeSeq()
	se.ps.addLine("n%d [label=\"program\"]", mySeq)
	se.count++
	for _, decl := range program.decl_list {
		switch node := decl.(type) {
		case *AstVarDecl:
			se.visitVarDecl(node)
			se.ps.addLine("n%d -> n%d", mySeq, node.seq)
			break

		case *AstFuncDecl:
			se.visitFuncDecl(node)
			se.ps.addLine("n%d -> n%d", mySeq, node.seq)
			break

		default:
			doPanic("unsupported ast type in program: %T", node)
		}
	}
}

func (se *astDotifier) visitVarDecl(node *AstVarDecl) {
	node.seq = se.newNodeSeq()
	se.ps.addLine("n%d [label=\"VarDecl: %s %s\"]", node.seq, node.name, node.type_)
	se.lastSeq = node.seq
}

func (se *astDotifier) visitFuncDecl(node *AstFuncDecl) {
	node.seq = se.newNodeSeq()
	se.ps.addLine("n%d [label=\"FuncDecl: %s\"]", node.seq, node.name)

	for _, varDecl := range node.params {
		se.visitVarDecl(varDecl)
		se.ps.addLine("n%d -> n%d", node.seq, se.lastSeq)
	}

	se.visitCodeBlock(node.block)
	se.ps.addLine("n%d -> n%d", node.seq, se.lastSeq)

	se.lastSeq = node.seq
}

func (se *astDotifier) visitCodeBlock(node *AstCodeBlock) interface{} {
	node.seq = se.newNodeSeq()
	se.ps.addLine("n%d [label=\"CodeBlock\"]", node.seq)

	for _, varDecl := range node.vars {
		se.visitVarDecl(varDecl)
		se.ps.addLine("n%d -> n%d", node.seq, varDecl.seq)
	}

	for _, ast := range node.stat_list {
		switch ast.(type) {
		case *AstAssgin, *AstBinOP, *AstUnaryOP, *AstIntConst, *AstVarNameRef, *AstFuncCall, *AstReturn:
			se.visitAst(ast)
			se.ps.addLine("n%d -> n%d", node.seq, se.lastSeq)
			break

		case *AstConditionBlock, *AstWhileBlock:
			se.visitAst(ast)
			se.ps.addLine("n%d -> n%d", node.seq, se.lastSeq)
			break

		case *AstNoopStat:
			break

		default:
			doPanic("error ast in func block: %T", ast)
		}
	}

	se.lastSeq = node.seq
	return nil
}

func (se *astDotifier) visitConditionBlock(node *AstConditionBlock) interface{} {
	node.seq = se.newNodeSeq()
	if node.first {
		se.ps.addLine("n%d [label=\"if\"]", node.seq)
	} else {
		se.ps.addLine("n%d [label=\"elif\"]", node.seq)
	}

	se.visitAst(node.cond)
	se.ps.addLine("n%d -> n%d", node.seq, se.lastSeq)

	se.visitAst(node.block)
	se.ps.addLine("n%d -> n%d", node.seq, se.lastSeq)

	if node.altCondBlock != nil {
		se.visitAst(node.altCondBlock)
		se.ps.addLine("n%d -> n%d", node.seq, se.lastSeq)
	}

	if node.altBlock != nil {
		se.visitAst(node.altBlock)
		se.ps.addLine("n%d -> n%d", node.seq, se.lastSeq)
	}

	se.lastSeq = node.seq
	return nil
}

func (se *astDotifier) visitWhileBlock(node *AstWhileBlock) interface{} {
	node.seq = se.newNodeSeq()
	se.ps.addLine("n%d [label=\"while\"]", node.seq)

	se.visitAst(node.cond)
	se.ps.addLine("n%d -> n%d", node.seq, se.lastSeq)

	se.visitAst(node.block)
	se.ps.addLine("n%d -> n%d", node.seq, se.lastSeq)

	se.lastSeq = node.seq
	return nil
}

func (se *astDotifier) visitFuncCall(node *AstFuncCall) interface{} {
	node.seq = se.newNodeSeq()
	se.ps.addLine("n%d [label=\"FuncCall: %s\"]", node.seq, node.name)

	for _, ast := range node.args {
		se.visitAst(ast)
		se.ps.addLine("n%d -> n%d", node.seq, se.lastSeq)
	}

	se.lastSeq = node.seq
	return nil
}

func (se *astDotifier) visitReturn(node *AstReturn) interface{} {
	node.seq = se.newNodeSeq()
	se.ps.addLine("n%d [label=\"return\"]", node.seq)

	se.visitAst(node.expr)
	se.ps.addLine("n%d -> n%d", node.seq, se.lastSeq)
	se.lastSeq = node.seq
	return nil
}

func (se *astDotifier) visitAssign(node *AstAssgin) interface{} {
	node.seq = se.newNodeSeq()
	se.ps.addLine("n%d [label=\"Assgin\"]", node.seq)

	se.visitAst(node.dst)
	se.ps.addLine("n%d -> n%d", node.seq, se.lastSeq)

	switch node.expr.(type) {
	case *AstBinOP, *AstUnaryOP, *AstIntConst, *AstVarNameRef, *AstFuncCall:
		se.visitAst(node.expr)
		se.ps.addLine("n%d -> n%d", node.seq, se.lastSeq)
		break

	default:
		doPanic("error ast in func block: %T", node.expr)
	}

	se.lastSeq = node.seq
	return node.seq
}

func (se *astDotifier) visitBinOP(node *AstBinOP) interface{} {
	node.seq = se.newNodeSeq()
	se.ps.addLine("n%d [label=\"BinOp: %s\"]", node.seq, node.op)

	switch node.left.(type) {
	case *AstBinOP, *AstUnaryOP, *AstIntConst, *AstVarNameRef, *AstFuncCall:
		se.visitAst(node.left)
		se.ps.addLine("n%d -> n%d", node.seq, se.lastSeq)
		break

	default:
		doPanic("error in binop left, unknown ast type: %s, line: %d", node.left, node.line)
	}

	switch node.right.(type) {
	case *AstBinOP, *AstUnaryOP, *AstIntConst, *AstVarNameRef, *AstFuncCall:
		se.visitAst(node.right)
		se.ps.addLine("n%d -> n%d", node.seq, se.lastSeq)
		break

	default:
		doPanic("error in binop right, unknown ast type: %s, line: %d", node.right, node.line)
	}

	se.lastSeq = node.seq
	return node.seq
}

func (se *astDotifier) visitUnaryOP(node *AstUnaryOP) interface{} {
	node.seq = se.newNodeSeq()
	se.ps.addLine("n%d [label=\"UnaryOp: %s\"]", node.seq, node.op)

	switch node.dst.(type) {
	case *AstBinOP, *AstUnaryOP, *AstIntConst, *AstVarNameRef:
		seq := se.visitAst(node.dst).(string)
		se.ps.addLine("n%d -> n%d", node.seq, seq)
		break

	default:
		doPanic("error in unaryop dst, unknown ast type: %s, line: %d", node.dst, node.line)
	}

	se.lastSeq = node.seq
	return node.seq
}

func (se *astDotifier) visitIntConst(node *AstIntConst) interface{} {
	node.seq = se.newNodeSeq()
	se.ps.addLine("n%d [label=\"IntConst: %d\"]", node.seq, node.value)
	se.lastSeq = node.seq
	return node.seq
}

func (se *astDotifier) visitVarRef(node *AstVarNameRef) interface{} {
	node.seq = se.newNodeSeq()
	se.ps.addLine("n%d [label=\"VarRef: %s\"]", node.seq, node.name)
	se.lastSeq = node.seq
	return node.seq
}

func (se *astDotifier) gendot(root AstNode) (result error) {
	header := `digraph astgraph {
	  node [shape=circle, fontsize=12, fontname="Courier", height=.1];
	  ranksep=.3;
	  edge [arrowsize=.5]
	`

	se.ps.addLine(header)

	defer func() {
		if r := recover(); r != nil {
			result = r.(error)
		}
	}()

	switch node := root.(type) {
	case *AstProgram:
		se.visitProgram(node)
		break

	default:
		return errors.Errorf("root ast type should be program, actual recv: %T", root)
	}

	se.ps.addLine("}")
	return nil
}

func (se *astDotifier) visitAst(ast AstNode) interface{} {
	//fmt.Printf("visit ast: %T\n", ast)
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

	case *AstIntConst:
		return se.visitIntConst(statement)

	case *AstVarNameRef:
		return se.visitVarRef(statement)

	case *AstFuncCall:
		return se.visitFuncCall(statement)

	case *AstReturn:
		return se.visitReturn(statement)

	case *AstCodeBlock:
		return se.visitCodeBlock(statement)

	case *AstConditionBlock:
		return se.visitConditionBlock(statement)

	case *AstWhileBlock:
		return se.visitWhileBlock(statement)

	default:
		doPanic("dotifier: unknown ast type: %T", ast)
	}

	return nil
}

func newDotifier() *astDotifier {
	se := &astDotifier{}
	se.ps = &printStream{}
	return se
}
