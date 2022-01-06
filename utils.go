package main

import (
	"errors"
	"go/ast"
	"log"
	"strings"
)

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

func generateCombinedNamesWithOffset(values []*Value) (string, [][2]int) {

	if values == nil || len(values) == 0 {
		return EmptyString, [][2]int{}
	}

	var names = strings.Builder{}
	var offset [][2]int = make([][2]int, 0, len(values))
	var accumulator int = 0

	for idx, value := range values {
		names.WriteString(value.name)
		nameOffset := [...]int{accumulator, accumulator + len(value.name)}
		offset[idx] = nameOffset
	}
	return names.String(), offset

}
