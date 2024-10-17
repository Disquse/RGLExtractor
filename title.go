package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
)

var (
	errInvalidMagic   = errors.New("title: invalid file magic")
	errUnknownVersion = errors.New("title: unknown version")
	errSizeMismatch   = errors.New("title: buffer size mismatch")
)

type rglTitle struct {
	Name    string
	Magic   []byte
	Version uint32
	Length  uint32
	Data    []byte
}

func ReadTitleFromFile(filePath string) (*rglTitle, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	title, err := ReadTitleFromBuffer(content)
	if err != nil {
		return nil, err
	}

	// Get title name from file path
	title.Name = getTitleFileName(filePath)

	return title, nil
}

func ReadTitleFromBuffer(content []byte) (*rglTitle, error) {
	buffer := bytes.NewBuffer(content)
	reader := NewReader(buffer)

	magic := make([]byte, 4)
	_, err := reader.Read(magic)
	if err != nil {
		return nil, err
	}

	if string(magic) != "RGLM" {
		return nil, errInvalidMagic
	}

	version, err := reader.ReadUint32()
	if err != nil {
		return nil, err
	}

	length, err := reader.ReadUint32()
	if err != nil {
		return nil, err
	}

	if version != 1 || length > uint32(len(content)) {
		return nil, errUnknownVersion
	}

	// Hardcoded offset?
	reader.SetOffset(0x50)

	data := make([]byte, length)
	size, err := reader.Read(data)
	if err != nil {
		return nil, err
	}

	if size != int(length) {
		return nil, errSizeMismatch
	}

	return &rglTitle{
		Name:    "",
		Magic:   magic,
		Version: version,
		Length:  length,
		Data:    data,
	}, nil
}

func getTitleFileName(path string) string {
	var parts []string
	for {
		dir, file := filepath.Split(path)
		if file != "" {
			parts = append([]string{file}, parts...)
		}
		if dir == "" || dir == path {
			break
		}
		path = filepath.Clean(dir)
	}

	if len(parts) < 2 {
		return ""
	}

	return parts[len(parts)-2]
}

func (title *rglTitle) decrypt() string {
	// Key and IV are both empty
	key := make([]byte, 32)
	iv := make([]byte, 16)

	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}

	buffer := make([]byte, title.Length)
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(buffer, title.Data)
	content := string(buffer)

	// FIXME: hacky trimming
	for i := 0; i < len(content); i += 1 {
		if content[i] == '{' {
			content = content[i : len(content)-1]
			break
		}
	}
	for i := len(content) - 1; i > 0; i -= 1 {
		if content[i] == '}' {
			content = content[:i+1]
			break
		}
	}

	return content
}
