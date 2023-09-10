package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
)

var filename string = "output"
var filePermissions fs.FileMode = 0600
var bufferSize int = 4096

// TODO create AUTH tokens for accessing the POST method (file upload); they should be single-use and expire very quickly
func main() {
	var args = os.Args[1:]
	if len(args) >= 1 {
		filename = args[0]
	}

	http.HandleFunc("/", readHandler)
	var error = http.ListenAndServe(":8080", nil)
	if error != nil {
		fmt.Fprintln(os.Stderr, error)
	}
}

func getFilename(filename string) string {
	return "upload/" + filename
}

// handler for file uploads
func readHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != "POST" {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// TODO we may actually not want to override existing files... use `os.O_EXCL`?
	var filename = getFilename(filename)
	var buffer = make([]byte, bufferSize)
	var hash, hashName = md5.New(), "md5"
	var file, error = os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, filePermissions)
	defer file.Close()

	// if we can't open the file for some reason, we delete it
	if error != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		log.Printf("Error while opening file '%s': %s", filename, error)
		return
	}

	// read the entire request body chunk-wise
	for {
		var numBytes, error = request.Body.Read(buffer)
		if numBytes == 0 {
			break
		}
		if error != nil && error != io.EOF {
			writer.WriteHeader(http.StatusInternalServerError)
			log.Printf("Error while reading HTTP request body: %s", error)
			return
		}

		// write the data chunk to disk and update the hash
		hash.Write(buffer[:numBytes])
		error = writeToFile(file, buffer[:numBytes])
		if error != nil {
			log.Printf("Error while writing chunk to file '%s': %s", filename, error)
			return
		}
	}

	// finalize the hash and write results to the HTTP response as well as a file
	var hashResult = fmt.Sprintf("%x  %s", hash.Sum([]byte{}), filename)
	var checksumFilename = fmt.Sprintf("%s.%s", filename, hashName)

	fmt.Fprintln(writer, hashResult)
	error = os.WriteFile(checksumFilename, []byte(hashResult), filePermissions)
	if error != nil {
		log.Printf("Error while writing checksum to %s: %s", checksumFilename, error)
		return
	}
}

// write the data to the file
// TODO make it buffered?
func writeToFile(file *os.File, data []byte) error {
	var totalBytesWritten = 0

	for {
		if totalBytesWritten >= len(data) {
			break
		}

		var numBytes, error = file.Write(data[totalBytesWritten:])
		totalBytesWritten += numBytes

		if error != nil {
			return error
		}
	}

	return nil
}
