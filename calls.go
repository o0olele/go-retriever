package goretriever

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

// FuncDescriptor stores an information about
// id, type and if function requires custom instrumentation.
type FuncDescriptor struct {
	Id       string
	DeclType string
}

// Function TypeHash. Each function is itentified by its
// id and type.
func (fd FuncDescriptor) TypeHash() string {
	return fd.Id + fd.DeclType
}

// LoadMode. Tells about needed information during analysis.
const LoadMode packages.LoadMode = packages.NeedName |
	packages.NeedTypes |
	packages.NeedSyntax |
	packages.NeedTypesInfo |
	packages.NeedFiles

func getPkgs(projectPath string, packagePattern []string, fset *token.FileSet, envs []string) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Fset: fset,
		Mode: LoadMode,
		Dir:  projectPath,
		Env:  envs,
	}

	return packages.Load(cfg, packagePattern...)
}

// FindRootFunctions looks for all root functions eg. entry points.
// Currently an entry point is a function that contains call of function
// passed as functionLabel paramaterer.
func FindRootFunctions(projectPath string, packagePattern []string, functionLabel string, envs []string) []*FuncDescriptor {
	var (
		fset          = token.NewFileSet()
		pkgs, _       = getPkgs(projectPath, packagePattern, fset, envs)
		currentFun    *FuncDescriptor
		rootFunctions []*FuncDescriptor
	)

	for _, pkg := range pkgs {
		for _, node := range pkg.Syntax {
			ast.Inspect(node, func(n ast.Node) bool {
				switch xNode := n.(type) {
				case *ast.CallExpr:
					selector, ok := xNode.Fun.(*ast.SelectorExpr)
					if !ok {
						break
					}
					if selector.Sel.Name == functionLabel {
						rootFunctions = append(rootFunctions, currentFun)
					}

				case *ast.FuncDecl:
					def := pkg.TypesInfo.Defs[xNode.Name]
					if def == nil {
						break
					}

					currentFun = &FuncDescriptor{
						Id:       def.Pkg().Path() + "." + def.Name(),
						DeclType: def.Type().String(),
					}
				}
				return true
			})
		}
	}
	return rootFunctions
}

// GetMostInnerAstIdent takes most inner identifier used for
// function call. For a.b.foo(), `b` will be the most inner identifier.
func GetMostInnerAstIdent(inSel *ast.SelectorExpr) *ast.Ident {
	var l []*ast.Ident
	var e ast.Expr
	e = inSel
	for e != nil {
		if _, ok := e.(*ast.Ident); ok {
			l = append(l, e.(*ast.Ident))
			break
		} else if _, ok := e.(*ast.SelectorExpr); ok {
			l = append(l, e.(*ast.SelectorExpr).Sel)
			e = e.(*ast.SelectorExpr).X
		} else if _, ok := e.(*ast.CallExpr); ok {
			e = e.(*ast.CallExpr).Fun
		} else if _, ok := e.(*ast.IndexExpr); ok {
			e = e.(*ast.IndexExpr).X
		} else if _, ok := e.(*ast.UnaryExpr); ok {
			e = e.(*ast.UnaryExpr).X
		} else if _, ok := e.(*ast.ParenExpr); ok {
			e = e.(*ast.ParenExpr).X
		} else if _, ok := e.(*ast.SliceExpr); ok {
			e = e.(*ast.SliceExpr).X
		} else if _, ok := e.(*ast.IndexListExpr); ok {
			e = e.(*ast.IndexListExpr).X
		} else if _, ok := e.(*ast.StarExpr); ok {
			e = e.(*ast.StarExpr).X
		} else if _, ok := e.(*ast.TypeAssertExpr); ok {
			e = e.(*ast.TypeAssertExpr).X
		} else if _, ok := e.(*ast.CompositeLit); ok {
			// TODO dummy implementation
			if len(e.(*ast.CompositeLit).Elts) == 0 {
				e = e.(*ast.CompositeLit).Type
			} else {
				e = e.(*ast.CompositeLit).Elts[0]
			}
		} else if _, ok := e.(*ast.KeyValueExpr); ok {
			e = e.(*ast.KeyValueExpr).Value
		} else {
			// TODO this is uncaught expression
			panic("uncaught expression")
		}
	}
	if len(l) < 2 {
		panic("selector list should have at least 2 elems")
	}
	// caller or receiver is always
	// at position 1, function is at 0
	return l[1]
}

// GetPkgPathFromRecvInterface builds package path taking
// receiver interface into account.
func GetPkgPathFromRecvInterface(pkg *packages.Package,
	pkgs []*packages.Package, funDeclNode *ast.FuncDecl, interfaces map[string]bool,
) string {
	var pkgPath string
	for _, v := range funDeclNode.Recv.List {
		for _, dependentpkg := range pkgs {
			for _, defs := range dependentpkg.TypesInfo.Defs {
				if defs == nil {
					continue
				}
				inter, ok := defs.Type().Underlying().(*types.Interface)
				if !ok {
					continue
				}
				if len(v.Names) == 0 {
					continue
				}
				def := pkg.TypesInfo.Defs[v.Names[0]]
				if def == nil {
					continue
				}

				if types.Implements(def.Type(), inter) {
					interfaceExists := interfaces[defs.Type().String()]
					if interfaceExists {
						pkgPath = defs.Type().String()
					}
					break
				}
			}
		}
	}
	return pkgPath
}

