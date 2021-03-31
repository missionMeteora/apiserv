package apiserv

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/OneOfOne/otk"
	tkErrors "github.com/missionMeteora/toolkit/errors"
)

// Common responses
var (
	RespMethodNotAllowed Response = NewJSONErrorResponse(http.StatusMethodNotAllowed)
	RespNotFound         Response = NewJSONErrorResponse(http.StatusNotFound)
	RespForbidden        Response = NewJSONErrorResponse(http.StatusForbidden)
	RespBadRequest       Response = NewJSONErrorResponse(http.StatusBadRequest)
	RespOK               Response = NewJSONResponse("OK")
	RespEmpty            Response = &simpleResp{code: http.StatusNoContent}
	RespPlainOK          Response = &simpleResp{code: http.StatusOK}
	RespRedirectRoot              = Redirect("/", false)

	// Break can be returned from a handler to break a handler chain.
	// It doesn't write anything to the connection.
	// if you reassign this, a wild animal will devour your face.
	Break Response = &simpleResp{}
)

// Common mime-types
const (
	MimeJSON       = "application/json; charset=utf-8"
	MimeXML        = "application/xml; charset=utf-8"
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
func ReadJSONResponse(rc io.ReadCloser, dataValue interface{}) (r *JSONResponse, err error) {
	defer rc.Close()

	r = &JSONResponse{
		Data: dataValue,
	}

	if err = json.NewDecoder(rc).Decode(r); err != nil {
		return
	}

	if r.Success {
		return
	}

	var me MultiError
	for _, v := range r.Errors {
		me.Push(v)
	}

	if err = me.Err(); err == nil {
		err = errors.New(http.StatusText(r.Code))
	}

	return
}

func JSONRequest(method, url string, reqData, respData interface{}) (err error) {
	return otk.Request(method, "", url, reqData, func(r *http.Response) error {
		_, err := ReadJSONResponse(r.Body, respData)
		return err
	})
}

// JSONResponse is the default standard api response
type JSONResponse struct {
	Data    interface{} `json:"data,omitempty"`
	Errors  []*Error    `json:"errors,omitempty"`
	Code    int         `json:"code"`
	Success bool        `json:"success"`
	Indent  bool        `json:"-"`
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

	r.Success = r.Code >= http.StatusOK && r.Code < http.StatusBadRequest

	return ctx.JSON(r.Code, r.Indent, r)
}

func NewXMLResponse(data interface{}) *XMLResponse {
	return &XMLResponse{
		Code: http.StatusOK,
		Data: data,
	}
}

// XMLResponse is the default standard api response using xml from data
type XMLResponse struct {
	Data    interface{} `json:"data,omitempty"`
	Errors  []*Error    `json:"errors,omitempty"`
	Code    int         `json:"code"`
	Success bool        `json:"success"`
	Indent  bool        `json:"-"`
}

type xmlErrorResponse struct {
	XMLName xml.Name `xml:"errors"`
	Errors  []*Error `json:"errors,omitempty" xml:"errors,omitempty"`
}

// WriteToCtx writes the response to a ResponseWriter
func (r *XMLResponse) WriteToCtx(ctx *Context) error {
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

	r.Success = r.Code >= http.StatusOK && r.Code < http.StatusBadRequest

	ctx.SetContentType(MimeXML)
	ctx.WriteHeader(r.Code)
	ctx.Write([]byte(xml.Header))

	enc := xml.NewEncoder(ctx)
	if r.Indent {
		enc.Indent("", "\t")
	}

	if len(r.Errors) > 0 {
		er := &xmlErrorResponse{Errors: r.Errors}
		return enc.Encode(er)
	}

	return enc.Encode(r.Data)
}

// NewJSONErrorResponse returns a new error response.
// each err can be:
// 1. string or []byte
// 2. error
// 3. Error / *Error
// 4. another response, its Errors will be appended to the returned Response.
// 5. MultiError
// 6. if errs is empty, it will call http.StatusText(code) and set that as the error.
func NewJSONErrorResponse(code int, errs ...interface{}) (r *JSONResponse) {
	if len(errs) == 0 {
		errs = append(errs, http.StatusText(code))
	}

	r = &JSONResponse{
		Code:   code,
		Errors: make([]*Error, 0, len(errs)),
	}

	for _, err := range errs {
		r.appendErr(err)
	}

	return r
}

// ErrorList returns an errors.ErrorList of this response's errors or nil.
// Deprecated: handled using MultiError
func (r *JSONResponse) ErrorList() *tkErrors.ErrorList {
	if len(r.Errors) == 0 {
		return nil
	}
	var el tkErrors.ErrorList
	for _, err := range r.Errors {
		el.Push(err)
	}
	return &el
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
	case MultiError:
		for _, err := range v {
			r.appendErr(err)
		}
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
	j, _ := json.MarshalIndent(e, "", "\t")
	return string(j)
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
	v    interface{}
	ct   string
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

	r := &JSONPResponse{
		JSONResponse: JSONResponse{
			Code:   code,
			Errors: make([]*Error, 0, len(errs)),
		},
		Callback: callbackKey,
	}

	for _, err := range errs {
		r.appendErr(err)
	}

	return r
}

// JSONPResponse is the default standard api response
type JSONPResponse struct {
	Callback string `json:"-"`
	JSONResponse
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

	r.Success = r.Code >= http.StatusOK && r.Code < http.StatusBadRequest
	return ctx.JSONP(http.StatusOK, r.Callback, r)
}
