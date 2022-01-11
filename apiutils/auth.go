package apiutils

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	jwt "github.com/golang-jwt/jwt"
	jwtReq "github.com/golang-jwt/jwt/request"
	"github.com/missionMeteora/apiserv"
)

type (
	// MapClaims is an alias for jwt.MapClaims
	MapClaims = jwt.MapClaims
	// StandardClaims is an alias for jwt.StandardClaims
	StandardClaims = jwt.StandardClaims

	// TokenKeyFunc is a callback to return a key for the given token
	TokenKeyFunc = func(ctx *apiserv.Context, tok Token) (extra apiserv.M, key interface{}, err error)
)

type Token struct {
	*jwt.Token
}

// GetOk only works with MapClaims
func (t Token) GetOk(key string) (v interface{}, ok bool) {
	m, _ := t.Claims.(MapClaims)
	v, ok = m[key]
	return
}

// Get only works with MapClaims
func (t Token) Get(key string) interface{} {
	v, _ := t.GetOk(key)
	return v
}

func (t Token) Set(k string, v interface{}) (ok bool) {
	var m MapClaims
	if m, ok = t.Claims.(MapClaims); ok {
		m[k] = v
	}
	return
}

// SetExpiry sets the expiry date of the token, ts is time.Time{}.Unix().
func (t Token) SetExpiry(ts int64) (ok bool) {
	return t.Set("exp", float64(ts))
}

func (t Token) Expiry() (ts int64, ok bool) {
	switch v := t.Get("exp").(type) {
	case json.Number:
		i, err := v.Int64()
		return i, err == nil
	case float64:
		return int64(v), true
	default:
		return 0, false
	}
}

const (
	// TokenContextKey is the key used to access the saved token inside an apiserv.Context.
	TokenContextKey = ":JTK:"
)

// errors
var (
	ErrNoAuthHeader = errors.New("missing Authorization: Bearer header is not set")
)

// DefaultAuth has the default values for Auth
var DefaultAuth = &Auth{
	SigningMethod: jwt.SigningMethodHS256,
	Extractor:     *jwtReq.OAuth2Extractor,

	NewClaims: func() jwt.Claims { return jwt.MapClaims{} },
}

var DefaultParser = &jwt.Parser{
	UseJSONNumber: true,
}

// NewAuth returns a new Auth struct with the given keyForUser and the defaults from DefaultAuth
func NewAuth(checkTokenFn TokenKeyFunc, authKeyFunc TokenKeyFunc, extractors ...jwtReq.Extractor) (a *Auth) {
	var cookies []string
	for _, e := range extractors {
		if e, ok := e.(CookieExtractor); ok {
			cookies = append(cookies, e...)
		}
	}

	return &Auth{
		CheckToken:  checkTokenFn,
		AuthToken:   authKeyFunc,
		AuthCookies: cookies,

		SigningMethod: DefaultAuth.SigningMethod,
		Extractor:     append(extractors, DefaultAuth.Extractor...),
		NewClaims:     DefaultAuth.NewClaims,
	}
}

// Auth is a simple handler for authorization using JWT with a simple
type Auth struct {
	SigningMethod jwt.SigningMethod
	Extractor     jwtReq.MultiExtractor

	AuthCookies []string
	CookieHost  string
	CookieHTTPS bool

	NewClaims func() jwt.Claims

	// TokenKey is used inside the CheckAuth middleware.
	CheckToken TokenKeyFunc

	// AuthKeyFunc is used inside the SignIn middleware.
	AuthToken TokenKeyFunc
}

// CheckAuth handles checking auth headers.
// If the token is valid, it is set to the ctx using the TokenContextKey.
func (a *Auth) CheckAuth(ctx *apiserv.Context) apiserv.Response {
	var extra apiserv.M
	tok, err := jwtReq.ParseFromRequest(ctx.Req, a.Extractor, func(tok *jwt.Token) (key interface{}, err error) {
		extra, key, err = a.CheckToken(ctx, Token{Token: tok})
		return
	}, jwtReq.WithClaims(a.NewClaims()), jwtReq.WithParser(DefaultParser))
	if err != nil {
		return apiserv.NewJSONErrorResponse(http.StatusUnauthorized, err)
	}

	ctx.Set(TokenContextKey, tok)

	if len(extra) > 0 {
		return apiserv.NewJSONResponse(extra)
	}

	return nil
}

// SignIn handles signing by calling Auth.AuthKeyFunc, if the func returns a key it signs the token and
// sets the Authorization: Bearer header.
// Can be chained with SignUp if needed.
func (a *Auth) SignIn(ctx *apiserv.Context) apiserv.Response {
	tok := jwt.NewWithClaims(a.SigningMethod, a.NewClaims())
	extra, key, err := a.AuthToken(ctx, Token{Token: tok})
	if err != nil {
		return apiserv.NewJSONErrorResponse(http.StatusUnauthorized, err)
	}

	signed, err := a.signAndSetHeaders(ctx, Token{Token: tok}, key)
	if err != nil {
		// only reason this would return an error is if there's something wrong with sonic.Marshal
		return apiserv.NewJSONErrorResponse(http.StatusInternalServerError, err)
	}

	if extra == nil {
		extra = apiserv.M{}
	}

	extra["access_token"] = signed

	return apiserv.NewJSONResponse(extra)
}

func (a *Auth) signAndSetHeaders(ctx *apiserv.Context, tok Token, key interface{}) (signedString string, err error) {
	if signedString, err = tok.SignedString(key); err != nil {
		// only reason this would return an error is if there's something wrong with sonic.Marshal
		return
	}

	ctx.Set(TokenContextKey, tok)

	exp, ok := tok.Expiry()
	if ok && exp > 0 {
		exp = int64(time.Until(time.Unix(exp, 0)))
	}

	for _, c := range a.AuthCookies {
		if err = ctx.SetCookie(c, signedString, a.CookieHost, a.CookieHTTPS, time.Duration(exp)); err != nil {
			return
		}
	}

	return
}
