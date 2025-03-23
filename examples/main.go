package main

import (
	"fmt"
	"os"
	"strings"

	goretriever "github.com/o0lele/go-retriver"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/pointer"
	"golang.org/x/tools/go/ssa/ssautil"
)

func Test() {
	// 生成Go Packages
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.LoadAllSyntax,
		Dir:  "../detour-go/detour",
	})
	if err != nil {
		panic(err.Error())
	}

	// for _, p := range pkgs {
	// 	b, _ := p.MarshalJSON()
	// 	println(string(b))
	// }

	// 生成ssa 构建编译
	// var mode ssa.BuilderMode
	// mode.Set("CDP")
	prog, ssaPkgs := ssautil.Packages(pkgs, 0)
	prog.Build()

	// println(len(ssaPkgs))

	// for _, s := range ssaPkgs {
	// 	println(s.Pkg)
	// }

	// 使用pointer生成调用链路
	result, err := pointer.Analyze(&pointer.Config{
		Mains:          ssaPkgs,
		BuildCallGraph: true,
	})
	if err != nil {
		println(result)
		panic(err.Error())
	}

	// 遍历调用链路
	var callMap = make(map[string]map[string]bool)
	visitFunc := func(edge *callgraph.Edge) error {
		if edge == nil {
			return nil
		}
		// 解析调用者和被调用者
		caller, callee, err := parseCallEdge(edge)
		if err != nil {
			panic(err.Error())
		}
		// 记录调用关系
		if callMap[caller] == nil {
			callMap[caller] = make(map[string]bool)
		}
		callMap[caller][callee] = true
		return nil
	}
	err = callgraph.GraphVisitEdges(result.CallGraph, visitFunc)
	if err != nil {
		panic(err.Error())
	}

	println(callMap)
}

func parseCallEdge(edge *callgraph.Edge) (string, string, error) {
	const callArrow = "-->"
	edgeStr := fmt.Sprintf("%+v", edge)
	strArray := strings.Split(edgeStr, callArrow)
	if len(strArray) != 2 {
		return "", "", fmt.Errorf("invalid format: %v", edgeStr)
	}
	callerNodeStr, calleeNodeStr := strArray[0], strArray[1]
	caller, callee := getCallRoute(callerNodeStr), getCallRoute(calleeNodeStr)
	return caller, callee, nil
}

func getCallRoute(nodeStr string) string {
	nodeStr = strings.TrimSpace(nodeStr)
	if strings.Contains(nodeStr, ":") {
		nodeStr = nodeStr[strings.Index(nodeStr, ":")+1:]
	}
	nodeStr = strings.ReplaceAll(nodeStr, "*", "")
	nodeStr = strings.ReplaceAll(nodeStr, "(", "")
	nodeStr = strings.ReplaceAll(nodeStr, ")", "")
	nodeStr = strings.ReplaceAll(nodeStr, "<", "")
	nodeStr = strings.ReplaceAll(nodeStr, ">", "")
	if strings.Contains(nodeStr, "$") {
		nodeStr = nodeStr[:strings.Index(nodeStr, "$")]
	}
	if strings.Contains(nodeStr, "#") {
		nodeStr = nodeStr[:strings.Index(nodeStr, "#")]
	}
	return strings.TrimSpace(nodeStr)
}

func GetStructedData() {
	pkgs := goretriever.Parse("../detour-go")

	file, err := os.OpenFile("output.txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		panic(err)
	}
	var parentSeg = "####"
	var childSeg = "###"
	// to text file
	for _, pkg := range pkgs {
		for _, s := range pkg.Structs {
			file.WriteString(parentSeg)
			file.WriteString(s.Code)
			file.WriteString(childSeg)
			file.WriteString(pkg.Name + "/" + s.Name)

			for _, m := range s.Methods {
				file.WriteString(parentSeg)
				file.WriteString(m.Code)
				file.WriteString(childSeg)
				file.WriteString(pkg.Name + "/" + s.Name + "." + m.Name)
			}
		}

		for _, f := range pkg.Functions {
			file.WriteString(parentSeg)
			file.WriteString(f.Code)
			file.WriteString(childSeg)
			file.WriteString(pkg.Name + "/" + f.Name)
		}
		break
	}
	file.Close()
}

func main() {
	Test()
}
