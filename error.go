package easyftp

import "fmt"

type UnexpectedCodeError struct {
	Code int
	Msg  string
}

func NewUnexpectedCodeError(code int, msg string) error {
	return &UnexpectedCodeError{
		Code: code,
		Msg:  msg,
	}
}

func (err *UnexpectedCodeError) Error() string {
	return fmt.Sprintf("UnexpectedCodeError: code(%d): %s", err.Code, err.Msg)
}

type InvalidRespMsgError struct {
	Cmd    string
	Reason string
	Resp   string
}

func NewInvalidRespMsgError(cmd string, reason string, resp string) error {
	return &InvalidRespMsgError{
		Cmd:    cmd,
		Reason: reason,
		Resp:   resp,
	}
}

func (err *InvalidRespMsgError) Error() string {
	return fmt.Sprintf("InvalidRespMsgError: cmd(%d): reason(%s) : %s",
		err.Cmd, err.Reason, err.Resp)
}
