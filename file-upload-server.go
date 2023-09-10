package main

import (
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"regexp"
)

var baseDirectory = "./upload"
var filePermissions fs.FileMode = 0640
var pathPermissions fs.FileMode = 0750
var bufferSize int = 4096
var port = 8080

// TODO create AUTH tokens for accessing the POST method (file upload); they should be single-use and expire very quickly
func main() {
	http.HandleFunc("/", readHandler)

	log.Printf("Listening on port %d\n", port)
	var error = http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if error != nil {
		fmt.Fprintln(os.Stderr, error)
	}
}

// Create a random string (with all lowercase characters) with the given length.
func createRandomString(length int) string {
	var bytes = make([]byte, length)
	rand.Read(bytes)

	// ensure that every byte represents a lowercase character
	for i, byte := range bytes {
		bytes[i] = (byte % 26) + 97
	}

	return string(bytes)
}

// Create a directory in the base directory with the given name, and a file called "data" inside it.
// The return values will be the full name of the created data file, a file object for it, and any error along the way.
func makeFile(name string) (string, *os.File, error) {
	if len(name) == 0 {
		name = createRandomString(16)
	}

	var path = fmt.Sprintf("%s/%s", baseDirectory, name)
	var mkdirError = os.Mkdir(path, pathPermissions)
	if mkdirError != nil {
		return "", nil, mkdirError
	}

	var filename = fmt.Sprintf("%s/data", path)
	var file, openError = os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_EXCL, filePermissions)

	return filename, file, openError
}

// Handler for file uploads via HTTP POST operations.
// The uploaded file will be stored in "{baseDirectory}/{NAME}/data", where "{NAME}" corresponds
// to the path specified by the HTTP request.
// If an empty path is specified, a random "{NAME}" will be generated.
func readHandler(writer http.ResponseWriter, request *http.Request) {
	var urlRegex = regexp.MustCompile("^/?([a-zA-Z0-9-_.,]*)/?$")
	var matches = urlRegex.FindStringSubmatch(request.URL.Path)

	if request.Method != "POST" {
		// we only allow POST requests
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	} else if matches == nil || len(matches) < 1 {
		// only a very limited subset of possible file names is allowed
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	var buffer = make([]byte, bufferSize)
	var hash, hashName = md5.New(), "md5"
	var filename, file, error = makeFile(matches[1])
	defer file.Close()

	// if we can't open the file for some reason, we delete it
	if error != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		log.Printf("Error while creating upload: %s", error)
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
	error = os.WriteFile(checksumFilename, []byte(hashResult+"\n"), filePermissions)
	if error != nil {
		log.Printf("Error while writing checksum to %s: %s", checksumFilename, error)
		return
	}
}

// Write the data bytes to the specified file.
// If an error occurs, return it.
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
