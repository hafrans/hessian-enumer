package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"go/ast"
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
)

type Package struct {
	name  string
	defs  map[*ast.Ident]types.Object
	files []*File
}

type File struct {
	pkg         *Package
	fileName    string
	syntaxTree  *ast.File
	parsed      bool
	accumulator int64
	typeName    string
	typeValues  []*Value
}

type Value struct {
	name   string
	signed bool
	value  int64
	repl   string
}

type FileBuffer struct {
	path string
	fileName string
	fileSuffix string
	buffer bytes.Buffer
}


type Generator struct {
	files []*FileBuffer
}


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

	scanPackages(tags, packageLocations...)
}


func traverseNode(node *ast.Node) bool {

	return true
}

// scan all packages provided by pattern
func scanPackages(buildTags []string, pattern ...string) {

	pkgCfg := &packages.Config{
		Mode: packages.NeedFiles | packages.NeedName | packages.NeedSyntax |
			packages.NeedModule | packages.NeedTypesInfo | packages.NeedTypes,
		Tests:      false,
		Context:    context.TODO(),
		BuildFlags: []string{fmt.Sprintf("-tags=%s", strings.Join(buildTags, " "))},
	}

	_, err := packages.Load(pkgCfg, pattern...)

	if err != nil {
		log.Fatalf("fatal error when scan package %s", err)
	}

}
