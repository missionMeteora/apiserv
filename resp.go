package apiserv

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/missionMeteora/toolkit/errors"
)

// Common responses
var (
	RespNotFound   Response = NewJSONErrorResponse(http.StatusNotFound, http.StatusText(http.StatusNotFound))
	RespForbidden  Response = NewJSONErrorResponse(http.StatusForbidden, http.StatusText(http.StatusForbidden))
	RespBadRequest Response = NewJSONErrorResponse(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
	RespEmpty      Response = &JSONResponse{Code: http.StatusNoContent}
	RespRedirRoot           = Redirect("/", false)

	// Break can be returned from a handler to break a handler chain.
	// It doesn't write anything to the connection.
	// if you reassign this, a wild animal will devour your face.
	Break Response = &JSONResponse{Code: -1}
)

// Common mime-types
const (
	MimeJSON       = "application/json; charset=utf-8"
	MimeJavascript = "application/javascript; charset=utf-8"
	MimeHTML       = "text/html; charset=utf-8"
	MimePlain      = "text/plain; charset=utf-8"
	MimeBinary     = "application/octet-stream"
)

// Response represents a generic return type for http responses.
type Response interface {
	WriteToCtx(ctx *Context) error
}

// NewJSONResponse returns a new success response (code 200) with the specific data
func NewJSONResponse(data interface{}) *JSONResponse {
	return &JSONResponse{
		Code: http.StatusOK,
		Data: data,
	}
}

// ReadJSONResponse reads a response from an io.ReadCloser and closes the body.
// dataValue is the data type you're expecting, for example:
//	r, err := ReadJSONResponse(res.Body, &map[string]*Stats{})
func ReadJSONResponse(rc io.ReadCloser, dataValue interface{}) (*JSONResponse, error) {
	var r JSONResponse
	r.Data = dataValue
	if err := json.NewDecoder(rc).Decode(&r); err != nil {
		rc.Close()
		return nil, err
	}
	rc.Close()
	return &r, nil
}

// JSONResponse is the default standard api response
type JSONResponse struct {
	Code   int         `json:"code"` // if code is not set, it defaults to 200 if error is nil otherwise 400.
	Data   interface{} `json:"data,omitempty"`
	Errors []*Error    `json:"errors,omitempty"`

	Success bool `json:"success"` // automatically set to true if r.Code >= 200 && r.Code < 300.
	Indent  bool `json:"-"`       // if set to true, the json encoder will output indented json.
}

// ErrorList returns an errors.ErrorList of this response's errors or nil.
func (r *JSONResponse) ErrorList() *errors.ErrorList {
	if len(r.Errors) == 0 {
		return nil
	}
	var el errors.ErrorList
	for _, err := range r.Errors {
		el.Push(err)
	}
	return &el
}

// WriteToCtx writes the response to a ResponseWriter
func (r *JSONResponse) WriteToCtx(ctx *Context) error {
	switch r.Code {
	case 0:
		if len(r.Errors) > 0 {
			r.Code = http.StatusBadRequest
		} else {
			r.Code = http.StatusOK
		}

	case http.StatusNoContent: // special case
		ctx.WriteHeader(204)
		return nil
	}

	r.Success = r.Code >= http.StatusOK && r.Code < http.StatusMultipleChoices

	return ctx.JSON(r.Code, r.Indent, r)
}

// NewJSONErrorResponse returns a new error response.
// each err can be:
// 1. string or []byte
// 2. error
// 3. Error / *Error
// 4. another response, its Errors will be appended to the returned Response.
// 5. if errs is empty, it will call http.StatusText(code) and set that as the error.
func NewJSONErrorResponse(code int, errs ...interface{}) *JSONResponse {
	if len(errs) == 0 {
		errs = append(errs, http.StatusText(code))
	}

	var (
		r = &JSONResponse{
			Code:   code,
			Errors: make([]*Error, 0, len(errs)),
		}
	)

	for _, err := range errs {
		r.appendErr(err)
	}

	return r
}

func (r *JSONResponse) appendErr(err interface{}) {
	switch v := err.(type) {
	case Error:
		r.Errors = append(r.Errors, &v)
	case *Error:
		r.Errors = append(r.Errors, v)
	case string:
		r.Errors = append(r.Errors, &Error{Message: v})
	case []byte:
		r.Errors = append(r.Errors, &Error{Message: string(v)})
	case *JSONResponse:
		r.Errors = append(r.Errors, v.Errors...)
	case *errors.ErrorList:
		v.ForEach(func(err error) {
			r.appendErr(err)
		})
	case error:
		r.Errors = append(r.Errors, &Error{Message: v.Error()})
	default:
		log.Panicf("unsupported error type (%T): %v", v, v)
	}
}

// Error is returned in the error field of a Response.
type Error struct {
	Message   string `json:"message,omitempty"`
	Field     string `json:"field,omitempty"`
	IsMissing bool   `json:"isMissing,omitempty"`
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}

	if e.Field != "" && e.IsMissing {
		return `field "` + e.Field + `" is missing`
	}

	return fmt.Sprintf("Error{Message: %q, Field: %q, IsMissing: %v}", e.Message, e.Field, e.IsMissing)
}

// Redirect returns a redirect Response.
// if perm is false it uses http.StatusFound (302), otherwise http.StatusMovedPermanently (302)
func Redirect(url string, perm bool) Response {
	code := http.StatusFound
	if perm {
		code = http.StatusMovedPermanently
	}
	return RedirectWithCode(url, code)
}

// RedirectWithCode returns a redirect Response with the specified status code.
func RedirectWithCode(url string, code int) Response {
	return redirResp{url, code}
}

type redirResp struct {
	url  string
	code int
}

func (r redirResp) WriteToCtx(ctx *Context) error {
	if r.url == "" {
		return ErrInvalidURL
	}
	http.Redirect(ctx, ctx.Req, r.url, r.code)
	return nil
}

// File returns a file response.
// example: return File("plain/html", "index.html")
func File(contentType, fp string) Response {
	return fileResp{contentType, fp}
}

type fileResp struct {
	ct string
	fp string
}

func (f fileResp) WriteToCtx(ctx *Context) error {
	if f.ct != "" {
		ctx.SetContentType(f.ct)
	}
	return ctx.File(f.fp)
}

// PlainResponse returns a plain text response.
func PlainResponse(contentType string, v interface{}) Response {
	return plainResp{contentType, v}
}

type plainResp struct {
	ct string
	v  interface{}
}

func (r plainResp) WriteToCtx(ctx *Context) error {
	if r.ct != "" {
		ctx.SetContentType(r.ct)
	}
	var err error
	switch v := r.v.(type) {
	case []byte:
		_, err = ctx.Write(v)
	case string:
		_, err = io.WriteString(ctx, v)
	case fmt.Stringer:
		_, err = io.WriteString(ctx, v.String())
	default:
		_, err = fmt.Fprintf(ctx, "%v", r.v)
	}
	return err
}
