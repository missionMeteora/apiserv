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
	SigningMethod: jwt.SigningMethodHS256,
	Extractor:     jwtReq.AuthorizationHeaderExtractor,

	NewClaims: func() jwt.Claims { return jwt.MapClaims{} },
}

// NewAuth returns a new Auth struct with the given keyForUser and the defaults from DefaultAuth
func NewAuth(checkTokenFn KeyFunc, authKeyFunc AuthKeyFunc) (a *Auth) {
	return &Auth{
		CheckTokenFunc: checkTokenFn,
		AuthKeyFunc:    authKeyFunc,

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

	// CheckTokenFunc is used inside the CheckAuth middleware.
	CheckTokenFunc KeyFunc

	// AuthKeyFunc is used inside the SignIn middleware.
	AuthKeyFunc AuthKeyFunc
}

// CheckAuth handles checking auth headers.
// If the token is valid, it is set to the ctx using the TokenContextKey.
func (a *Auth) CheckAuth(ctx *apiserv.Context) apiserv.Response {
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
// Can be chained with SignUp if needed.
func (a *Auth) SignIn(ctx *apiserv.Context) apiserv.Response {
	tok := jwt.NewWithClaims(a.SigningMethod, a.NewClaims())
	key, err := a.AuthKeyFunc(ctx, tok)
	if err != nil {
		return apiserv.NewJSONErrorResponse(http.StatusUnauthorized, err)
	}

	if _, err := a.signAndSetHeaders(ctx, tok, key); err != nil {
		// only reason this would return an error is if there's something wrong with json.Marshal
		return apiserv.NewJSONErrorResponse(http.StatusInternalServerError, err)
	}
	return nil
}

func (a *Auth) signAndSetHeaders(ctx *apiserv.Context, tok *Token, key interface{}) (signedString string, err error) {
	if signedString, err = tok.SignedString(key); err != nil {
		// only reason this would return an error is if there's something wrong with json.Marshal
		return
	}

	ctx.Set(TokenContextKey, tok)
	ctx.Header().Set("Authorization", "Bearer "+signedString)
	return
}
