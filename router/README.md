# api/router
## an extremely fast, 0-garbage http router.

### Benchmarks

	➜ go test -bench=. -cpu 8 -tags httprouter
	PASS
	BenchmarkHttpRouter5Params-8    10000000               162 ns/op             160 B/op          1 allocs/op
	BenchmarkHttpRouterStatic-8     50000000              24.1 ns/op               0 B/op          0 allocs/op
	BenchmarkJumpRouter5Params-8     5000000               376 ns/op               0 B/op          0 allocs/op
	BenchmarkJumpRouterStatic-8     20000000              62.0 ns/op               0 B/op          0 allocs/op

### TODO

* Add fasthttp benchmarks and tests.
