package easyftp

import "net"
import "fmt"
import "os"
// import "io"
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

// We make this class public to make extension easier, so if you are not intrest
// in this, you can ignore the details and use Client only
type Conn struct {
	Debug     bool
	conn      net.Conn
	cmdBuf    []byte
	respLine  []byte
	availData []byte

	// If this is not nil, we'ar a data connection,
	// when we're closed, we need read the result from
	// control connection
	control *Conn
}

func NewConn(conn net.Conn, debug bool) *Conn {
	c := new(Conn)
	c.Debug = debug
	c.conn = conn
	c.cmdBuf = make([]byte, maxCmdLength)
	c.respLine = make([]byte, maxRespLineLength)
	c.availData = c.respLine[:0]
	return c
}

func (c *Conn) Close() error {
	err := c.conn.Close()
	// If we're DTP, we need read the result from control connection
	if c.control != nil {
		code, msg, err2 := c.control.ReadResponse()
		if err2 != nil {
			err = err2
		}
		if code != 226 {
			err = NewUnexpectedCodeError(code, msg)
		}
	}
	return err
}

func (c *Conn) Read(b []byte) (int, error) {
	return c.conn.Read(b)
}

func (c *Conn) Write(b []byte) (int, error) {
	return c.conn.Write(b)
}

func (c *Conn) SendCommand(cmd string, msg string) error {
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

func (c *Conn) ReadResponse() (code int, msg string, err error) {
	c.availData = c.respLine[:0]
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

	c.availData = c.respLine[crnlPos+2 : received]

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
