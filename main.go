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

	previousState := loadPreviousState(*stateFile)
	currentExported := analyzeDirectory(*dir)
	newVersion := determineVersion(previousState, currentExported)
	saveCurrentState(*stateFile, newVersion, currentExported)
	fmt.Printf("New version: %s\n", newVersion)
}

func loadPreviousState(filename string) State {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return State{Version: "0.0.0"}
		}

		log.Fatalf("failed to open state file: %v", err)
	}
	defer file.Close()

	var state State
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&state); err != nil {
		log.Fatalf("failed to decode state file: %v", err)
	}

	return state
}

func analyzeDirectory(dir string) Exported {
	exported := Exported{
		Types:     make(map[string]string),
		Functions: make(map[string]string),
	}
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, parser.AllErrors)
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
								printer := &bytes.Buffer{}
								if err := format.Node(printer, fset, s.Type); err != nil {
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
	return exported
}

func determineVersion(previous State, current Exported) string {
	major, minor, patch := 0, 0, 0
	fmt.Sscanf(previous.Version, "%d.%d.%d", &major, &minor, &patch)

	removed := false
	added := false
	changed := false

	for name, typ := range previous.Exported.Types {
		if currentType, ok := current.Types[name]; !ok || currentType != typ {
			removed = true
			break
		}
	}

	for name, typ := range current.Types {
		if previousType, ok := previous.Exported.Types[name]; !ok || previousType != typ {
			added = true
			break
		}
	}

	for name, typ := range previous.Exported.Functions {
		if currentFunc, ok := current.Functions[name]; !ok || currentFunc != typ {
			removed = true
			break
		}
	}

	for name, typ := range current.Functions {
		if previousFunc, ok := previous.Exported.Functions[name]; !ok || previousFunc != typ {
			added = true
			break
		}
	}

	if removed {
		major++
		minor = 0
		patch = 0
	} else if added {
		minor++
		patch = 0
	} else if changed {
		patch++
	}

	return fmt.Sprintf("%d.%d.%d", major, minor, patch)
}

func saveCurrentState(filename, version string, exported Exported) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("failed to create state file: %v", err)
	}
	defer file.Close()

	state := State{
		Version:  version,
		Exported: exported,
	}

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(&state); err != nil {
		log.Fatalf("failed to encode state file: %v", err)
	}
}
