package apiutils

import (
	"errors"
	"net/http"
	"strings"
	"sync"

	jwt "github.com/dgrijalva/jwt-go"
	jwtReq "github.com/dgrijalva/jwt-go/request"
	"github.com/missionMeteora/apiserv"
)

type (
	// MapClaims is an alias for jwt.MapClaims
	MapClaims = jwt.MapClaims
	// StandardClaims is an alias for jwt.StandardClaims
	StandardClaims = jwt.StandardClaims

	// Token is an alias for jwt.Token
	Token = jwt.Token

	// KeyFunc is an alias for jwt.Keyfunc
	KeyFunc = jwt.Keyfunc

	// AuthKeyFunc is called in the SignIn middleware, should either return a signing key (usually []byte) or an error
	AuthKeyFunc = func(ctx *apiserv.Context, preSignedToken *Token) (key interface{}, err error)
)

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
	SighingMethod: jwt.SigningMethodHS256,
	Extractor:     jwtReq.AuthorizationHeaderExtractor,

	NewClaims: func() jwt.Claims { return jwt.MapClaims{} },
}

// NewAuth returns a new Auth struct with the given keyForUser and the defaults from DefaultAuth
func NewAuth(checkTokenFn KeyFunc, authKeyFunc AuthKeyFunc) (a *Auth) {
	return &Auth{
		CheckTokenFunc: checkTokenFn,
		AuthKeyFunc:    authKeyFunc,
	}
}

type Auth struct {
	SighingMethod jwt.SigningMethod
	Extractor     jwtReq.Extractor

	NewClaims func() jwt.Claims

	// CheckTokenFunc is used inside the CheckAuth middleware.
	CheckTokenFunc KeyFunc

	// AuthKeyFunc is used inside the SignIn middleware.
	AuthKeyFunc AuthKeyFunc

	io sync.Once
}

// CheckAuth handles checking auth headers.
// If the token is valid, it is set to the ctx using the TokenContextKey.
func (a *Auth) CheckAuth(ctx *apiserv.Context) apiserv.Response {
	a.init()

	tok, err := jwtReq.ParseFromRequest(ctx.Req, a.Extractor, a.CheckTokenFunc,
		jwtReq.WithClaims(a.NewClaims()))

	if err != nil {
		return apiserv.NewJSONErrorResponse(http.StatusUnauthorized, err)
	}

	ctx.Set(TokenContextKey, tok)
	return nil
}

// SignIn handles signing by calling Auth.AuthKeyFunc, if the func returns a key it signs the token and
// sets the Authorization: Bearer header.
// Can be chained with
func (a *Auth) SignIn(ctx *apiserv.Context) apiserv.Response {
	a.init()

	tok := jwt.NewWithClaims(a.SighingMethod, a.NewClaims())
	key, err := a.AuthKeyFunc(ctx, tok)
	if err != nil {
		return apiserv.NewJSONErrorResponse(http.StatusUnauthorized, err)
	}

	if _, err := a.SignInAndSetHeaders(ctx, tok, key); err != nil {
		// only reason this would return an error is if there's something wrong with json.Marshal
		return apiserv.NewJSONErrorResponse(http.StatusInternalServerError, err)
	}
	return nil
}

// SignAndSetHeaders
func (a *Auth) SignInAndSetHeaders(ctx *apiserv.Context, tok *Token, key interface{}) (signedString string, err error) {
	if signedString, err = tok.SignedString(key); err != nil {
		// only reason this would return an error is if there's something wrong with json.Marshal
		return
	}

	ctx.Set(TokenContextKey, tok)
	ctx.Header().Set("Authorization", "Bearer "+signedString)
	return
}

func (a *Auth) init() {
	a.io.Do(func() {
		if a == DefaultAuth { // sanity check, probably should panic here, but we'll play nice
			return
		}

		if a.SighingMethod == nil {
			a.SighingMethod = DefaultAuth.SighingMethod
		}

		if a.Extractor == nil {
			a.Extractor = DefaultAuth.Extractor
		}

		if a.NewClaims == nil {
			a.NewClaims = DefaultAuth.NewClaims
		}
	})
}

// ParseJWT parses a token out of a context, returns nil
func ParseJWT(ctx *apiserv.Context, claims jwt.Claims, keyFunc jwt.Keyfunc) (*jwt.Token, error) {
	h := ctx.Req.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return nil, ErrNoAuthHeader
	}

	if claims == nil {
		claims = jwt.MapClaims{}
	}

	return jwt.ParseWithClaims(strings.TrimSpace(h[7:]), claims, keyFunc)
}

// MakeJWTWithMethod is a wrapper around creating a new token and signing it
func MakeJWTWithMethod(method jwt.SigningMethod, claims jwt.Claims, key interface{}) (string, error) {
	if claims == nil {
		claims = jwt.MapClaims{}
	}
	t := jwt.NewWithClaims(method, claims)

	return t.SignedString(key)
}

// MakeJWT returns MakeJWTWithMethod(jwt.SigningMethodHS256, claims, key)
func MakeJWT(claims jwt.Claims, key []byte) (string, error) {
	return MakeJWTWithMethod(jwt.SigningMethodHS256, claims, key)
}
