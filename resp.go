package apiserv

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// Common responses
var (
	RespMethodNotAllowed Response = NewJSONErrorResponse(http.StatusMethodNotAllowed)
	RespNotFound         Response = NewJSONErrorResponse(http.StatusNotFound)
	RespForbidden        Response = NewJSONErrorResponse(http.StatusForbidden)
	RespBadRequest       Response = NewJSONErrorResponse(http.StatusBadRequest)
	RespEmpty            Response = &simpleResp{code: http.StatusNoContent}
	RespOK               Response = &simpleResp{code: http.StatusOK}
	RespRedirectRoot              = Redirect("/", false)

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
func ReadJSONResponse(rc io.ReadCloser, dataValue interface{}) (rp *JSONResponse, err error) {
	var r JSONResponse
	r.Data = dataValue
	err = json.NewDecoder(rc).Decode(&r)
	rc.Close()
	if err != nil {
		return
	}

	if !r.Success {
		errs := make([]string, 0, len(r.Errors))
		for _, v := range r.Errors {
			if v != nil {
				errs = append(errs, v.Error())
			}
		}

		if len(errs) > 0 {
			return nil, errors.New(strings.Join(errs, "\n"))
		}

		// No error provided, utilize the response status for messaging
		err = errors.New(http.StatusText(r.Code))
	}

	rp = &r
	return
}

// JSONResponse is the default standard api response
type JSONResponse struct {
	Code   int         `json:"code"` // if code is not set, it defaults to 200 if error is nil otherwise 400.
	Data   interface{} `json:"data,omitempty"`
	Errors []*Error    `json:"errors,omitempty"`

	Success bool `json:"success"` // automatically set to true if r.Code >= 200 && r.Code < 300.
	Indent  bool `json:"-"`       // if set to true, the json encoder will output indented json.
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
		ctx.WriteHeader(http.StatusNoContent)
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

// PlainResponse returns SimpleResponse(200, contentType, val).
func PlainResponse(contentType string, val interface{}) Response {
	return SimpleResponse(http.StatusOK, contentType, val)
}

// SimpleResponse is a QoL wrapper to return a response with the specified code and content-type.
// val can be: nil, []byte, string, io.Writer, anything else will be written with fmt.Printf("%v").
func SimpleResponse(code int, contentType string, val interface{}) Response {
	return &simpleResp{
		ct:   contentType,
		v:    val,
		code: code,
	}
}

type simpleResp struct {
	ct   string
	v    interface{}
	code int
}

func (r *simpleResp) WriteToCtx(ctx *Context) error {
	if r.ct != "" {
		ctx.SetContentType(r.ct)
	}

	if r.code > 0 {
		ctx.WriteHeader(r.code)
	}

	var err error
	switch v := r.v.(type) {
	case nil:
	case []byte:
		_, err = ctx.Write(v)
	case string:
		_, err = io.WriteString(ctx, v)
	case io.Reader:
		_, err = io.Copy(ctx, v)
	default:
		_, err = fmt.Fprintf(ctx, "%v", r.v)
	}
	return err
}

// NewJSONPResponse returns a new success response (code 200) with the specific data
func NewJSONPResponse(callbackKey string, data interface{}) *JSONPResponse {
	return &JSONPResponse{
		Callback: callbackKey,
		JSONResponse: JSONResponse{
			Code: http.StatusOK,
			Data: data,
		},
	}
}

// NewJSONPErrorResponse returns a new error response.
// each err can be:
// 1. string or []byte
// 2. error
// 3. Error / *Error
// 4. another response, its Errors will be appended to the returned Response.
// 5. if errs is empty, it will call http.StatusText(code) and set that as the error.
func NewJSONPErrorResponse(callbackKey string, code int, errs ...interface{}) *JSONPResponse {
	if len(errs) == 0 {
		errs = append(errs, http.StatusText(code))
	}

	if len(callbackKey) == 0 {
		callbackKey = "console.error"
	}

	var (
		r = &JSONPResponse{
			JSONResponse: JSONResponse{
				Code:   code,
				Errors: make([]*Error, 0, len(errs)),
			},
			Callback: callbackKey,
		}
	)

	for _, err := range errs {
		r.appendErr(err)
	}

	return r
}

// JSONPResponse is the default standard api response
type JSONPResponse struct {
	JSONResponse
	Callback string `json:"-"`
}

// WriteToCtx writes the response to a ResponseWriter
func (r *JSONPResponse) WriteToCtx(ctx *Context) error {
	switch r.Code {
	case 0:
		r.Code = http.StatusOK

	case http.StatusNoContent: // special case
		ctx.WriteHeader(http.StatusNoContent)
		return nil
	}

	r.Success = r.Code >= http.StatusOK && r.Code < http.StatusMultipleChoices
	return ctx.JSONP(http.StatusOK, r.Callback, r)
}
