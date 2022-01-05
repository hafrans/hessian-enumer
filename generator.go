package main

import (
	"bytes"
)

type (
	Generator struct {
		parser   *Parser
		fileList []*FileBuffer
	}

	FileBuffer struct {
		buffer   bytes.Buffer
		filePath string
		file     *File
	}
)
