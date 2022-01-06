package main

import (
	"bytes"
	"log"
	"path"
	"strings"
)

type (
	Generator struct {
		parser   *Parser
		fileList []*FileBuffer
	}

	FileBuffer struct {
		buffer        bytes.Buffer
		filePath      string
		file          *File
		javaClassName string
		typeName      string
		typeValues    []*Value
	}
)

func NewFileBuffers(file *File, parser *Parser) []*FileBuffer{
	if file == nil {
		return nil
	}
	enumTypes := file.EnumTypes

	if enumTypes == nil || len(enumTypes) == 0 {
		return nil
	}

	fileBuffers := make([]*FileBuffer,0, len(enumTypes))

	for typeName, typeValues := range enumTypes {
		fileBuffers = append(fileBuffers, &FileBuffer{
			buffer: bytes.Buffer{},
			typeName:typeName,
			file: file,
			filePath: generateTargetFilePath(typeName, file),
		    javaClassName: parser.typeClassMap[typeName],
		    typeValues: typeValues,
		})
	}

	return fileBuffers
}

func generateTargetFilePath(typeName string, file *File) string {
	if typeName == EmptyString {
		log.Fatalln("type name is empty when generate target file path!")
	}
	if file == nil {
		log.Fatalln("file ptr is nil when generate target file path")
	}
	return  path.Dir(file.filePath) + DirectorySeparator +
		strings.ToLower(typeName) + GoFileExtension
}