// GetPkgPathFromFunctionRecv build package path taking function receiver parameters.
func GetPkgPathFromFunctionRecv(pkg *packages.Package,
	pkgs []*packages.Package, funDeclNode *ast.FuncDecl, interfaces map[string]bool) string {
	pkgPath := GetPkgPathFromRecvInterface(pkg, pkgs, funDeclNode, interfaces)
	if len(pkgPath) != 0 {
		return pkgPath
	}
	for _, v := range funDeclNode.Recv.List {
		if len(v.Names) == 0 {
			continue
		}
		funType := pkg.TypesInfo.Defs[v.Names[0]].Type()
		pkgPath = funType.String()
		// We don't care if that's pointer, remove it from
		// type id
		if _, ok := funType.(*types.Pointer); ok {
			pkgPath = strings.TrimPrefix(pkgPath, "*")
		}
		// We don't care if called via index, remove it from
		// type id
		if _, ok := funType.(*types.Slice); ok {
			pkgPath = strings.TrimPrefix(pkgPath, "[]")
		}
	}

	return pkgPath
}

// GetSelectorPkgPath builds packages path according to selector expr.
func GetSelectorPkgPath(sel *ast.SelectorExpr, pkg *packages.Package, pkgPath string) string {
	caller := GetMostInnerAstIdent(sel)
	if caller == nil {
		return pkgPath
	}
	obj := pkg.TypesInfo.Uses[caller]
	if obj == nil {
		return pkgPath
	}
	if strings.Contains(obj.Type().String(), "invalid") {
		return pkgPath
	}

	pkgPath = obj.Type().String()
	// We don't care if that's pointer, remove it from
	// type id
	if _, ok := obj.Type().(*types.Pointer); ok {
		pkgPath = strings.TrimPrefix(pkgPath, "*")
	}
	// We don't care if called via index, remove it from
	// type id
	if _, ok := obj.Type().(*types.Slice); ok {
		pkgPath = strings.TrimPrefix(pkgPath, "[]")
	}

	return pkgPath
}

// GetPkgNameFromUsesTable gets package name from uses table.
func GetPkgNameFromUsesTable(pkg *packages.Package, ident *ast.Ident) string {
	var pkgPath string
	var obj = pkg.TypesInfo.Uses[ident]
	if obj == nil {
		return pkgPath
	}
	if obj.Pkg() != nil {
		pkgPath = obj.Pkg().Path()
	}
	return pkgPath
}

// GetPkgNameFromDefsTable gets package name from uses table.
func GetPkgNameFromDefsTable(pkg *packages.Package, ident *ast.Ident) string {
	var pkgPath string
	var def = pkg.TypesInfo.Defs[ident]
	if def == nil {
		return pkgPath
	}
	if def.Pkg() != nil {
		pkgPath = def.Pkg().Path()
	}
	return pkgPath
}

// GetPkgPathForFunction builds package path, delegates work to
// other helper functions defined above.
func GetPkgPathForFunction(pkg *packages.Package,
	pkgs []*packages.Package, funDecl *ast.FuncDecl, interfaces map[string]bool) string {
	if funDecl.Recv != nil {
		return GetPkgPathFromFunctionRecv(pkg, pkgs, funDecl, interfaces)
	}
	return GetPkgNameFromDefsTable(pkg, funDecl.Name)
}

func getTypesObject(pkg *packages.Package, id *ast.Ident) types.Object {
	return pkg.TypesInfo.Uses[id]
}

func newFuncDescFromCallIdent(pkg *packages.Package, f ast.Expr) *FuncDescriptor {
	id, ok := f.(*ast.Ident)
	if !ok {
		return nil
	}

	obj := getTypesObject(pkg, id)
	if obj == nil {
		return nil
	}

	var pkgPath = GetPkgNameFromUsesTable(pkg, id)
	// fmt.Println("\t\t\tFuncCall:", funId, pkg.TypesInfo.Uses[id].Type().String(),
	// 	" @called : ",
	// 	fset.File(node.Pos()).Name())
	return &FuncDescriptor{
		Id:       pkgPath + "." + obj.Name(),
		DeclType: obj.Type().String(),
	}
}

func newFuncDescFromCallSelector(pkg *packages.Package, f ast.Expr) *FuncDescriptor {
	sel, ok := f.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	obj := getTypesObject(pkg, sel.Sel)
	if obj == nil {
		return nil
	}

	pkgPath := GetPkgNameFromUsesTable(pkg, sel.Sel)
	if sel.X != nil {
		pkgPath = GetSelectorPkgPath(sel, pkg, pkgPath)
	}

	// fmt.Println("\t\t\tFuncCall via selector:", funId, pkg.TypesInfo.Uses[sel.Sel].Type().String(),
	// 	" @called : ",
	// 	fset.File(node.Pos()).Name())
	return &FuncDescriptor{
		Id:       pkgPath + "." + obj.Name(),
		DeclType: obj.Type().String(),
	}
}

