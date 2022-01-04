package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/packages"
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

	FileBuffer struct {
		buffer   bytes.Buffer
		filePath string
		file     *File
	}

	Value struct {
		name     string
		fullName string
		signed   bool
		value    uint64
		repl     string
	}

	Generator struct {
		pkgs        []*Package
		files       []*File
		fileBuffers []*FileBuffer
		fileSuffix  string
		buildTags   []string
		// typeName -> JavaClassName
		typeClassMap map[string]string

		// if javaClassName not exists in typeClassMap
		// it use javaPackagePrefix.TypeName as javaClassName
		javaPackagePrefix string
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
	var generator Generator = Generator{
		buildTags:         tags,
		typeClassMap:      make(map[string]string),
		javaPackagePrefix: strings.Trim(javaPackagePrefix, PackageSeparator),
	}

	log.Println("starting scanning packages")
	generator.scanPackages(packageLocations...)
	log.Println("starting parse types")
	generator.parseType()
	log.Println("all enum classes is generated.")
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
			EnumTypes:  make(map[string][]*Value),
		}
	}

	g.pkgs = append(g.pkgs, pkg)
	g.files = append(g.files, pkg.files...)

}

func (g *Generator) parseType() {

	for _, file := range g.files {

		// get all decls
	DeclLoop:
		for _, decl := range file.syntaxTree.Decls {
			genDecl, ok := decl.(*ast.GenDecl)

			if !ok {
				continue
			}

			if genDecl.Tok == token.CONST {
				// const block

				specs := genDecl.Specs
				if specs == nil || len(specs) == 0 {
					continue
				}
				curTypeName := ""
				curTypeValues := make([]*Value, 0, 4)
				for _, spec := range specs {

					valueSpec, _ := spec.(*ast.ValueSpec) // all specs in const block is ValueSpec

					typ, err := getAndCheckTypeName(valueSpec, EmptyString)
					if err != nil {
						continue DeclLoop
					}

					if typ != "" {
						if _, ok := file.EnumTypes[typ]; !ok {
							file.EnumTypes[typ] = curTypeValues
							curTypeName = typ
						} else if curTypeName != typ {
							log.Fatalln("multi type in one const block is not allowed.")
						}
					} else {
						if curTypeName == "" {
							continue DeclLoop
						} else {
							typ = curTypeName
						}
					}

					// get all Name
					for _, name := range valueSpec.Names {
						if name.Name == UnderScoreName {
							continue
						}

						basicType, value, u64 := g.ParseConstName(name, typ, file)

						val := &Value{
							fullName: name.Name,
							name:     name.Name[len(typ):],
							value:    u64,
							signed:   basicType&types.IsUnsigned == 0,
							repl:     value.String(),
						}

						file.EnumTypes[curTypeName] = append(file.EnumTypes[curTypeName], val)
					}

				}

			} else if genDecl.Tok == token.TYPE {
				if len(genDecl.Specs) != 1 {
					continue
				}
				typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec)
				if !ok {
					continue
				}

				if _, err, ok := checkHessianJavaEnumType(ok, typeSpec, ""); err != nil || !ok {
					// not javaEnum
					continue
				}

				// get javaClassName by //go:hessian comment
				g.ParseComment(genDecl, typeSpec)

			}
		}
	}
}

