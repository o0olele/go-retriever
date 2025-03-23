package goretriever

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func ParseString(name, content string) (*Package, error) {
	fSet := token.NewFileSet()

	f, err := parser.ParseFile(fSet, name, content, 0)
	if err != nil {
		return nil, err
	}

	pkg := NewPackage(name)

	err = pkg.FromString(content, f, fSet)
	if err != nil {
		return nil, err
	}

	return pkg, nil
}

func Parse(dir string) []*Package {

	var structedPkgs []*Package
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if info.Name() == "." || info.Name() == ".." {
				return nil
			}

			fs := token.NewFileSet()
			pkgs, err := parser.ParseDir(fs, path, func(fi os.FileInfo) bool {
				return strings.HasSuffix(fi.Name(), ".go")
			}, parser.ParseComments)
			if err != nil {
				panic(err)
			}

			for _, pkg := range pkgs {

				tmp := NewPackage(pkg.Name)
				for path, f := range pkg.Files {
					file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
					if err != nil {
						return err
					}

					tmp.ParseStruct(file, f, fs)

					file.Close()
				}
				for path, f := range pkg.Files {
					file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
					if err != nil {
						return err
					}

					tmp.ParseFunction(file, f, fs)

					file.Close()
				}

				structedPkgs = append(structedPkgs, tmp)
			}
		}
		return nil
	})

	if err != nil {
		panic(err)
	}

	return structedPkgs
}