// BuildCallGraph builds an information about flow graph
// in the following form child->parent.
func BuildCallGraph(
	projectPath string,
	packagePattern []string,
	funcDecls map[*FuncDescriptor]bool,
	interfaces map[string]bool,
	envs []string,
) (map[*FuncDescriptor][]*FuncDescriptor, map[*FuncDescriptor][]*FuncDescriptor) {
	var (
		fset              = token.NewFileSet()
		pkgs, _           = getPkgs(projectPath, packagePattern, fset, envs)
		currentFun        *FuncDescriptor
		backwardCallGraph = make(map[*FuncDescriptor][]*FuncDescriptor)
		forwardCallGraph  = make(map[*FuncDescriptor][]*FuncDescriptor)
	)

	for _, pkg := range pkgs {

		for _, node := range pkg.Syntax {
			// fmt.Println("\t\t", fset.File(node.Pos()).Name())
			ast.Inspect(node, func(n ast.Node) bool {
				switch xNode := n.(type) {
				case *ast.CallExpr:

					fun := newFuncDescFromCallIdent(pkg, xNode.Fun)
					if fun == nil {
						fun = newFuncDescFromCallSelector(pkg, xNode.Fun)
					}

					if fun != nil {
						if !Contains(backwardCallGraph[fun], currentFun) {
							if funcDecls[fun] {
								backwardCallGraph[fun] = append(backwardCallGraph[fun], currentFun)
							}
						}
						if !Contains(forwardCallGraph[currentFun], fun) {
							forwardCallGraph[currentFun] = append(forwardCallGraph[currentFun], fun)
						}
					}

				case *ast.FuncDecl:
					def, ok := pkg.TypesInfo.Defs[xNode.Name]
					if ok {
						var pkgPath = GetPkgPathForFunction(pkg, pkgs, xNode, interfaces)
						var funId = pkgPath + "." + def.Name()
						var fun = &FuncDescriptor{
							Id:       funId,
							DeclType: def.Type().String(),
						}
						funcDecls[fun] = true
						currentFun = fun

						// fmt.Println("\t\t\tFuncDecl:", funId, def.Type().String())
					}
				}
				return true
			})
		}
	}
	return backwardCallGraph, forwardCallGraph
}

// FindFuncDecls looks for all function declarations.
func FindFuncDecls(projectPath string, packagePattern []string, interfaces map[string]bool, envs []string) map[*FuncDescriptor]bool {
	var (
		fset      = token.NewFileSet()
		pkgs, _   = getPkgs(projectPath, packagePattern, fset, envs)
		funcDecls = make(map[*FuncDescriptor]bool)
	)

	for _, pkg := range pkgs {
		for _, node := range pkg.Syntax {
			fmt.Println("\t\t", fset.File(node.Pos()).Name())

			ast.Inspect(node, func(n ast.Node) bool {
				funDeclNode, ok := n.(*ast.FuncDecl)
				if !ok {
					return true
				}

				var pkgPath = GetPkgPathForFunction(pkg, pkgs, funDeclNode, interfaces)

				def, ok := pkg.TypesInfo.Defs[funDeclNode.Name]
				if !ok {
					return true
				}

				var fun = &FuncDescriptor{
					Id:       pkgPath + "." + def.Name(),
					DeclType: def.Type().String(),
				}

				funcDecls[fun] = true
				return true
			})
		}
	}
	return funcDecls
}

// FindInterfaces looks for all interfaces.
func FindInterfaces(projectPath string, packagePattern []string, envs []string) map[string]bool {
	var (
		fset          = token.NewFileSet()
		pkgs, _       = getPkgs(projectPath, packagePattern, fset, envs)
		interaceTable = make(map[string]bool)
	)

	for _, pkg := range pkgs {
		for _, node := range pkg.Syntax {
			ast.Inspect(node, func(n ast.Node) bool {
				typeSpecNode, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}
				_, ok = typeSpecNode.Type.(*ast.InterfaceType)
				if !ok {
					return true
				}
				def, ok := pkg.TypesInfo.Defs[typeSpecNode.Name]
				if !ok {
					return true
				}
				interaceTable[def.Type().String()] = true
				return true
			})
		}
	}
	return interaceTable
}

// InferRootFunctionsFromGraph tries to infer entry points from passed call graph.
func InferRootFunctionsFromGraph(callgraph map[*FuncDescriptor][]*FuncDescriptor) []*FuncDescriptor {
	var allFunctions = make(map[*FuncDescriptor]bool)
	var rootFunctions []*FuncDescriptor

	for k, v := range callgraph {
		allFunctions[k] = true
		for _, childFun := range v {
			allFunctions[childFun] = true
		}
	}
	for k := range allFunctions {
		_, exists := callgraph[k]
		if !exists {
			rootFunctions = append(rootFunctions, k)
		}
	}
	return rootFunctions
}

// Contains.
func Contains(a []*FuncDescriptor, x *FuncDescriptor) bool {
	for _, n := range a {
		if x.TypeHash() == n.TypeHash() {
			return true
		}
	}
	return false
}
