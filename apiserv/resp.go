package apiserv

import (
	"encoding/json"
	"net/http"
)

// Common responses
var (
	RespNotFound   = NewErrorResponse(http.StatusNotFound, "-")
	RespForbidden  = NewErrorResponse(http.StatusForbidden, "-")
	RespBadRequest = NewErrorResponse(http.StatusBadRequest, "-")

	MimeJSON  = `application/json; charset=utf-8`
	MimeHTML  = `text/html; charset=utf-8`
	MimePlain = `text/plain; charset=utf-8`
)

// NewResponse returns a new success response (code 200) with the specific data
func NewResponse(data interface{}) *Response {
	return &Response{
		Code: 200,
		Data: data,
	}
}

// Response is the default standard api response
type Response struct {
	Code  int            `json:"code"`
	Data  interface{}    `json:"data,omitempty"`
	Error *ErrorResponse `json:"error,omitempty"`
}

// Output writes the response to a ResponseWriter
func (r *Response) Output(rw http.ResponseWriter) error {
	h := rw.Header()
	h.Set("Content-Type", MimeJSON)
	h.Set("X-Content-Type-Options", "nosniff") // fixes IE xss exploit

	rw.WriteHeader(r.Code)
	return json.NewEncoder(rw).Encode(r)
}

// NewErrorResponse returns a new error response
// passing "-" to msg makes it use the default status message (http.StatusText)
func NewErrorResponse(code int, msg string, fields ...FieldError) *Response {
	if msg == "-" {
		msg = http.StatusText(code)
	}
	return &Response{
		Code: code,
		Error: &ErrorResponse{
			Message: msg,
			Fields:  fields,
		},
	}
}

// ErrorResponse is returned in the error field of a Response
type ErrorResponse struct {
	Message string       `json:"message,omitempty"`
	Fields  []FieldError `json:"fields,omitempty"`
}

// FieldError is a per-field error message
type FieldError struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}
