package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/o0lele/go-retriver/parser"
)

func main() {
	pkgs := parser.Parse("D:\\Code\\detour-go-main")

	b, err := json.Marshal(pkgs)
	if err != nil {
		log.Fatal(err)
	}

	os.WriteFile("output.json", b, 0644)
}
