package main

import (
	"fmt"
	"github.com/anxp/file_rw"
)

func main() {

	// Create directory structure related to executable and file inside:
	err := file_rw.FilePutContents("related/to/executable/APPENDABLE_FILE.TXT", "This text will append to a file on each run\n", "APPEND", true)
	fmt.Println(err)

	// Create a file in the same directory where executable are:
	err = file_rw.FilePutContents("OVERWRITABLE_FILE.TXT", "This text will always overwrite previous one\n", "OVERWRITE", false)
	fmt.Println(err)

	// Try to put a file to not-existing directory structure:
	err = file_rw.FilePutContents("related/but/not/existing/path/HELLO_WORLD.TXT", "HELLO WORLD\n", "APPEND", false)
	fmt.Println(err)

	// Read from file located in the same directory with executable:
	txt, err := file_rw.FileReadContents("OVERWRITABLE_FILE.TXT")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(txt)
	}

	// BUFFERED WRITE EXAMPLE:
	writer, err := file_rw.NewBufferedWriter("BUFFERED.TXT", "OVERWRITE", false)

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
}
