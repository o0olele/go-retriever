package goretriever

import (
	"bytes"
	"go/ast"
	"go/token"
	"io"
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
		s = &Struct{
			Name: structName,
		}
		p.Structs[structName] = s
	}
	s.AddMethod(f)
}

func (p *Package) ParseStruct(reader io.ReaderAt, f *ast.File, fileSet *token.FileSet) error {

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		s := newStructFromDecl(reader, genDecl, fileSet)
		if s != nil {
			p.AddStruct(s)
		}
	}
	return nil
}

func (p *Package) ParseFunction(reader io.ReaderAt, f *ast.File, fileSet *token.FileSet) error {

	for _, decl := range f.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		structName, f := newFunctionFromDecl(reader, funcDecl, fileSet)
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

func (p *Package) FromString(content string, f *ast.File, fileSet *token.FileSet) error {
	reader := bytes.NewReader([]byte(content))

	err := p.ParseStruct(reader, f, fileSet)
	if err != nil {
		return err
	}

	err = p.ParseFunction(reader, f, fileSet)
	if err != nil {
		return err
	}

	return nil
}

func (p *Package) FromFile(path string, f *ast.File, fileSet *token.FileSet) error {

	reader, err := os.OpenFile(path, os.O_RDONLY, fs.ModePerm)
	if err != nil {
		return err
	}
	defer reader.Close()

	err = p.ParseStruct(reader, f, fileSet)
	if err != nil {
		return err
	}

	err = p.ParseFunction(reader, f, fileSet)
	if err != nil {
		return err
	}

	return nil
}
