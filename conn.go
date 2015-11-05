package easyftp

import "net"
import "fmt"
import "os"
import "bytes"
import "strconv"
import "errors"

const (
	maxCmdLength      int = 8096
	maxRespLineLength int = 8096
)

var (
	space = []byte{' '}
	crnl  = []byte{'\r', '\n'}
)

type DataConn struct {
	control *ControlConn
	conn    net.Conn
}

func NewDataConn(control *ControlConn, conn net.Conn) *DataConn {
	return &DataConn{
		control: control,
		conn:    conn,
	}
}

func (c *DataConn) Close() error {
	err := c.conn.Close()
	// Since we're DTP, we need read the result from control connection
	code, msg, err2 := c.control.ReadResponse()
	if err2 != nil {
		err = err2
	}
	if code != 226 {
		err = NewUnexpectedCodeError(code, msg)
	}
	return err
}

func (c *DataConn) Read(b []byte) (int, error) {
	return c.conn.Read(b)
}

func (c *DataConn) Write(b []byte) (int, error) {
	return c.conn.Write(b)
}

// We make this class public to make extension easier, so if you are not intrest
// in this, you can ignore the details and use Client only
type ControlConn struct {
	Debug     bool
	conn      net.Conn
	cmdBuf    []byte
	respLine  []byte
}

func NewControlConn(conn net.Conn, debug bool) *ControlConn {
	c := new(ControlConn)
	c.Debug = debug
	c.conn = conn
	c.cmdBuf = make([]byte, maxCmdLength)
	c.respLine = make([]byte, maxRespLineLength)
	return c
}

func (c *ControlConn) Close() error {
	err := c.conn.Close()
	return err
}

func (c *ControlConn) SendCommand(cmd string, msg string) error {
	cmdFullLen := len(cmd) + len(msg) + len(space) + len(crnl)
	if cmdFullLen > maxCmdLength {
		return errors.New("command is too long")
	}

	n := copy(c.cmdBuf, cmd)
	if len(msg) > 0 {
		n += copy(c.cmdBuf[n:], space)
		n += copy(c.cmdBuf[n:], msg)
	}
	n += copy(c.cmdBuf[n:], crnl)

	if c.Debug {
		fmt.Fprintf(os.Stderr, "%p send: %s", c, string(c.cmdBuf[:n]))
	}

	_, err := c.conn.Write(c.cmdBuf[:n])
	if err != nil {
		return err
	}

	return err
}

func (c *ControlConn) ReadResponse() (code int, msg string, err error) {
	received := 0
	crnlPos := 0
	line := c.respLine[:0]
	for {
		n, err := c.conn.Read(c.respLine[received:])
		if err != nil {
			return -1, "", err
		}
		received += n

		crnlPos = bytes.Index(c.respLine[:received], crnl)
		if crnlPos < 0 {
			if received == len(c.respLine) {
				// TODO: read until we get a crnl
				return -1, "", errors.New("response is too long")
			}
		} else {
			line = c.respLine[:crnlPos]
			break
		}
	}

	if c.Debug {
		fmt.Fprintf(os.Stderr, "%p recv: %s\n", c, string(line))
	}

	if len(line) < 3 {
		return -1, "", errors.New("response is too short")
	}

	code, err = strconv.Atoi(string(line[:3]))
	if err != nil {
		return -1, "", err
	}

	if len(line) >= 4 {
		msg = string(line[4:])
	}
	return
}
