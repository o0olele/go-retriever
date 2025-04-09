package goretriever

import (
	"go/ast"
	"go/token"
	"io"
)

type Struct struct {
	Name    string
	Code    string
	Methods map[string]*Function
	Beg     int `json:"-"`
	End     int `json:"-"`
}

func newStructFromDecl(reader io.ReaderAt, decl *ast.GenDecl, fileSet *token.FileSet) *Struct {
	if reader == nil || decl == nil {
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

		beg, end, err := getGenDeclOffset(decl, fileSet)
		if err != nil {
			panic(err)
		}

		code, err := parseCode(reader, int64(beg), int64(end))
		if err != nil {
			return nil
		}
		s.Code = code
		s.Beg = beg
		s.End = end
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
