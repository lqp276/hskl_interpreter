package hskl

const (
	Builtin_print  = "print"
	Builtin_printn = "printn"
	Builtin_str    = "str"
	Builtin_int    = "_intVal"
	Builtin_append = "append"
	Builtin_len    = "len"
)

func builtFuncMap(name string) string {
	if name == "int" {
		return Builtin_int
	}

	return name
}

func builtPrint() *AstFuncDecl {
	fc := &AstFuncDecl{}
	fc.builtin = true
	fc.name = Builtin_print
	fc.retType = newPrimType(symTypeVoid)

	args := "args"
	fc.va_param = &args

	fmtParam := &AstVarDecl{}
	fmtParam.name = "format"
	fmtParam.type_ = newPrimType(symTypeString)
	fc.params = []*AstVarDecl{fmtParam}

	return fc
}

func builtPrintn() *AstFuncDecl {
	fc := &AstFuncDecl{}
	fc.builtin = true
	fc.name = Builtin_printn
	fc.retType = newPrimType(symTypeVoid)

	args := "args"
	fc.va_param = &args

	fmtParam := &AstVarDecl{}
	fmtParam.name = "format"
	fmtParam.type_ = newPrimType(symTypeString)
	fc.params = []*AstVarDecl{fmtParam}

	return fc
}

func builtinStr() *AstFuncDecl {
	fc := &AstFuncDecl{}
	fc.builtin = true
	fc.name = Builtin_str
	fc.retType = newPrimType(symTypeString)

	fmtParam := &AstVarDecl{}
	fmtParam.name = "val"
	fmtParam.type_ = newPrimType(symTypeAny)
	fc.params = []*AstVarDecl{fmtParam}
	return fc
}

func builtinInt() *AstFuncDecl {
	fc := &AstFuncDecl{}
	fc.builtin = true
	fc.name = Builtin_int
	fc.retType = newPrimType(symTypeInt)

	fmtParam := &AstVarDecl{}
	fmtParam.name = "val"
	fmtParam.type_ = newPrimType(symTypeAny)
	fc.params = []*AstVarDecl{fmtParam}
	return fc
}

func builtinAppend() *AstFuncDecl {
	fc := &AstFuncDecl{}
	fc.builtin = true
	fc.name = Builtin_append
	fc.retType = &AstArrayType{elemType: newPrimType(symTypeAny)}

	fmtParam := &AstVarDecl{}
	fmtParam.name = "arr"
	fmtParam.type_ = newPrimType(symTypeAny)
	fc.params = []*AstVarDecl{fmtParam}

	elemParm := &AstVarDecl{name: "elem", type_: newPrimType(symTypeAny)}
	fc.params = append(fc.params, elemParm)

	fc.fixRetType = func(fn *AstFuncCall) AstNode {
		return fn.args[0]
	}

	return fc
}

func builtinLen() *AstFuncDecl {
	fc := &AstFuncDecl{}
	fc.builtin = true
	fc.name = Builtin_len
	fc.retType = newPrimType(symTypeInt)

	fmtParam := &AstVarDecl{}
	fmtParam.name = "arr"
	fmtParam.type_ = &AstArrayType{elemType: newPrimType(symTypeAny)}
	fc.params = []*AstVarDecl{fmtParam}
	return fc
}

func getBuiltinFunc() []*AstFuncDecl {
	fl := []*AstFuncDecl{}
	fl = append(fl, builtPrint(), builtPrintn(), builtinStr(), builtinInt())
	fl = append(fl, builtinAppend(), builtinLen())
	return fl
}
