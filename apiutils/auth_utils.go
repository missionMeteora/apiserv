package apiutils

import (
	"net/http"

	jwtReq "github.com/golang-jwt/jwt/v4/request"
)

// CookieExtractor implements an Extractor to use auth token from cookies
type CookieExtractor []string

func (e CookieExtractor) ExtractToken(req *http.Request) (string, error) {
	for _, cookie := range e {
		if c, _ := req.Cookie(cookie); c != nil {
			return c.Value, nil
		}
	}
	return "", jwtReq.ErrNoTokenInRequest
}