func (g *Generator) ParseConstName(name *ast.Ident, typ string, file *File) (types.BasicInfo, constant.Value, uint64) {
	// check Name is with prefix or not
	if name.Name == typ || strings.Index(name.Name, typ) != 0 {
		log.Fatalf("the field %s is not with a type name prefix\"%s\"", name.Name, typ)
	}

	if name.Name[len(typ):] == UnderScoreName {
		log.Fatalf("the field %s is not allowed", name.Name)
	}

	typeObj, ok := file.pkg.defs[name]

	if !ok {
		log.Fatalf("invalid value for constant %s in TypeInfo", name)
	}

	basicType := typeObj.Type().Underlying().(*types.Basic).Info()

	if basicType&types.IsInteger == 0 {
		log.Fatalf("can't handle non-integer constant type %s", typ)
	}

	value := typeObj.(*types.Const).Val() // Guaranteed to succeed as this is CONST.
	if value.Kind() != constant.Int {
		log.Fatalf("can't happen: constant is not an integer %s", name)
	}

	i64, isInt := constant.Int64Val(value)
	u64, isUint := constant.Uint64Val(value)
	if !isInt && !isUint {
		log.Fatalf("internal error: value of %s is not an integer: %s", name, value.String())
	}

	if !isInt {
		u64 = uint64(i64)
	}
	return basicType, value, u64
}

func (g *Generator) ParseComment(genDecl *ast.GenDecl, typeSpec *ast.TypeSpec) {
	if genDecl.Doc != nil && genDecl.Doc.List != nil && len(genDecl.Doc.List) > 0 {
		for _, comment := range genDecl.Doc.List {
			if len(comment.Text) > GoHessianHeadLength && strings.Index(comment.Text, GoHessianHead) == 0 {
				commandText := strings.TrimSpace(comment.Text[GoHessianHeadLength:])
				javaClassName := parseGoHessianHeadComment(commandText)
				if javaClassName != EmptyString {
					g.typeClassMap[typeSpec.Name.Name] = javaClassName
				} else {
					if g.javaPackagePrefix == EmptyString {
						log.Fatalln("default java package not specified. if you want to use default package, use -package=. .")
					}

					strBuilder := strings.Builder{}
					strBuilder.WriteString(g.javaPackagePrefix)
					strBuilder.WriteString(PackageSeparator)
					strBuilder.WriteString(typeSpec.Name.Name)

					g.typeClassMap[typeSpec.Name.Name] = strBuilder.String()
				}
				break
			}
		}
	}
}

// TypeCheck Validation Helper
func getAndCheckTypeName(valueSpec *ast.ValueSpec, targetTypeName string) (typ string, err error) {
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
		return "", errors.New("typeName is empty when get TypeName")
	}

	if targetTypeName != "" && typ != targetTypeName {
		return "", errors.New("typeName is not the targetTypeName")
	}

	// check it's type
	if typeIdent.Obj == nil {
		return "", errors.New("typeIdent Obj is nil")
	}

	typeSpec, ok := typeIdent.Obj.Decl.(*ast.TypeSpec)
	if !ok {
		return "", errors.New("can not cast typeIdent.Obj.Decl. to *ast.TypeSpec")
	}

	s, err2, done := checkHessianJavaEnumType(ok, typeSpec, typ)
	if done {
		return s, err2
	}

	return "", errors.New("type check failed")
}

// Check the type whether is Hessian.JavaEnum
func checkHessianJavaEnumType(ok bool, typeSpec *ast.TypeSpec, typ string) (string, error, bool) {
	selectorExpr, ok := typeSpec.Type.(*ast.SelectorExpr)

	if !ok {
		return "", errors.New("can not cast typeSpec.Type. to *ast.selectorExpr"), false
	}

	if selectorExpr.Sel != nil && selectorExpr.Sel.Name == HessianJavaEnum {
		return typ, nil, true
	}
	return "", nil, false
}

// Parse GoHessian Head Comment.
// TODO(hafrans): more parameter is needed.
func parseGoHessianHeadComment(comment string) string {
	javaClassName := ""

	keyValuePairs := strings.Split(comment, SpaceSeparator)
	if len(keyValuePairs) == 0 {
		return javaClassName
	}

	for _, kv := range keyValuePairs {
		keyValuePair := strings.Split(kv, ParameterSeparator)
		if len(keyValuePair) == 1 {
			// command parameter
		} else if len(keyValuePair) == 2 {
			key := keyValuePair[0]
			val := keyValuePair[1]

			switch key {
			case "class", "c":
				javaClassName = val
			}

		}
	}

	return javaClassName
}
