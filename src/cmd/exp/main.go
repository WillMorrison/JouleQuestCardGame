package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"strings"

	"golang.org/x/tools/go/packages"
)

type AnnotatedDecl struct {
	decl       ast.Decl
	directives []ast.Directive
	pkg        *packages.Package
}

func (as AnnotatedDecl) Position() token.Position {
	return as.pkg.Fset.Position(as.decl.Pos())
}

func (as AnnotatedDecl) MatchDirective(tool string, name string) bool {
	for _, directive := range as.directives {
		if directive.Tool == tool && directive.Name == name {
			return true
		}
	}
	return false
}

type ConstSpec struct {
	Name  string
	Type  string
	Value string
}

func (as AnnotatedDecl) ConstSpecs() []ConstSpec {
	var constSpecs []ConstSpec
	gd, ok := as.decl.(*ast.GenDecl)
	if !ok {
		return constSpecs
	}
	if gd.Tok != token.CONST {
		return constSpecs
	}
	for _, spec := range gd.Specs {
		if cv, ok := spec.(*ast.ValueSpec); ok {
			for _, name := range cv.Names {
				to := as.pkg.TypesInfo.Defs[name]
				if co, ok := to.(*types.Const); ok {
					constSpecs = append(constSpecs, ConstSpec{name.Name, to.Type().String(), co.Val().String()})
				}
			}
		}
	}
	return constSpecs
}

type Arg struct {
	Name string
	Type string
}

type FuncSignature struct {
	Name string
	Args []Arg

	Returns []string
}

func (fs FuncSignature) String() string {
	var b strings.Builder
	b.WriteString(fs.Name)
	b.WriteString("(")
	for i, arg := range fs.Args {
		if i != 0 {
			b.WriteString(", ")
		}
		b.WriteString(arg.Name)
		b.WriteString(": ")
		b.WriteString(arg.Type)
	}
	b.WriteString(") -> (")
	for _, ret := range fs.Returns {
		b.WriteString(ret)
	}
	b.WriteString(")")
	return b.String()
}

func (as AnnotatedDecl) FuncSignature() FuncSignature {
	fd, ok := as.decl.(*ast.FuncDecl)
	if !ok {
		return FuncSignature{}
	}
	fs := FuncSignature{Name: fd.Name.Name}
	sig := as.pkg.TypesInfo.Defs[fd.Name].Type()
	if sig, ok := sig.(*types.Signature); ok {
		for v := range sig.Params().Variables() {
			fs.Args = append(fs.Args, Arg{Name: v.Name(), Type: v.Type().String()})
		}
		for v := range sig.Results().Variables() {
			fs.Returns = append(fs.Returns, v.Type().String())
		}
	}
	return fs
}

type TaggedField struct {
	Name      string
	Tag       string
	export    bool         // Generate scalar getter
	set       bool         // Generate scalar setter
	enum      *ast.Ident   // Generate enum wrapper for the setter arg or getter result
	index     string       // Generate an index argument with this name
	indexEnum *ast.Ident   // The index argument should be this enum type
	nest      TaggedStruct // Nested structures with tagged fields

	pkg   *packages.Package
	field *ast.Field
}

func directivesFromComments(cGroup *ast.CommentGroup) []ast.Directive {
	if cGroup == nil {
		return nil
	}
	var directives []ast.Directive
	for _, c := range cGroup.List {
		if d, ok := ast.ParseDirective(c.Pos(), c.Text); ok {
			directives = append(directives, d)
		}
	}
	return directives
}

type TaggedStruct struct {
	Ident        *ast.Ident
	TaggedFields []TaggedField
}

func TaggedStructFromTypeSpec(ts *ast.TypeSpec, pkg *packages.Package) TaggedStruct {
	ident := ts.Name
	var taggedFields []TaggedField
	if ts.Type.(*ast.StructType).Fields == nil {
		return TaggedStruct{}
	}
	for _, field := range ts.Type.(*ast.StructType).Fields.List {
		if field.Tag != nil && field.Tag.Kind == token.STRING {
			tag := field.Tag.Value
			taggedFields = append(taggedFields, TaggedField{Name: field.Names[0].Name, Tag: tag, pkg: pkg, field: field})
		}
	}
	if len(taggedFields) == 0 {
		return TaggedStruct{}
	}
	return TaggedStruct{Ident: ident, TaggedFields: taggedFields}
}

type Info struct {
	Decls         []AnnotatedDecl
	TaggedStructs []TaggedStruct
}

func packageInfo(pkg *packages.Package) Info {
	var info Info
	for _, astFile := range pkg.Syntax {
		for _, decl := range astFile.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				if d.Tok == token.CONST {
					directives := directivesFromComments(d.Doc)
					if len(directives) == 0 {
						continue
					}
					info.Decls = append(info.Decls, AnnotatedDecl{decl, directives, pkg})
				} else if d.Tok == token.TYPE {
					for _, spec := range d.Specs {
						if ts, ok := spec.(*ast.TypeSpec); ok {
							if _, ok := ts.Type.(*ast.StructType); ok {
								tagged := TaggedStructFromTypeSpec(ts, pkg)
								if tagged.TaggedFields != nil {
									info.TaggedStructs = append(info.TaggedStructs, tagged)
								}
							}
						}
					}
				}
			case *ast.FuncDecl:
				directives := directivesFromComments(d.Doc)
				if len(directives) == 0 {
					continue
				}
				info.Decls = append(info.Decls, AnnotatedDecl{decl, directives, pkg})
			}
		}
	}
	return info
}

func main() {
	pkgPath := flag.String("pkg", "github.com/WillMorrison/JouleQuestCardGame/compact/wasm", "Go package to scan for const blocks.")
	flag.Parse()
	if *pkgPath == "" {
		flag.Usage()
		log.Fatal("missing -pkg")
	}

	pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedName | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedDeps}, *pkgPath)
	if err != nil {
		log.Fatal(err)
	}

	var allInfo Info
	packages.Visit(
		pkgs,
		func(pkg *packages.Package) bool {
			return strings.HasPrefix(pkg.PkgPath, "github.com/WillMorrison/JouleQuestCardGame")
		},
		func(pkg *packages.Package) {
			info := packageInfo(pkg)
			allInfo.Decls = append(allInfo.Decls, info.Decls...)
			allInfo.TaggedStructs = append(allInfo.TaggedStructs, info.TaggedStructs...)
		},
	)

	for _, annotatedDecl := range allInfo.Decls {
		fmt.Println(annotatedDecl.Position().String())
		for _, directive := range annotatedDecl.directives {
			fmt.Printf("//%s:%s %s\n", directive.Tool, directive.Name, directive.Args)
		}
		switch annotatedDecl.decl.(type) {
		case *ast.GenDecl:
			for _, constSpec := range annotatedDecl.ConstSpecs() {
				fmt.Println("const", constSpec.Name, constSpec.Type, "=", constSpec.Value)
			}
		case *ast.FuncDecl:
			fmt.Println("func", annotatedDecl.FuncSignature().String())
		}
	}
	for _, taggedStruct := range allInfo.TaggedStructs {
		fmt.Println(taggedStruct.Ident.Name)
		for _, taggedField := range taggedStruct.TaggedFields {
			fmt.Println("  ", taggedField.Name, taggedField.Tag)
		}
	}
}
