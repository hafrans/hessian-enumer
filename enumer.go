package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/types"
	"log"
	"os"
	"strconv"
	"strings"
)

const (
	EmptyString         = ""
	DefaultBase         = 10
	UnderScoreName      = "_"
	SpaceSeparator      = " "
	ParameterSeparator  = "="
	PackageSeparator    = "."
	HessianJavaEnum     = "JavaEnum"
	GoHessianHead       = "//go:hessian"
	GoHessianHeadLength = len(GoHessianHead)
)

type (
	Package struct {
		name  string
		path  string
		defs  map[*ast.Ident]types.Object
		files []*File
	}

	File struct {
		pkg        *Package
		filePath   string
		syntaxTree *ast.File
		EnumTypes  map[string][]*Value
	}

	Value struct {
		name     string
		fullName string
		signed   bool
		value    uint64
		repl     string
	}


)

func (v *Value) String() string {
	if v.repl == EmptyString {
		if v.signed {
			return strconv.FormatInt(int64(v.value), DefaultBase)
		} else {
			return strconv.FormatUint(v.value, DefaultBase)
		}
	}
	return v.repl
}

var (
	typeNames = flag.String("path", "", "comma-separated list of path")
	className = flag.String("package", "", "canonical java package (eg. com.hafrans.test), case sensitive")
	buildTags = flag.String("tags", "", "comma-separated list of build tags to apply")
)

func init() {
	log.SetFlags(0)
	log.SetPrefix("hessian-enumer:")
}

func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of hessian2 JavaEnum generator \n")
	fmt.Fprintf(os.Stderr, "\t hessisan-enumer [flags] - [directory]")
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
	var javaPackagePrefix = "com.demo"
	var _ = "Pill2"
	var parser Parser = Parser{
		buildTags:         tags,
		targetFiles: make(map[*File]struct{}),
		typeClassMap:      make(map[string]string),
		javaPackagePrefix: strings.Trim(javaPackagePrefix, PackageSeparator),
	}

	log.Println("starting scanning packages")
	parser.scanPackages(packageLocations...)
	log.Println("starting parse types")
	parser.parseType()
	log.Println("all enum classes is generated.")
}
