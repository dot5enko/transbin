package utils

import (
	"fmt"
	"runtime/debug"
)

type errorWithStackTrace struct {
	msg string
}

func (e errorWithStackTrace) Error() string {
	return e.msg
}

func Error(format string, args ...interface{}) error {

	args = append(args, debug.Stack())

	e := errorWithStackTrace{fmt.Sprintf(format+"\n\nStack:\n%s", args...)}
	return e
}
