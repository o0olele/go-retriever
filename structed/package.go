package structed

import (
	"go/ast"
	"go/token"
	"io/fs"
	"os"
)

type Package struct {
	Name      string
	Structs   map[string]*Struct
	Functions map[string]*Function
}

func NewPackage(name string) *Package {
	return &Package{
		Name:      name,
		Structs:   make(map[string]*Struct),
		Functions: make(map[string]*Function),
	}
}

func (p *Package) AddStruct(s *Struct) {
	p.Structs[s.Name] = s
}

func (p *Package) AddFunction(f *Function) {
	p.Functions[f.Name] = f
}

func (p *Package) AddMethod(structName string, f *Function) {
	s := p.Structs[structName]
	if s == nil {
		return
	}
	s.AddMethod(f)
}

func (p *Package) ParseOnlyStruct(path string, f *ast.File, fileSet *token.FileSet) error {
	file, err := os.OpenFile(path, os.O_RDONLY, fs.ModePerm)
	if err != nil {
		return err
	}
	defer file.Close()

	println(path, f.Name.Name)
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		s := newStructFromDecl(file, genDecl, fileSet)
		if s != nil {
			p.AddStruct(s)
		}
	}
	return nil
}

func (p *Package) ParseOnlyFunction(path string, f *ast.File, fileSet *token.FileSet) error {
	file, err := os.OpenFile(path, os.O_RDONLY, fs.ModePerm)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, decl := range f.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		structName, f := newFunctionFromDecl(file, funcDecl, fileSet)
		if f == nil {
			continue
		}

		if structName != "" {
			p.AddMethod(structName, f)
		} else {
			p.AddFunction(f)
		}
	}

	return nil
}

func (p *Package) ParseFile(path string, f *ast.File, fileSet *token.FileSet) error {

	file, err := os.OpenFile(path, os.O_RDONLY, fs.ModePerm)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		s := newStructFromDecl(file, genDecl, fileSet)
		if s != nil {
			p.AddStruct(s)
		}
	}

	for _, decl := range f.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		structName, f := newFunctionFromDecl(file, funcDecl, fileSet)
		if f == nil {
			continue
		}

		if structName != "" {
			p.AddMethod(structName, f)
		} else {
			p.AddFunction(f)
		}
	}

	return nil
}
