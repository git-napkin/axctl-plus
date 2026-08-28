package ipc

import "fmt"

type Error struct {
	Code    string
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("ipc: %s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("ipc: %s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error {
	return e.Err
}

var (
	ErrWindowNotFound = &Error{
		Code:    "WINDOW_NOT_FOUND",
		Message: "window not found",
	}
	ErrWorkspaceNotFound = &Error{
		Code:    "WORKSPACE_NOT_FOUND",
		Message: "workspace not found",
	}
	ErrCompositorNotAvailable = &Error{
		Code:    "COMPOSITOR_NOT_AVAILABLE",
		Message: "compositor is not available or not running",
	}
)

func NewError(code, message string, err error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Err:     err,
	}
}
