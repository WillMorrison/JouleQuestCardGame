package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"text/template"

	"github.com/WillMorrison/JouleQuestCardGame/internal/wasmcodegen"
)

const (
	joulequestPackages = "github.com/WillMorrison/JouleQuestCardGame/..."
	wasmPkg            = "github.com/WillMorrison/JouleQuestCardGame/compact/wasm"
)

//go:embed *.tmpl
var tmplSources embed.FS

type tmplInput struct {
	Exports []wasmcodegen.WasmExportedFunc
}

func loadInput() (tmplInput, error) {
	log.Println("Loading packages from " + joulequestPackages)
	pkgs, err := wasmcodegen.Load(joulequestPackages)
	if err != nil {
		return tmplInput{}, fmt.Errorf("package load error: %s", err)
	}

	var ti tmplInput
	for _, pkg := range pkgs {
		if pkg.PkgPath == wasmPkg {
			ti.Exports = pkg.Exports
		}
	}
	if ti.Exports == nil {
		return tmplInput{}, fmt.Errorf("no exported functions found")
	}
	return ti, nil
}

func generate(tmpl *template.Template, ti tmplInput, tmplName, pyPath string) error {

	pyBuf := bytes.Buffer{}
	if err := tmpl.ExecuteTemplate(&pyBuf, tmplName, ti); err != nil {
		return fmt.Errorf("template execution error: %s", err)
	}
	log.Printf("Writing %s\n", pyPath)
	if err := os.WriteFile(pyPath, pyBuf.Bytes(), os.FileMode(0664)); err != nil {
		return fmt.Errorf("error writing %s: %s", pyPath, err)
	}
	return nil
}

func main() {
	outDir := flag.String("out_dir", "", "path to a directory where the generated Python code should go.")
	flag.Parse()
	if *outDir == "" {
		flag.Usage()
		log.Fatal("missing -out_dir")
	}

	input, err := loadInput()
	if err != nil {
		log.Fatalf("Load error: %s\n", err)
	}

	templates, err := template.ParseFS(tmplSources, "*")
	if err != nil {
		log.Fatalf("Template parse error: %s\n", err)
	}

	if err := os.MkdirAll(*outDir, 0755); err != nil {
		log.Fatalf("MkdirAll error: %s", err)
	}
	if err := generate(templates, input, "_api_check.py.tmpl", path.Join(*outDir, "_api_check.py")); err != nil {
		log.Fatalf("Generate error: %s", err)
	}
	if err := generate(templates, input, "_client.py.tmpl", path.Join(*outDir, "_client.py")); err != nil {
		log.Fatalf("Generate error: %s", err)
	}
	if err := generate(templates, input, "__init__.py.tmpl", path.Join(*outDir, "__init__.py")); err != nil {
		log.Fatalf("Generate error: %s", err)
	}
	if err := generate(templates, input, "__main__.py.tmpl", path.Join(*outDir, "__main__.py")); err != nil {
		log.Fatalf("Generate error: %s", err)
	}
}
