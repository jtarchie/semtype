package main

import (
	"bytes"
	"encoding/gob"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
)

type Exported struct {
	Types     map[string]string
	Functions map[string]string
}

type State struct {
	Version  string
	Exported Exported
}

func main() {
	dir := flag.String("dir", "./", "directory to analyze")
	stateFile := flag.String("state", "", "path to state file")
	flag.Parse()

	if *stateFile == "" {
		*stateFile = filepath.Join(*dir, "semtype.dat")
	}

	file, err := os.Open(*stateFile)
	var previousState State
	if err != nil {
		if os.IsNotExist(err) {
			previousState = State{Version: "0.0.0"}
		} else {
			log.Fatalf("failed to open state file: %v", err)
		}
	} else {
		defer file.Close()
		decoder := gob.NewDecoder(file)
		if err := decoder.Decode(&previousState); err != nil {
			log.Fatalf("failed to decode state file: %v", err)
		}
	}

	exported := Exported{
		Types:     make(map[string]string),
		Functions: make(map[string]string),
	}
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, *dir, nil, parser.AllErrors)
	if err != nil {
		log.Fatalf("Failed to parse directory: %v", err)
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				switch d := decl.(type) {
				case *ast.GenDecl:
					for _, spec := range d.Specs {
						switch s := spec.(type) {
						case *ast.TypeSpec:
							if s.Name.IsExported() {
								// Create a simplified version of the type
								var simplifiedType ast.Node
								switch t := s.Type.(type) {
								case *ast.StructType:
									// Create new struct type without field details
									simplifiedType = &ast.StructType{
										Fields: &ast.FieldList{},
									}
								default:
									simplifiedType = t
								}

								printer := &bytes.Buffer{}
								if err := format.Node(printer, fset, simplifiedType); err != nil {
									log.Printf("Failed to format type %s: %v", s.Name.Name, err)
									continue
								}
								exported.Types[s.Name.Name] = printer.String()
							}
						}
					}
				case *ast.FuncDecl:
					if d.Name.IsExported() {
						printer := &bytes.Buffer{}
						if err := format.Node(printer, fset, d.Type); err != nil {
							log.Printf("Failed to format type %s: %v", d.Name.Name, err)
							continue
						}
						exported.Functions[d.Name.Name] = printer.String()
					}
				}
			}
		}
	}

	major, minor, patch := 0, 0, 0
	_, err = fmt.Sscanf(previousState.Version, "%d.%d.%d", &major, &minor, &patch)
	if err != nil {
		log.Fatalf("failed to parse version: %v", err)
	}

	removed := false
	added := false

	for name, typ := range previousState.Exported.Types {
		if currentType, ok := exported.Types[name]; !ok || currentType != typ {
			removed = true
			goto bump
		}
	}

	for name, typ := range exported.Types {
		if previousType, ok := previousState.Exported.Types[name]; !ok || previousType != typ {
			added = true
			goto bump
		}
	}

	for name, typ := range previousState.Exported.Functions {
		if currentFunc, ok := exported.Functions[name]; !ok || currentFunc != typ {
			removed = true
			goto bump
		}
	}

	for name, typ := range exported.Functions {
		if previousFunc, ok := previousState.Exported.Functions[name]; !ok || previousFunc != typ {
			added = true
			goto bump
		}
	}

bump:

	if removed {
		major++
		minor = 0
		patch = 0
	} else if added {
		minor++
		patch = 0
	} else {
		patch++
	}

	newVersion := fmt.Sprintf("%d.%d.%d", major, minor, patch)

	file, err = os.Create(*stateFile)
	if err != nil {
		log.Fatalf("failed to create state file: %v", err)
	}
	defer file.Close()

	state := State{
		Version:  newVersion,
		Exported: exported,
	}

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(&state); err != nil {
		log.Fatalf("failed to encode state file: %v", err)
	}

	fmt.Printf("%s\n", newVersion)
}
