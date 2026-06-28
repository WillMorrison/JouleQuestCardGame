package wasmcodegen

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"strings"

	"github.com/iancoleman/strcase"
	"golang.org/x/tools/go/packages"
)

const (
	LoadMode = packages.NeedName | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo
)

type Int32Val struct{}

func (v Int32Val) String() string {
	return "int32"
}

func (v Int32Val) PythonType() string {
	return "int"
}

func (v Int32Val) ClientValType() string {
	return "ValType.I32"
}

type Arg struct {
	Name string
	Int32Val
}

func (a Arg) String() string {
	return fmt.Sprintf("%s %s", a.Name, a.Int32Val.String())
}

func (a Arg) SnakeName() string {
	return strcase.ToSnake(a.Name)
}

type Result struct {
	Int32Val
}

func (r Result) String() string {
	return r.Int32Val.String()
}

type WasmExportedFunc struct {
	// Directives associated with this function (len>0)
	Directives []ast.Directive
	// Go function name
	FuncName   string
	FuncArgs   []Arg
	FuncResult []Result
	// Name from the //go:wasmexport directive. Empty if not exported
	WasmExportName string
	// WASM module name, defaults to the package name
	WasmModuleName string
}

func (wef WasmExportedFunc) String() string {
	var b strings.Builder
	b.WriteString("func ")
	b.WriteString(wef.WasmModuleName)
	b.WriteString(".")
	b.WriteString(wef.WasmExportName)
	b.WriteString("(")
	for i, a := range wef.FuncArgs {
		if i != 0 {
			b.WriteString(", ")
		}
		b.WriteString(a.String())
	}
	b.WriteString(")->(")
	for i, r := range wef.FuncResult {
		if i != 0 {
			b.WriteString(", ")
		}
		b.WriteString(r.String())
	}
	b.WriteString(")")
	return b.String()
}

func (wef WasmExportedFunc) SnakeName() string {
	return strcase.ToSnake(wef.WasmExportName)
}

type Package struct {
	Name    string
	PkgPath string
	Exports []WasmExportedFunc

	fset     *token.FileSet
	typeInfo *types.Info
}

// Load returns all packages with WASM generator information
func Load(pattern string) ([]Package, error) {
	cfg := packages.Config{Mode: LoadMode}
	pkgs, err := packages.Load(&cfg, pattern)
	if err != nil {
		return nil, err
	}
	pres := make([]Package, 0, len(pkgs))
	for _, pkg := range pkgs {
		p := Package{
			Name:     pkg.Name,
			PkgPath:  pkg.PkgPath,
			fset:     pkg.Fset,
			typeInfo: pkg.TypesInfo,
		}
		for _, astFile := range pkg.Syntax {
			for _, d := range astFile.Decls {
				if err := p.maybeAddWasmExportedFunc(d); err != nil {
					log.Printf("warn: %s", err)
				}
			}

		}
		if len(p.Exports) != 0 {
			pres = append(pres, p)
		}
	}
	return pres, nil
}

func (p *Package) maybeAddWasmExportedFunc(decl ast.Decl) error {
	// first check if this decl is a function with a //go:wasmexport directive
	fd, ok := decl.(*ast.FuncDecl)
	if !ok || fd == nil {
		return nil // Node was not a function declaration
	}
	if fd.Doc == nil {
		return nil // Node has no doc comments (thus no directives apply)
	}
	wef := WasmExportedFunc{
		FuncName:       fd.Name.Name,
		WasmModuleName: p.Name,
	}
	// Parse doc comments for directives
	for _, c := range fd.Doc.List {
		if d, ok := ast.ParseDirective(c.Pos(), c.Text); ok {
			wef.Directives = append(wef.Directives, d)
			if d.Tool == "go" && d.Name == "wasmexport" {
				wef.WasmExportName = d.Args
			}
		}
	}
	if wef.WasmExportName == "" {
		return nil // No wasmexport directive
	}

	// Warn about things we don't want to see exported
	if fd.Recv != nil {
		return fmt.Errorf("%s: //go:wasmexport directive on a function %q with receiver", p.fset.Position(fd.Pos()), fd.Name.String())
	}
	if fd.Type.TypeParams != nil {
		return fmt.Errorf("%s: //go:wasmexport directive on a generic function %q", p.fset.Position(fd.Pos()), fd.Name.String())
	}

	// Get function signature
	defType, ok := p.typeInfo.Defs[fd.Name]
	if !ok {
		return fmt.Errorf("%s: cannot get type of %s", p.fset.Position(fd.Pos()), fd.Name.String())
	}
	fnType, ok := defType.(*types.Func)
	if !ok {
		return fmt.Errorf("%s: type of %s is not a function", p.fset.Position(fd.Pos()), fd.Name.String())
	}

	// Get parameter types (if any)
	if fnType.Signature().Params() != nil && fnType.Signature().Params().Len() > 0 {
		for argVar := range fnType.Signature().Params().Variables() {
			argType, ok := argVar.Type().(*types.Basic)
			if !ok || argType.Kind() != types.Int32 {
				return fmt.Errorf("%s: type %s of arg %s to %s is not int32", p.fset.Position(fd.Pos()), argType.Name(), argVar.Name(), fd.Name.String())
			}
			wef.FuncArgs = append(wef.FuncArgs, Arg{Name: argVar.Name(), Int32Val: Int32Val{}})
		}
	}

	// Get result type (if any)
	if fnType.Signature().Results() != nil && fnType.Signature().Results().Len() > 0 {
		if fnType.Signature().Results().Len() > 1 {
			return fmt.Errorf("%s: too many return values from %s, expect 0 or 1", p.fset.Position(fd.Pos()), fd.Name.String())
		}
		resVar := fnType.Signature().Results().At(0)
		resType, ok := resVar.Type().(*types.Basic)
		if !ok || resType.Kind() != types.Int32 {
			return fmt.Errorf("%s: return type of %s is not int32, found %s", p.fset.Position(fd.Pos()), fd.Name.String(), resType.Name())
		}
		wef.FuncResult = append(wef.FuncResult, Result{Int32Val{}})
	}

	p.Exports = append(p.Exports, wef)
	return nil
}
