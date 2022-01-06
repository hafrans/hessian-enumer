package main

import (
	"bytes"
	"log"
	"path"
	"strings"
	"text/template"
)

const TemplateFile = "./default.tpl"
const TemplateName = "default"


type (

	Generator struct {
		parser   *Parser
		fileList []*FileBuffer
	}

	FileBuffer struct {
		buffer        bytes.Buffer
		filePath      string
		generated     bool
		File          *File
		JavaClassName string
		TypeName      string
		TypeValues    []*Value
	}
)

func NewFileBuffers(file *File, parser *Parser) []*FileBuffer {
	if file == nil {
		return nil
	}
	enumTypes := file.EnumTypes

	if enumTypes == nil || len(enumTypes) == 0 {
		return nil
	}

	fileBuffers := make([]*FileBuffer, 0, len(enumTypes))

	for typeName, typeValues := range enumTypes {
		fileBuffers = append(fileBuffers, &FileBuffer{
			buffer:        bytes.Buffer{},
			TypeName:      typeName,
			File:          file,
			filePath:      generateTargetFilePath(typeName, file),
			JavaClassName: parser.typeClassMap[typeName],
			TypeValues:    typeValues,
		})
	}

	return fileBuffers
}

func (fb *FileBuffer) generateFileBuffer() bytes.Buffer {
	if fb.generated {
		return fb.buffer
	}

	// read template

	template.New(TemplateName)



	return fb.buffer
}

func generateTargetFilePath(typeName string, file *File) string {
	if typeName == EmptyString {
		log.Fatalln("type name is empty when generate target File path!")
	}
	if file == nil {
		log.Fatalln("File ptr is nil when generate target File path")
	}
	return path.Dir(file.filePath) + DirectorySeparator +
		strings.ToLower(typeName) + GoFileExtension
}


