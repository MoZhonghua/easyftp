package main

import "easyftp"
import "io"
import "os"
import "fmt"

var _ = io.Copy
var _ = os.Stderr

func uploadFile(c *easyftp.Client, from string, to string) {
	r, err := os.Open(from)
	if err != nil {
		fmt.Println("failed to open upload local file:", err)
		return
	}
	defer r.Close()

	err = c.Stor(to, r)
	if err != nil {
		fmt.Println("failed to open remote file:", err)
		return
	}
}

func abortedDownloadFile(c *easyftp.Client, from string, to string) {
	w, err := os.Create(to)
	if err != nil {
		fmt.Println("failed to open download dest file:", err)
		return
	}
	defer w.Close()

	r, err := c.Retr(from)
	if err != nil {
		fmt.Println("failed to open remote file:", err)
		return
	}
	err = r.Close()
	if err != nil {
		fmt.Println("aborted file downloading:", err)
		return
	}
	// We close the connection without reading anything
	// io.Copy(w, r)
}

func downloadFile(c *easyftp.Client, from string, to string) {
	w, err := os.Create(to)
	if err != nil {
		fmt.Println("failed to open download dest file:", err)
		return
	}
	defer w.Close()

	r, err := c.Retr(from)
	if err != nil {
		fmt.Println("failed to open remote file:", err)
		return
	}
	defer r.Close()
	io.Copy(w, r)
}

func listDir(c *easyftp.Client) {
	r, err := c.List(".")
	if err != nil {
		fmt.Println("LIST: ", err)
		return
	}
	defer r.Close()
	io.Copy(os.Stdout, r)
}

func main() {
	c := easyftp.NewClient()
	c.Debug = true
	err := c.Dial("127.0.0.1", 21)
	if err != nil {
		panic(err)
	}

	defer c.Quit()

	err = c.Login("anonymous", "")
	if err != nil {
		fmt.Println(err)
		return
	}

	_, err = c.Size("/test.go")
	if err != nil {
		fmt.Println("SIZE: ", err)
	}

	_, err = c.Pwd()
	if err != nil {
		fmt.Println("PWD: ", err)
	}

	_, err = c.Mkd("_test")
	if err != nil {
		fmt.Println("MKD:", err)
	}

	_, err = c.Cwd("_test")
	if err != nil {
		fmt.Println("CWD:", err)
	}

	_, err = c.Cwd("..")
	if err != nil {
		fmt.Println("CWD:", err)
	}

	downloadFile(c, "/test.go", "1.txt")
	downloadFile(c, "/test.go", "2.txt")
	uploadFile(c, "1.txt", "/100.txt")
	abortedDownloadFile(c, "/test.go", "2.txt")

	_, err = c.Dele("/100.txt")
	if err != nil {
		fmt.Println("DELE: ", err)
	}

	_, err = c.Dele("/100.txt")
	if err != nil {
		fmt.Println("DELE: ", err)
	}

	listDir(c)
}
