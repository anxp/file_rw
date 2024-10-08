package main

import (
	"fmt"
	"github.com/anxp/file_rw"
)

func main() {

	// Create directory structure related to executable and file inside:
	err := file_rw.FileWriteText("related/to/executable/APPENDABLE_FILE.TXT", "This text will append to a file on each run\n", file_rw.WMODE_APPEND, true)
	fmt.Println(err)

	// Create a file in the same directory where executable are:
	err = file_rw.FileWriteText("OVERWRITABLE_FILE.TXT", "This text will always overwrite previous one\n", file_rw.WMODE_OVERWRITE, false)
	fmt.Println(err)

	// Try to put a file to not-existing directory structure:
	err = file_rw.FileWriteText("related/but/not/existing/path/HELLO_WORLD.TXT", "HELLO WORLD\n", file_rw.WMODE_APPEND, false)
	fmt.Println(err)

	// Read from file located in the same directory with executable:
	txt, err := file_rw.FileReadText("OVERWRITABLE_FILE.TXT")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(txt)
	}

	// ==================== BUFFERED WRITE Example: ====================================================================
	writer, err := file_rw.NewBufferedWriter("BUFFERED.TXT", file_rw.WMODE_OVERWRITE, false)

	if err != nil {
		fmt.Print(err)
	}

	writer.DoBufferedWrite("Data line 1\n")
	writer.DoBufferedWrite("Data line 2\n")
	writer.DoBufferedWrite("Data line 3\n")
	writer.DoBufferedWrite("Data line 4\n")
	writer.DoBufferedWrite("Data line 5\n")
	writer.DoBufferedWrite("Data line 6\n")

	writer.CloseBufferedWrite()
	// =================================================================================================================

	// ==================== APPEND Bytes Example: ======================================================================
	testData1 := []byte("Hello world!\n")
	testData2 := []byte("Data to append line 2\n")
	testData3 := []byte("Data to append line 3\n")
	testData4 := []byte("Data to append line 4\n")

	err = file_rw.FileWriteBytes("AppendBytesTest.txt", &testData1, file_rw.WMODE_APPEND, true)
	if err != nil {
		fmt.Println(err)
	}
	file_rw.FileWriteBytes("AppendBytesTest.txt", &testData2, file_rw.WMODE_APPEND, true)
	file_rw.FileWriteBytes("AppendBytesTest.txt", &testData3, file_rw.WMODE_APPEND, true)
	file_rw.FileWriteBytes("AppendBytesTest.txt", &testData4, file_rw.WMODE_APPEND, true)
	//==================================================================================================================

	// ==================== INSERT Bytes Example: ======================================================================
	testData := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\n"
	insertion := []byte("This piece of text\nshould be inserted\nbetween 2nd and 3rd lines\n")

	err = file_rw.FileWriteText("InsertTest.txt", testData, file_rw.WMODE_OVERWRITE, true)
	if err != nil {
		fmt.Println(err)
	}

	firstPartBytes := []byte("Line 1\nLine 2\n")
	err = file_rw.FileInsertBytes("/home/andrii/go/src/file_rw/InsertTest.txt", int64(len(firstPartBytes)), &insertion)
	if err != nil {
		fmt.Println(err)
	}
	// =================================================================================================================
}
