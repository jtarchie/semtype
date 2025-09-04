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
	"log/slog"
	"os"
	"path/filepath"
)

// Exported holds the exported types and functions from a Go package
type Exported struct {
	Types     map[string]string
	Functions map[string]string
}

// State represents the current state of the semantic versioning analysis
type State struct {
	Version  string
	Exported Exported
}

// Version represents a semantic version
type Version struct {
	Major, Minor, Patch int
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	if err := run(); err != nil {
		slog.Error("execution failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	config, err := parseFlags()
	if err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	previousState, err := loadState(config.stateFile)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	currentExported, err := analyzePackage(config.dir)
	if err != nil {
		return fmt.Errorf("analyzing package: %w", err)
	}

	newVersion := calculateVersion(previousState, currentExported)

	newState := State{
		Version:  newVersion.String(),
		Exported: currentExported,
	}

	if err := saveState(config.stateFile, newState); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	fmt.Println(newVersion.String())
	return nil
}

// config holds the parsed command line flags
type config struct {
	dir       string
	stateFile string
}

func parseFlags() (*config, error) {
	dir := flag.String("dir", "./", "directory to analyze")
	stateFile := flag.String("state", "", "path to state file")
	flag.Parse()

	if *stateFile == "" {
		*stateFile = filepath.Join(*dir, "semtype.dat")
	}

	return &config{
		dir:       *dir,
		stateFile: *stateFile,
	}, nil
}

func loadState(stateFile string) (State, error) {
	file, err := os.Open(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return State{Version: "0.0.0", Exported: Exported{
				Types:     make(map[string]string),
				Functions: make(map[string]string),
			}}, nil
		}
		return State{}, fmt.Errorf("opening state file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			slog.Warn("failed to close state file", "error", closeErr)
		}
	}()

	var state State
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&state); err != nil {
		return State{}, fmt.Errorf("decoding state file: %w", err)
	}

	return state, nil
}

func saveState(stateFile string, state State) error {
	file, err := os.Create(stateFile)
	if err != nil {
		return fmt.Errorf("creating state file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			slog.Warn("failed to close state file", "error", closeErr)
		}
	}()

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(&state); err != nil {
		return fmt.Errorf("encoding state: %w", err)
	}

	return nil
}

func analyzePackage(dir string) (Exported, error) {
	exported := Exported{
		Types:     make(map[string]string),
		Functions: make(map[string]string),
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
	if err != nil {
		return exported, fmt.Errorf("parsing directory: %w", err)
	}

	for _, pkg := range pkgs {
		if err := analyzePackageFiles(fset, pkg.Files, &exported); err != nil {
			return exported, err
		}
	}

	return exported, nil
}

func analyzePackageFiles(fset *token.FileSet, files map[string]*ast.File, exported *Exported) error {
	for _, file := range files {
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				if err := analyzeGenDecl(fset, d, exported); err != nil {
					return err
				}
			case *ast.FuncDecl:
				if err := analyzeFuncDecl(fset, d, exported); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func analyzeGenDecl(fset *token.FileSet, d *ast.GenDecl, exported *Exported) error {
	for _, spec := range d.Specs {
		if typeSpec, ok := spec.(*ast.TypeSpec); ok && typeSpec.Name.IsExported() {
			simplified := simplifyType(typeSpec.Type)
			formatted, err := formatNode(fset, simplified)
			if err != nil {
				slog.Warn("failed to format type", "name", typeSpec.Name.Name, "error", err)
				continue
			}
			exported.Types[typeSpec.Name.Name] = formatted
		}
	}
	return nil
}

func analyzeFuncDecl(fset *token.FileSet, d *ast.FuncDecl, exported *Exported) error {
	if !d.Name.IsExported() {
		return nil
	}

	formatted, err := formatNode(fset, d.Type)
	if err != nil {
		slog.Warn("failed to format function", "name", d.Name.Name, "error", err)
		return nil
	}

	exported.Functions[d.Name.Name] = formatted
	return nil
}

func simplifyType(typeNode ast.Expr) ast.Node {
	structType, ok := typeNode.(*ast.StructType)
	if !ok {
		return typeNode
	}

	// Only include exported fields in struct types
	var exportedFields []*ast.Field
	for _, field := range structType.Fields.List {
		if len(field.Names) > 0 && field.Names[0].IsExported() {
			exportedFields = append(exportedFields, field)
		}
	}

	return &ast.StructType{
		Struct: structType.Struct,
		Fields: &ast.FieldList{
			Opening: structType.Fields.Opening,
			List:    exportedFields,
			Closing: structType.Fields.Closing,
		},
	}
}

func formatNode(fset *token.FileSet, node ast.Node) (string, error) {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func calculateVersion(previousState State, currentExported Exported) Version {
	previousVersion := parseVersion(previousState.Version)

	hasBreaking := hasBreakingChanges(previousState.Exported, currentExported)
	hasFeatures := hasNewFeatures(previousState.Exported, currentExported)

	if hasBreaking {
		return Version{Major: previousVersion.Major + 1, Minor: 0, Patch: 0}
	}
	if hasFeatures {
		return Version{Major: previousVersion.Major, Minor: previousVersion.Minor + 1, Patch: 0}
	}
	return Version{Major: previousVersion.Major, Minor: previousVersion.Minor, Patch: previousVersion.Patch + 1}
}

func parseVersion(version string) Version {
	var v Version
	n, err := fmt.Sscanf(version, "%d.%d.%d", &v.Major, &v.Minor, &v.Patch)
	if err != nil || n != 3 {
		slog.Warn("failed to parse version, using default", "version", version, "error", err)
		return Version{Major: 0, Minor: 0, Patch: 0}
	}
	return v
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func hasBreakingChanges(previous, current Exported) bool {
	// Check for removed or changed types
	for name, previousType := range previous.Types {
		currentType, exists := current.Types[name]
		if !exists || currentType != previousType {
			return true
		}
	}

	// Check for removed or changed functions
	for name, previousFunc := range previous.Functions {
		currentFunc, exists := current.Functions[name]
		if !exists || currentFunc != previousFunc {
			return true
		}
	}

	return false
}

func hasNewFeatures(previous, current Exported) bool {
	// Check for new types
	for name := range current.Types {
		if _, exists := previous.Types[name]; !exists {
			return true
		}
	}

	// Check for new functions
	for name := range current.Functions {
		if _, exists := previous.Functions[name]; !exists {
			return true
		}
	}

	return false
}
