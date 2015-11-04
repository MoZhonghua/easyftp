# easyftp
A FTP client implementation in GO

# example
```go
package main

import (
	"github.com/MoZhonghua/easyftp"
	"fmt"
	"os"
	"io"
)

func dieIfError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s", err.Error())
		panic(err)
	}
}

func main() {
	ftp := easyftp.NewClient()
	ftp.Debug = true

	err := ftp.Dial("127.0.0.1", 21)
	dieIfError(err)

	// login
	err = ftp.Login("anonymous", "")
	dieIfError(err)

	// pwd
	msg, err := ftp.Pwd()
	dieIfError(err)
	fmt.Printf("PWD: %s\n", msg)

	// make dir
	msg, err = ftp.Mkd("/testdir")
	dieIfError(err)
	fmt.Printf("MKD: %s\n", msg)

	// stor file
	r, err := os.Open("local_file.txt")
	dieIfError(err)
	defer r.Close()
	err = ftp.Stor("/remote_file.txt", r)
	dieIfError(err)

	// retr file
	w, err := os.Create("local_file2.txt")
	dieIfError(err)
	defer w.Close()

	data, err := ftp.Retr("/remote_file.txt")
	dieIfError(err)
	_, err = io.Copy(w, data)
	dieIfError(err)

	// you must close this and check the return value
	err = data.Close()
	dieIfError(err)

	ftp.Quit()
}
```
