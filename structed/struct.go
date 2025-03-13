package structed

import (
	"go/ast"
	"go/token"
	"os"

	"github.com/o0lele/go-retriver/common"
)

type Struct struct {
	Name    string
	Code    string
	Methods map[string]*Function
}

func newStructFromDecl(file *os.File, decl *ast.GenDecl, fileSet *token.FileSet) *Struct {
	if file == nil || decl == nil {
		return nil
	}

	if decl.Tok != token.TYPE {
		return nil
	}

	if len(decl.Specs) == 0 {
		return nil
	}

	spec := decl.Specs[0]
	if ts, ok := spec.(*ast.TypeSpec); ok {
		s := &Struct{
			Name:    ts.Name.Name,
			Methods: make(map[string]*Function),
		}

		beg, end, err := common.GetGenDeclOffset(decl, fileSet)
		if err != nil {
			panic(err)
		}

		code, err := common.ParseCode(file, int64(beg), int64(end))
		if err != nil {
			return nil
		}
		s.Code = code
		return s
	}

	return nil
}

func (s *Struct) AddMethod(f *Function) {
	f.Struct = s
	if s.Methods == nil {
		s.Methods = make(map[string]*Function)
	}
	s.Methods[f.Name] = f
}
