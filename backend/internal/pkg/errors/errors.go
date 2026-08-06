package errors

import "net/http"

// AppError carries an HTTP status alongside a message so handlers can map
// service-layer failures to responses without string-matching error text.
type AppError struct {
	Status  int
	Message string
}

func (e *AppError) Error() string { return e.Message }

func BadRequest(msg string) *AppError   { return &AppError{http.StatusBadRequest, msg} }
func Unauthorized(msg string) *AppError { return &AppError{http.StatusUnauthorized, msg} }
func Forbidden(msg string) *AppError    { return &AppError{http.StatusForbidden, msg} }
func NotFound(msg string) *AppError     { return &AppError{http.StatusNotFound, msg} }
func Conflict(msg string) *AppError     { return &AppError{http.StatusConflict, msg} }
func Internal(msg string) *AppError     { return &AppError{http.StatusInternalServerError, msg} }
