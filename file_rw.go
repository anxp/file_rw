package file_rw

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
)

var ErrFileEmpty = errors.New("file empty")

type FileRW struct {
	bufferedWriter *bufio.Writer
	fileResource   *os.File
}

type WMode int8

const (
	WMODE_APPEND WMode = iota + 1
	WMODE_OVERWRITE
)

// NewBufferedWriter instantiates new FileRW object.
// THIS INSTANCE NEEDED ONLY FOR BUFFERED WRITE!
// Parameters:
//
//	path - absolute (starts from /) or relative path to the file to be written to
//	mode - can have 2 values: "OVERWRITE" or "APPEND"
//	createPathIfNotExists - if true, an attempt will be made to create all intermediate folders
//
// Usage:
//
//		NewBufferedWriter() - Create new writer
//		for {
//	    	DoBufferedWrite() - Write to file. Usually due to big amount of data this is done in a loop
//		}
//		CloseBufferedWrite() - Don't forget to close writes! Actually this flushes buffer and closes pointer to file
//
// (All other functions can be called without object instantiation (static call))
func NewBufferedWriter(path string, mode WMode, createPathIfNotExists bool) (*FileRW, error) {
	if err, _ := validateFilePath(path, false); err != nil {
		return &FileRW{}, err
	}

	if f, err := createFileAtPath(path, mode, createPathIfNotExists); err != nil {
		return &FileRW{}, err
	} else {
		w := bufio.NewWriter(f)
		return &FileRW{bufferedWriter: w, fileResource: f}, nil
	}
}

// FileWriteText - writes text string in variable "data" to file in variable "path". Path can be absolute or relative.
// If createPathIfNotExists == true, an attempt will be made to recreate the specified directory structure.
func FileWriteText(path string, data string, mode WMode, createPathIfNotExists bool) error {
	if err, _ := validateFilePath(path, false); err != nil {
		return err
	}

	if f, err := createFileAtPath(path, mode, createPathIfNotExists); err != nil {
		return err
	} else if _, err = f.WriteString(data); err != nil {
		f.Close()
		return err
	} else {
		f.Close()
		return nil
	}
}

