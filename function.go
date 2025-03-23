package goretriever

import (
	"go/ast"
	"go/token"
	"io"
)

type Function struct {
	Name   string
	Code   string
	Struct *Struct `json:"-"`
}

func newFunctionFromDecl(reader io.ReaderAt, decl *ast.FuncDecl, fileSet *token.FileSet) (string, *Function) {
	if reader == nil || decl == nil {
		return "", nil
	}

	f := &Function{
		Name: decl.Name.Name,
	}

	var structName string
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		recv := decl.Recv.List[0]
		switch t := recv.Type.(type) {
		case *ast.StarExpr:
			if ident, ok := t.X.(*ast.Ident); ok {
				structName = ident.Name
			}
		case *ast.Ident:
			structName = t.Name
		}
	}

	beg, end, err := GetFuncDeclOffset(decl, fileSet)
	if err != nil {
		panic(err)
	}

	code, err := ParseCode(reader, int64(beg), int64(end))
	if err != nil {
		return "", nil
	}

	f.Code = code

	return structName, f
}
