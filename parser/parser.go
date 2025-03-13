package parser

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/o0lele/go-retriver/structed"
)

func Parse(dir string) []*structed.Package {

	var structedPkgs []*structed.Package
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
				tmp := structed.NewPackage(pkg.Name)
				for path, f := range pkg.Files {
					tmp.ParseOnlyStruct(path, f, fs)
				}
				for path, f := range pkg.Files {
					tmp.ParseOnlyFunction(path, f, fs)
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
