# File upload server

This is a simple file upload server, intended to enable uploading of many files in parallel.


## Prerequisites

* `go 1.21.0`


## Run

The server can be started with `go run file-upload-server.go`.
It will start listening to HTTP POST requests on port 8080.


## Use

To upload a file under a randomly generated file name, you can use the the following command:
`curl -X POST --data-binary @/path/to/file http://localhost:8080`

This will create a new directory in the base directory (`./uploads/` per default) with a random name.
This directory will contain a file called `data` holding the uploaded data, and `data.md5` containing the MD5 sum of the uploaded data:
```
upload
└── mpkdgxrezjugpaxk
    ├── data
    └── data.md5
```


### Explicitly naming uploads

If you want to force a specific name for the upload, you can use:
`curl -X POST --data-binary @/path/to/file http://localhost:8080/my-name`

This will create the following structure:
```
upload
└── my-name
    ├── data
    └── data.md5
```


### Rate limiting

Note: `curl` supports a `--rate-limit` flag that can be used to limit the transmission speed.
This can be useful if you want to see the server's behavior with several parallel uploads.
