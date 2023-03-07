package file_rw

import (
	"bufio"
	"errors"
	"os"
	"strings"
)

type FileRW struct {
	bufferedWriter *bufio.Writer
	fileResource   *os.File
}

// NewFileRW instantiates new FileRW object. This instance needed only for buffered write:
//
// - PrepareBufferedWrite
// - DoBufferedWrite
// - CloseBufferedWrite
//
// All other functions can be called without object instantiation (statically)
func NewFileRW() *FileRW {
	return &FileRW{}
}

// FilePutContents - writes text string in variable "data" to file in variable "path". Path can be absolute or relative.
// If createPathIfNotExists == true, an attempt will be made to recreate the specified directory structure.
// Variable "mode" can have 2 values: "OVERWRITE" or "APPEND"
func FilePutContents(path string, data string, mode string, createPathIfNotExists bool) error {
	if err := validatePath(path); err != nil {
		return err
	}

	if f, err := createFileAtPath(path, mode, createPathIfNotExists); err != nil {
		return err
	} else if _, err := f.WriteString(data); err != nil {
		f.Close()
		return err
	} else {
		f.Close()
		return nil
	}
}

// FileReadContents - returns content of the file as a string
// path - full (if start from /) or relative path to a file
func FileReadContents(path string) (string, error) {
	if err := validatePath(path); err != nil {
		return "", err
	}

	if fileContentBytes, err := os.ReadFile(path); err != nil {
		return "", err
	} else {
		return string(fileContentBytes), nil
	}
}

func (frw *FileRW) PrepareBufferedWrite(path string, mode string, createPathIfNotExists bool) error {
	if frw.fileResource != nil && frw.bufferedWriter != nil {
		return errors.New("file resource already initialized. Close existing file resource before creating new one")
	}

	if err := validatePath(path); err != nil {
		return err
	}

	if f, err := createFileAtPath(path, mode, createPathIfNotExists); err != nil {
		return err
	} else {
		w := bufio.NewWriter(f)

		frw.fileResource = f
		frw.bufferedWriter = w
	}

	return nil
}

func (frw *FileRW) DoBufferedWrite(data string) error {
	if _, err := frw.bufferedWriter.Write([]byte(data)); err != nil {
		return err
	}

	return nil
}

func (frw *FileRW) CloseBufferedWrite() {
	frw.bufferedWriter.Flush()
	frw.fileResource.Close()

	frw.bufferedWriter = nil
	frw.fileResource = nil
}

func validatePath(path string) error {
	if strings.HasSuffix(path, "/") {
		return errors.New("full file path cannot end with \"/\", it should end with file name")
	}

	if path == "" {
		return errors.New("path cannot be empty")
	}

	return nil
}

func createFileAtPath(path string, mode string, createPathIfNotExists bool) (*os.File, error) {
	prefix := ""
	fullPath := ""

	if strings.HasPrefix(path, "/") {
		prefix = "/"
	}

	pathComponents := strings.Split(path, "/")

	if pathComponents[0] == "" {
		// If first element is empty, THIS MEAN ABSOLUTE PATH SPECIFIED
		pathComponents = pathComponents[1:]
	}

	if len(pathComponents) > 1 {
		// Not only filename specified, but directory path too
		if pathToDirectory := prefix + strings.Join(pathComponents[:len(pathComponents)-1], "/"); pathToDirectory != "" && createPathIfNotExists {
			if err := os.MkdirAll(pathToDirectory, 0755); err != nil {
				return nil, errors.New("cannot create directory by path \"" + pathToDirectory + "\"")
			}
		}

		fullPath = prefix + strings.Join(pathComponents, "/")
	} else if len(pathComponents) == 1 {
		// Only filename is specified
		fullPath = prefix + pathComponents[0]

	} else {
		// pathComponents by some reason empty - maybe path is empty too?
		return nil, errors.New("incorrect or empty path")
	}

	fileMode := 0

	if mode == "APPEND" {
		fileMode = os.O_APPEND | os.O_CREATE | os.O_WRONLY
	} else if mode == "OVERWRITE" {
		fileMode = os.O_RDWR | os.O_CREATE | os.O_TRUNC
	} else {
		return nil, errors.New("not supported mode: " + mode + ". Only APPEND and OVERWRITE are supported")
	}

	return os.OpenFile(fullPath, fileMode, 0644)
}
