package apiserv

import (
	"log"
	"time"

	"github.com/missionMeteora/apiserv/router"
)

// Options are options used in creating the server
type options struct {
	ReadTimeout    time.Duration // see http.Server.ReadTimeout
	WriteTimeout   time.Duration // see http.Server.WriteTimeout
	MaxHeaderBytes int           // see http.Server.MaxHeaderBytes
	Logger         *log.Logger

	RouterOptions *router.Options // Additional options passed to the internal router.Router instance
}

// OptionCallback is a func to set internal server options.
type OptionCallback func(opt *options)

// ReadTimeout sets the read timeout on the server.
// see http.Server.ReadTimeout
func ReadTimeout(v time.Duration) OptionCallback {
	return func(opt *options) {
		opt.ReadTimeout = v
	}
}

// WriteTimeout sets the write timeout on the server.
// see http.Server.WriteTimeout
func WriteTimeout(v time.Duration) OptionCallback {
	return func(opt *options) {
		opt.WriteTimeout = v
	}
}

// MaxHeaderBytes sets the max size of headers on the server.
// see http.Server.MaxHeaderBytes
func MaxHeaderBytes(v int) OptionCallback {
	return func(opt *options) {
		opt.MaxHeaderBytes = v
	}
}

// SetErrLogger sets the error logger on the server.
func SetErrLogger(v *log.Logger) OptionCallback {
	return func(opt *options) {
		opt.Logger = v
	}
}

// SetRouterOptions sets apiserv/router.Options on the server.
func SetRouterOptions(v *router.Options) OptionCallback {
	return func(opt *options) {
		opt.RouterOptions = v
	}
}
