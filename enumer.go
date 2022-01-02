package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/packages"
	"log"
	"os"
	"strconv"
	"strings"
)

const (
	EmptyString = ""
	DefaultBase = 10
	HessianJavaEnum = "JavaEnum"
)

type (
	Package struct {
		name  string
		path  string
		defs  map[*ast.Ident]types.Object
		files []*File
	}

	File struct {
		pkg         *Package
		filePath    string
		syntaxTree  *ast.File
		accumulator int64
		typeName    string
		typeValues  []*Value
	}

	FileBuffer struct {
		buffer   bytes.Buffer
		filePath string
		file     *File
	}

	Value struct {
		name   string
		signed bool
		value  int64
		repl   string
	}

	Generator struct {
		pkgs        []*Package
		files       []*File
		fileBuffers []*FileBuffer
		fileSuffix  string
		buildTags   []string
	}
)

func (v *Value) String() string {
	if v.repl == EmptyString {
		if v.signed {
			return strconv.FormatInt(v.value, DefaultBase)
		} else {
			return strconv.FormatUint(uint64(v.value), DefaultBase)
		}
	}
	return v.repl
}

var (
	typeNames = flag.String("type", "", "comma-separated list of type names; must be set")
	className = flag.String("class", "", "canonical java class name (eg. com.hafrans.test.DemoEnum), case sensitive")
	buildTags = flag.String("tags", "", "comma-separated list of build tags to apply")
)

func init() {
	log.SetFlags(0)
	log.SetPrefix("hessian-enumer ")
}

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of hessian2 JavaEnum generator \n")
	fmt.Fprintf(os.Stderr, "\t hessisan-enumer [flags] -class o.a.s.TestEnum -type Test [directory]")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
}

func flagValidator() {
	flag.Usage = Usage
	flag.Parse()
}

func main() {

	var tags []string
	var packageLocations = []string{"./test/"}
	var _ = "Pill2"
	var generator Generator = Generator{
		buildTags: tags,
	}

	generator.scanPackages(packageLocations...)

	generator.parseType("Pill")

}

// scan all packages provided by pattern
func (g *Generator) scanPackages(pattern ...string) {

	pkgCfg := &packages.Config{
		Mode: packages.NeedFiles | packages.NeedName | packages.NeedSyntax |
			packages.NeedModule | packages.NeedTypesInfo | packages.NeedTypes,
		Tests:      false,
		BuildFlags: []string{fmt.Sprintf("-tags=%s", strings.Join(g.buildTags, " "))},
	}

	pkgs, err := packages.Load(pkgCfg, pattern...)

	if err != nil {
		log.Fatalf("fatal error when scan package %s", err)
	}

	if len(pkgs) == 0 {
		log.Fatalf("no appropriate packages in pattern %v", pattern)
	}

	fileLen := len(pkgs[0].Syntax)

	// append package
	pkg := &Package{
		path:  pkgs[0].PkgPath,
		defs:  pkgs[0].TypesInfo.Defs,
		name:  pkgs[0].Name,
		files: make([]*File, fileLen),
	}

	for i := 0; i < fileLen; i++ {
		pkg.files[i] = &File{
			pkg:        pkg,
			syntaxTree: pkgs[0].Syntax[i],
			filePath:   pkgs[0].GoFiles[i],
		}
	}

	g.pkgs = append(g.pkgs, pkg)
	g.files = append(g.files, pkg.files...)

}

func (g *Generator) parseType(typ string) {


	for _, file := range g.files {

		// get all decls
		DeclLoop:
		for _, decl := range file.syntaxTree.Decls {
			genDecl, ok := decl.(*ast.GenDecl)

			if !ok {
				continue
			}

			if genDecl.Tok != token.CONST {
				continue
			}
			// const block

			specs := genDecl.Specs
			if specs == nil || len(specs) == 0 {
				continue
			}

			for _, spec := range specs {

				valueSpec, _ := spec.(*ast.ValueSpec) // all spec in const block is spec

				typ, err := getAndCheckTypeName(valueSpec)
				if err != nil {
					continue DeclLoop
				}

				if typ != "" {
					if file.typeName == "" {
						file.typeName = typ
					} else if file.typeName != typ {
                        log.Fatalln("multi type in one const block is not allowed.")
					}
				}else{
					if file.typeName == "" {
						continue DeclLoop
					}else{
						typ = file.typeName
					}
				}

				// get all Name

			}

		}
	}
}


func getAndCheckTypeName(valueSpec *ast.ValueSpec) (typ string, err error) {
	typ = ""
	err = nil

	if valueSpec == nil {
		log.Fatalln("encountered an unexpect fatal error: valueSpec is nil")
	}

	if valueSpec.Type == nil {
		return "", nil
	}

	typeIdent, ok := valueSpec.Type.(*ast.Ident)

	if !ok {
		return "", errors.New("can not cast valueSpec.Type to ast.Ident")
	}

	typ = typeIdent.Name

	if len(typ) == 0 {
		return "" , errors.New("typeName is empty when get TypeName")
	}

	// check it's type
	if typeIdent.Obj == nil {
		return "" , errors.New("typeIdent Obj is nil")
	}

	typeSpec, ok := typeIdent.Obj.Decl.(*ast.TypeSpec)
	if !ok {
		return "", errors.New("can not cast typeIdent.Obj.Decl. to *ast.TypeSpec")
	}

	selectorExpr, ok := typeSpec.Type.(*ast.SelectorExpr)

	if !ok {
		return "", errors.New("can not cast typeSpec.Type. to *ast.selectorExpr")
	}

	if selectorExpr.Sel != nil && selectorExpr.Sel.Name == HessianJavaEnum {
		return typ, nil
	}

	return "", errors.New("type check failed")
}