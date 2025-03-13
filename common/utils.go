package common

import (
	"errors"
	"go/ast"
	"go/token"
	"os"
)

func Max[T token.Pos](a, b T) T {
	if a > b {
		return a
	}
	return b
}

func Min[T token.Pos](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func ParseCode(file *os.File, beg, end int64) (string, error) {
	if file == nil {
		return "", errors.New("invalid input")
	}

	buffer := make([]byte, end-beg+1)
	if _, err := file.ReadAt(buffer, beg); err != nil {
		return "", errors.New("failed to read code")
	}

	return string(buffer), nil
}

func GetFuncDeclOffset(decl *ast.FuncDecl, fileSet *token.FileSet) (int, int, error) {
	if decl == nil {
		return 0, 0, errors.New("invalid input")
	}

	beg := decl.Pos()
	end := decl.End()
	// if decl.Doc != nil {
	// 	for _, comment := range decl.Doc.List {
	// 		beg = Min[token.Pos](beg, comment.Pos())
	// 		end = Max[token.Pos](end, comment.End())
	// 	}
	// }

	if decl.Type != nil {
		beg = Min[token.Pos](beg, decl.Type.Pos())
		end = Max[token.Pos](end, decl.Type.End())
	}

	return fileSet.Position(beg).Offset, fileSet.Position(end).Offset, nil
}

func GetGenDeclOffset(decl *ast.GenDecl, fileSet *token.FileSet) (int, int, error) {
	if decl == nil {
		return 0, 0, errors.New("invalid input")
	}

	// var beg token.Pos = math.MaxInt
	// var end token.Pos

	beg := decl.Pos()
	end := decl.End()

	if decl.Doc != nil {
		for _, comment := range decl.Doc.List {
			if comment.Pos() != token.NoPos {
				beg = Min[token.Pos](beg, comment.Pos())
			}
			if comment.End() != token.NoPos {
				end = Max[token.Pos](end, comment.End())
			}
		}
	}

	if decl.Lparen != token.NoPos {
		beg = Min[token.Pos](beg, decl.Lparen)
	}
	if decl.Rparen != token.NoPos {
		end = Max[token.Pos](end, decl.Rparen)
	}

	return fileSet.Position(beg).Offset, fileSet.Position(end).Offset, nil
}