// FileReadText - returns content of the file as a string
// path - full (if start from /) or relative path to a file
func FileReadText(path string) (string, error) {
	if err, _ := validateFilePath(path, true); err != nil {
		return "", err
	}

	if fileContentBytes, err := os.ReadFile(path); err != nil {
		return "", err
	} else {
		return string(fileContentBytes), nil
	}
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

// FastLoadTxtFile intended for loading huge files.
// It loads file in several threads from disk and parse it in a slice of strings (\n considered as line endings), effectively allocating memory.
// This function can return different errors, but there are two special errors which can be useful in some cases:
//
//	os.ErrNotExist
//	file_rw.ErrFileEmpty
//
// for example, if file does not exist or is empty, this is not a reason to interrupt program execution, we can generate data and create/fill the file,
// but if there is another error, like problem with permissions or syntax error in path, we really have a problem.
// Use errors.Is(err, os.ErrNotExist) or errors.Is(err, file_rw.ErrFileEmpty) for convenient check for special error.
func FastLoadTxtFile(path string, allowEmptyLines bool, returnErrorOnEmptyFile bool) ([]string, error) {
	rawDataPointer, err := MultithreadedRead(path)

	if err != nil {
		return []string{}, err
	}

	lines, err := splitToLines(rawDataPointer, allowEmptyLines)

	if err != nil {
		return []string{}, err
	}

	if returnErrorOnEmptyFile && len(lines) == 0 {
		return []string{}, ErrFileEmpty
	}

	return lines, nil
}

func MultithreadedRead(path string) (*[]byte, error) {
	var f *os.File
	var err error
	var fSize int64
	var numberOfThreads int

	type filePart struct {
		partNumber       int
		startReadingByte int64
		content          []byte
		lengthRequested  int64
		lengthRead       int64
		error            error
	}

	if err, fSize = validateFilePath(path, true); err != nil {
		return nil, err
	}

	if f, err = os.OpenFile(path, os.O_RDONLY, 0); err != nil {
		return nil, err
	}

	defer f.Close()

	if fSize <= 1048576 { // 1MB
		numberOfThreads = 1
	} else if fSize <= 134217728 { // 134MB
		numberOfThreads = 8
	} else {
		numberOfThreads = 16
	}

	// ======================================= MAKE A FILE READING PLAN ================================================
	chunkSize := int64(math.Ceil(float64(fSize) / float64(numberOfThreads)))
	lastChunkSize := fSize - chunkSize*(int64(numberOfThreads)-1)
	fileInChunks := make([]filePart, numberOfThreads)
	startIndex := int64(0)

	for i := 0; i < numberOfThreads; i++ {
		fileInChunks[i] = filePart{
			partNumber:       i,
			startReadingByte: startIndex,
			content:          make([]byte, chunkSize),
			lengthRequested:  chunkSize,
		}

		// Last part of file. Usually it shorter than chunkSize, but this is not a rule (for ex. 512 / 4 = 128, all 4 chunks are equal!)
		if i == numberOfThreads-1 {
			fileInChunks[i] = filePart{
				partNumber:       i,
				startReadingByte: startIndex,
				content:          make([]byte, lastChunkSize),
				lengthRequested:  lastChunkSize,
			}
		}

		startIndex = startIndex + chunkSize
	}
	// =================================================================================================================

	// ======================================= PARALLEL READING ========================================================
	dataChannel := make(chan filePart)

	readChunkFn := func(f *os.File, partToRead filePart, dataChannel chan filePart) {

		// TODO: Investigate how ReadAt can change passed value if it not declared as a pointer?
		length, err := f.ReadAt(partToRead.content, partToRead.startReadingByte)

		if err != nil && err != io.EOF {
			partToRead.error = err
		}

		partToRead.lengthRead = int64(length)

		dataChannel <- partToRead
	}

	for i := 0; i < numberOfThreads; i++ {
		go readChunkFn(f, fileInChunks[i], dataChannel)
	}

	receivedFragments := 0
	errMessage := ""

	for {
		fPart := <-dataChannel

		if fPart.error != nil {
			errMessage += fPart.error.Error() + "; "
		}

		fileInChunks[fPart.partNumber] = fPart

		receivedFragments++
		if receivedFragments == numberOfThreads {
			break
		}
	}

	close(dataChannel)

	if errMessage != "" {
		return nil, errors.New(errMessage)
	}
	// =================================================================================================================

	// ======================================= ASSEMBLY THE FILE =======================================================
	assembledFile := make([]byte, 0, fSize)

	for i := 0; i < numberOfThreads; i++ {
		assembledFile = append(assembledFile, fileInChunks[i].content...)
	}

	if int64(len(assembledFile)) != fSize {
		return nil, fmt.Errorf("file size error: expected [%d], got [%d] bytes\n", fSize, len(assembledFile))
	}
	// =================================================================================================================

	return &assembledFile, nil
}

// FileWriteBytes put bytes slice to the file - either overwriting existing file or appending to the end of it.
func FileWriteBytes(path string, data *[]byte, mode WMode, createPathIfNotExists bool) error {
	var err error
	var f *os.File

	if err, _ = validateFilePath(path, false); err != nil {
		return err
	}

	if f, err = createFileAtPath(path, mode, createPathIfNotExists); err != nil {
		return err
	}

	if _, err = f.Write(*data); err != nil {
		return err
	}

	if err = f.Close(); err != nil {
		return err
	}

	return nil
}

// FileOverwriteBytes overwrites data in file starting from specified byte.
func FileOverwriteBytes(path string, fromByte int64, replacement *[]byte) error {
	var size int64
	var err error
	var f *os.File

	if err, size = validateFilePath(path, true); err != nil {
		return err
	}

	// if fromByte == size => it works exactly as APPEND (nothing to overwrite)
	if fromByte > size {
		return errors.New("incorrect write, the gap is not allowed")
	}

	// ==================== OVERWRITE DATA =============================================================================
	if f, err = os.OpenFile(path, os.O_WRONLY, 0644); err != nil {
		return err
	}

	if _, err = f.Seek(fromByte, 0); err != nil {
		return err
	}

	if _, err = f.Write(*replacement); err != nil {
		return err
	}

	if err = f.Close(); err != nil {
		return err
	}
	// =================================================================================================================

	return nil
}

// FileInsertBytes inserts new data in file starting from specified byte. All existing data moved forward for len(insertion) bytes.
// This function is effective when writing/inserting a piece of data at the end of file, when a small amount of data is written to disk.
// When inserting at the beginning of a file, this function will not provide any benefit since it will actually overwrite (almost) the entire file.
func FileInsertBytes(path string, fromByte int64, insertion *[]byte) error {
	var size int64
	var err error
	var f *os.File
	var remainder []byte

	if err, size = validateFilePath(path, true); err != nil {
		return err
	}

	// if fromByte == size => it works exactly as APPEND (nothing to overwrite)
	if fromByte > size {
		return errors.New("incorrect write, the gap is not allowed")
	}

	// ==================== PART ONE: READ & REMEMBER SECOND PART ======================================================
	if f, err = os.OpenFile(path, os.O_RDONLY, 0644); err != nil {
		return err
	}

	if _, err = f.Seek(fromByte, 0); err != nil {
		return err
	}

	if remainder, err = io.ReadAll(f); err != nil {
		return err
	}

	if err = f.Close(); err != nil {
		return err
	}
	// =================================================================================================================

	// ==================== PART TWO: WRITE INSERTION ==================================================================
	if f, err = os.OpenFile(path, os.O_WRONLY, 0644); err != nil {
		return err
	}

	if _, err = f.Seek(fromByte, 0); err != nil {
		return err
	}

	if _, err = f.Write(*insertion); err != nil {
		return err
	}
	// =================================================================================================================

	// ==================== PART THREE: WRITE REMAINDER ================================================================
	if _, err = f.Write(remainder); err != nil {
		return err
	}

	if err = f.Close(); err != nil {
		return err
	}
	// =================================================================================================================

	return nil
}

func splitToLines(data *[]byte, allowEmptyLines bool) ([]string, error) {
	// Count EOL: https://stackoverflow.com/questions/24562942/golang-how-do-i-determine-the-number-of-lines-in-a-file-efficiently

	// First, we need to check how many lines (ending with \n) are in raw byte slice.
	// Knowing this allows us to effectively allocate memory for the returned result.
	lineSep := []byte{'\n'}
	lineCount := bytes.Count(*data, lineSep) + 1 // For the case when last line does not end with \n, but with EOF (is that possible?)

	fileLines := make([]string, 0, lineCount)

	bytesReader := bytes.NewReader(*data)
	bufReader := bufio.NewReader(bytesReader)

	for {
		line, err := bufReader.ReadString('\n')

		if err != nil && err != io.EOF {
			return []string{}, err
		}

		trimmed := strings.TrimSpace(line)

		if allowEmptyLines == true || (allowEmptyLines == false && trimmed != "") {
			fileLines = append(fileLines, trimmed)
		}

		if err == io.EOF {
			break
		}
	}

	return fileLines, nil
}

// validatePath can be used for simple or full validation.
// Simple validation:
//
//	Only syntax is checked.
//	We require path NOT TO BE EMPTY and NOT ENDS WITH "/". This mode used for files to be created (fileShouldExist = false).
//
// Full validation:
//
//	Checks syntax AND file existence.
//	This mode used for files to be read, so file should already exist at filesystem (fileShouldExist = true).
//
// Returns:
//
//	error if path not valid (or file does not exist while it should) | nil
//	file size in bytes if applicable
func validateFilePath(path string, fileShouldExist bool) (error, int64) {
	if strings.HasSuffix(path, "/") {
		return errors.New("full file path cannot end with \"/\", it should end with file name"), 0
	}

	if path == "" {
		return errors.New("path cannot be empty"), 0
	}

	if fileShouldExist {
		// "path" in os.Stat(path) can be either absolute or relative
		if stat, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			return err, 0
		} else {
			return nil, stat.Size()
		}
	}

	return nil, 0
}

func createFileAtPath(path string, mode WMode, createPathIfNotExists bool) (*os.File, error) {
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

	if mode == WMODE_APPEND {
		fileMode = os.O_APPEND | os.O_CREATE | os.O_WRONLY
	} else if mode == WMODE_OVERWRITE {
		fileMode = os.O_RDWR | os.O_CREATE | os.O_TRUNC
	} else {
		return nil, errors.New("not supported write mode")
	}

	return os.OpenFile(fullPath, fileMode, 0644)
}
