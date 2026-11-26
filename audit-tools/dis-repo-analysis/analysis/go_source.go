package analysis

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
)

// addGoFileSignals analyses a Go source file to add signals indicating whether it is an application or a library.
func addGoFileSignals(name, root string, data []byte, signals map[string][]string) error {
	file, err := parser.ParseFile(token.NewFileSet(), name, data, 0)
	if err != nil {
		return fmt.Errorf("parse %s: %w", name, err)
	}
	if file.Name.Name == "main" {
		for _, declaration := range file.Decls {
			if function, ok := declaration.(*ast.FuncDecl); ok && function.Recv == nil && function.Name.Name == "main" {
				signals["application"] = append(signals["application"], name+": Go main function")
			}
		}
	} else if path.Dir(name) == root && len(signals["library"]) == 0 {
		signals["library"] = append(signals["library"], name+": reusable Go module-root package")
	}
	return nil
}
