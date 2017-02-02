package router

type Options struct {
	NoAutoCleanURL           bool
	NoDefaultNotHandler      bool
	NoDefaultPanicHandler    bool
	NoPanicOnInvalidAddRoute bool
	CatchPanics              bool
	MaxParamsPoolSize        int
}

// DefaultOptions are the options used when you pass nil to the router
var DefaultOptions = Options{
	MaxParamsPoolSize: 1000,
	CatchPanics:       true,
}
