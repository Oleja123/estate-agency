package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func processFile(path string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	// clear top-level comment groups
	f.Comments = nil

	ast.Inspect(f, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.File:
			v.Comments = nil
		case *ast.GenDecl:
			v.Doc = nil
		case *ast.FuncDecl:
			v.Doc = nil
		case *ast.Field:
			v.Doc = nil
			v.Comment = nil
		case *ast.ValueSpec:
			v.Doc = nil
			v.Comment = nil
		case *ast.TypeSpec:
			v.Doc = nil
		}
		return true
	})

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer file.Close()
	cfg := &printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 8}
	if err := cfg.Fprint(file, fset, f); err != nil {
		return fmt.Errorf("print %s: %w", path, err)
	}
	return nil
}

func main() {
	root := "cmd/app"
	var errs int
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, ".gen.go") || strings.Contains(path, "/vendor/") {
			return nil
		}
		fmt.Println("processing:", path)
		if err := processFile(path); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			errs++
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "walk error:", err)
		os.Exit(1)
	}
	if errs > 0 {
		os.Exit(2)
	}
	fmt.Println("done")
}
