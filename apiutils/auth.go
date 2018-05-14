package apiutils

import (
	"errors"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	jwtReq "github.com/dgrijalva/jwt-go/request"
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
	Extractor:     jwtReq.AuthorizationHeaderExtractor,

	NewClaims: func() jwt.Claims { return jwt.MapClaims{} },
}

// NewAuth returns a new Auth struct with the given keyForUser and the defaults from DefaultAuth
func NewAuth(checkTokenFn TokenKeyFunc, authKeyFunc TokenKeyFunc) (a *Auth) {
	return &Auth{
		CheckToken: checkTokenFn,
		AuthToken:  authKeyFunc,

		SigningMethod: DefaultAuth.SigningMethod,
		Extractor:     DefaultAuth.Extractor,
		NewClaims:     DefaultAuth.NewClaims,
	}
}

// Auth is a simple handler for authorization using JWT with a simple
type Auth struct {
	SigningMethod jwt.SigningMethod
	Extractor     jwtReq.Extractor

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
		extra, key, err = a.CheckToken(ctx, Token{tok})
		return
	}, jwtReq.WithClaims(a.NewClaims()))

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
	extra, key, err := a.AuthToken(ctx, Token{tok})
	if err != nil {
		return apiserv.NewJSONErrorResponse(http.StatusUnauthorized, err)
	}

	signed, err := a.signAndSetHeaders(ctx, Token{tok}, key)
	if err != nil {
		// only reason this would return an error is if there's something wrong with json.Marshal
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
		// only reason this would return an error is if there's something wrong with json.Marshal
		return
	}

	ctx.Set(TokenContextKey, tok)
	return
}
