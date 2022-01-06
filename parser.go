package main

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/packages"
	"log"
	"strings"
)

type Parser struct {
	pkgs          []*Package
	originalFiles []*File
	targetFiles   map[*File]struct{}
	fileSuffix    string
	buildTags     []string
	// TypeName -> JavaClassName
	typeClassMap map[string]string

	// if JavaClassName not exists in typeClassMap
	// it use javaPackagePrefix.TypeName as JavaClassName
	javaPackagePrefix string
}

// scan all packages provided by pattern
func (g *Parser) scanPackages(pattern ...string) {

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
	g.originalFiles = append(g.originalFiles, pkg.files...)

}

func (g *Parser) parseType() {

	for _, file := range g.originalFiles {

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

				// get JavaClassName by //go:hessian comment
				g.ParseComment(genDecl, typeSpec)

				// add this File into targetFile
				g.targetFiles[file] = struct{}{}
			}
		}
	}
}

func (g *Parser) ParseConstName(name *ast.Ident, typ string, file *File) (types.BasicInfo, constant.Value, uint64) {
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

func (g *Parser) ParseComment(genDecl *ast.GenDecl, typeSpec *ast.TypeSpec) {
	if genDecl.Doc != nil && genDecl.Doc.List != nil && len(genDecl.Doc.List) > 0 {
		for _, comment := range genDecl.Doc.List {
			if len(comment.Text) > GoHessianHeadLength && strings.Index(comment.Text, GoHessianHead) == 0 {
				commandText := strings.TrimSpace(comment.Text[GoHessianHeadLength:])
				javaClassName := parseGoHessianHeadComment(commandText)
				g.registerJavaClassName(javaClassName, typeSpec)
				break
			}
		}
	}
}

func (g *Parser) registerJavaClassName(javaClassName string, typeSpec *ast.TypeSpec) {
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
}
