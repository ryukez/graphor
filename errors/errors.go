package errors

import "fmt"

const (
	ConnectionRefused = iota + 1
	DropDBFailed
	MigrationFailed
	InsertionFailed
	DeletionFailed
	MutationCommitFailed
	QueryFailed
	UnmarshalizeFailed
	NoUidReturned
)

type Error interface {
	Add(key string, value string) Error
	Error() string
}

type _Error struct {
	Code    int               `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details"`
}

func New(code int, message string) Error {
	return &_Error{
		Code:    code,
		Message: message,
		Details: map[string]string{},
	}
}

func (err *_Error) Add(key string, value string) Error {
	err.Details[key] = value
	return err
}

func (err *_Error) Error() string {
	message := ""
	message += fmt.Sprintf("code: %d", err.Code)
	message += "message: " + err.Message

	for key, value := range err.Details {
		message += key + ": " + value
	}

	return message
}
