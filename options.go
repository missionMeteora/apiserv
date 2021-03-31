package apiserv

import (
	"log"
	"time"

	"github.com/missionMeteora/apiserv/router"
)

// Options allows finer control over the apiserv
type Options struct {
	Logger          *log.Logger
	RouterOptions   *router.Options
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	KeepAlivePeriod time.Duration
	MaxHeaderBytes  int
}

// Option is a func to set internal server Options.
type Option interface {
	apply(opt *Options)
}

type optionSetter func(opt *Options)

func (os optionSetter) apply(opt *Options) {
	os(opt)
}

// ReadTimeout sets the read timeout on the server.
// see http.Server.ReadTimeout
func ReadTimeout(v time.Duration) Option {
	return optionSetter(func(opt *Options) {
		opt.ReadTimeout = v
	})
}

// WriteTimeout sets the write timeout on the server.
// see http.Server.WriteTimeout
func WriteTimeout(v time.Duration) Option {
	return optionSetter(func(opt *Options) {
		opt.WriteTimeout = v
	})
}

// MaxHeaderBytes sets the max size of headers on the server.
// see http.Server.MaxHeaderBytes
func MaxHeaderBytes(v int) Option {
	return optionSetter(func(opt *Options) {
		opt.MaxHeaderBytes = v
	})
}

// SetErrLogger sets the error logger on the server.
func SetErrLogger(v *log.Logger) Option {
	return optionSetter(func(opt *Options) {
		opt.Logger = v
	})
}

// SetKeepAlivePeriod sets the underlying socket's keepalive period,
// set to -1 to disable socket keepalive.
// Not to be confused with http keep-alives which is controlled by apiserv.SetKeepAlivesEnabled.
func SetKeepAlivePeriod(p time.Duration) Option {
	return optionSetter(func(opt *Options) {
		opt.KeepAlivePeriod = p
	})
}

// SetRouterOptions sets apiserv/router.Options on the server.
func SetRouterOptions(v *router.Options) Option {
	return optionSetter(func(opt *Options) {
		opt.RouterOptions = v
	})
}

// SetNoCatchPanics toggles catching panics in handlers.
func SetNoCatchPanics(enable bool) Option {
	return optionSetter(func(opt *Options) {
		if opt.RouterOptions == nil {
			opt.RouterOptions = &router.Options{}
		}
		opt.RouterOptions.NoCatchPanics = enable
	})
}
