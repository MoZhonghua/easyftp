package easyftp

import "net"
import "fmt"
import "io"
import "strings"
import "strconv"

// This is a simple FTP client
// NOTE: this client is designed for single-threaded usage, if you
//       want to download/upload multiple files parallelly, you should
//       create multiple clients.
// NOTE: file uploading/downloading are done in binary mode, and can't
//       be changed by Client.Type() because we reset mode to binary before
//       uploading/downloading.
type Client struct {
	// Ftp connection, we make this visible for easier extension
	Conn   *ControlConn
	Debug  bool
	server string
}

func NewClient() *Client {
	return &Client{
		Conn:   nil,
		Debug:  false,
		server: "",
	}
}

func (c *Client) SendCommandAndGetResp(
	cmd string, data string) (code int, msg string, err error) {
	err = c.Conn.SendCommand(cmd, data)
	if err != nil {
		return
	}

	code, msg, err = c.Conn.ReadResponse()
	return
}

func (c *Client) Dial(server string, port int) error {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", server, port))
	if err != nil {
		return err
	}
	c.Conn = NewControlConn(conn, c.Debug)
	c.server = server

	_, _, err = c.Conn.ReadResponse()
	if err != nil {
		c.Conn.Close()
		c.server = ""
		return err
	}

	return nil
}

func (c *Client) Login(user, pass string) error {
	code, msg, err := c.SendCommandAndGetResp("USER", user)
	if err != nil {
		return err
	}
	if code != 331 {
		return NewUnexpectedCodeError(code, msg)
	}

	code, msg, err = c.SendCommandAndGetResp("PASS", pass)
	if err != nil {
		return err
	}
	if code != 230 {
		return NewUnexpectedCodeError(code, msg)
	}

	return nil
}

func (c *Client) Quit() {
	c.Conn.SendCommand("QUIT", "")
	c.Conn.ReadResponse()
	c.Conn.Close()
}

func (c *Client) Pwd() (msg string, err error) {
	// 257 "/" is the current directory.
	code, msg, err := c.SendCommandAndGetResp("PWD", "")
	if err != nil {
		return "", err
	}
	if code != 257 {
		return "", NewUnexpectedCodeError(code, msg)
	}

	return msg, err
}

func (c *Client) Mkd(dir string) (msg string, err error) {
	code, msg, err := c.SendCommandAndGetResp("MKD", dir)
	if err != nil {
		return "", err
	}
	if code != 250 {
		return "", NewUnexpectedCodeError(code, msg)
	}

	return msg, err
}

func (c *Client) Cwd(dir string) (msg string, err error) {
	// 250 "/_test" is the current directory.
	code, msg, err := c.SendCommandAndGetResp("CWD", dir)
	if err != nil {
		return "", err
	}
	if code != 250 {
		return "", NewUnexpectedCodeError(code, msg)
	}

	return msg, err
}

func (c *Client) BinayMode() error { return c.Type("I") }
func (c *Client) ASCIIMode() error { return c.Type("A") }

func (c *Client) Type(t string) error {
	// 200 Type set to: Binary.
	code, msg, err := c.SendCommandAndGetResp("TYPE", t)
	if err != nil {
		return err
	}
	if code != 200 {
		return NewUnexpectedCodeError(code, msg)
	}
	return nil
}

// RFC959: The data transfer is over the data connection in type ASCII or type
// EBCDIC. (The user must ensure that the TYPE is appropriately ASCII or
// EBCDIC).
func (c *Client) List(path string) (stream io.ReadCloser, err error) {
	err = c.ASCIIMode()
	if err != nil {
		return nil, err
	}

	port, err := c.pasvMode()
	if err != nil {
		return nil, err
	}

	// 150 File status okay. About to open data connection.
	code, msg, err := c.SendCommandAndGetResp("LIST", path)
	if err != nil {
		return nil, err
	}
	if code != 150 {
		return nil, NewUnexpectedCodeError(code, msg)
	}

	return c.newDataConn(port)
}

func (c *Client) Retr(path string) (stream io.ReadCloser, err error) {
	err = c.BinayMode()
	if err != nil {
		return nil, err
	}

	port, err := c.pasvMode()
	if err != nil {
		return nil, err
	}

	// 150 File status okay. About to open data connection.
	code, msg, err := c.SendCommandAndGetResp("RETR", path)
	if err != nil {
		return nil, err
	}
	if code != 150 {
		return nil, NewUnexpectedCodeError(code, msg)
	}

	return c.newDataConn(port)
}

func (c *Client) Stor(path string, r io.Reader) error {
	err := c.BinayMode()
	if err != nil {
		return err
	}

	// 150 File status okay. About to open data connection.
	port, err := c.pasvMode()
	if err != nil {
		return err
	}

	code, msg, err := c.SendCommandAndGetResp("STOR", path)
	if err != nil {
		return err
	}
	if code != 150 {
		return NewUnexpectedCodeError(code, msg)
	}

	conn, err := c.newDataConn(port)
	if err != nil {
		return err
	}
	io.Copy(conn, r)
	err = conn.Close()
	return err
}

func (c *Client) Dele(path string) (msg string, err error) {
	// 250 File removed.
	code, msg, err := c.SendCommandAndGetResp("DELE", path)
	if err != nil {
		return "", err
	}
	if code != 257 {
		return "", NewUnexpectedCodeError(code, msg)
	}

	return msg, err
}

func (c *Client) Size(path string) (size int64, err error) {
	// 550 SIZE not allowed in ASCII mode.
	err = c.BinayMode()
	if err != nil {
		return -1, err
	}

	// 213 170
	code, msg, err := c.SendCommandAndGetResp("SIZE", path)
	if err != nil {
		return -1, err
	}
	if code != 213 {
		return -1, NewUnexpectedCodeError(code, msg)
	}

	size, err = strconv.ParseInt(msg, 10, 63)
	return size, err
}

func (c *Client) pasvMode() (port int, err error) {
	err = c.Conn.SendCommand("PASV", "")
	if err != nil {
		return -1, err
	}

	code, msg, err := c.Conn.ReadResponse()
	if err != nil {
		return -1, err
	}

	if code != 227 {
		return -1, NewUnexpectedCodeError(code, msg)
	}

	if len(msg) == 0 {
		return -1, NewInvalidRespMsgError("PASV", "msg too short", msg)
	}

	// 227 Entering passive mode (127,0,0,1,206,177).
	start := strings.Index(msg, "(")
	end := strings.Index(msg, ")")
	if start == -1 || end == -1 {
		return -1, NewInvalidRespMsgError("PASV", "port not found", msg)
	}

	nums := strings.Split(msg[start:end], ",")
	if len(nums) != 6 {
		return -1, NewInvalidRespMsgError("PASV", "invalid port info", msg)
	}

	high, err := strconv.Atoi(nums[4])
	if err != nil {
		return -1, NewInvalidRespMsgError("PASV", "invalid port info", msg)
	}
	low, err := strconv.Atoi(nums[5])
	if err != nil {
		return -1, NewInvalidRespMsgError("PASV", "invalid port info", msg)
	}

	port = high*256 + low
	return port, nil
}

func (c *Client) newDataConn(port int) (*DataConn, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", c.server, port))
	if err != nil {
		return nil, err
	}

	dataConn := NewDataConn(c.Conn, conn)
	return dataConn, nil
}
