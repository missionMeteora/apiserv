package apiserv

import (
	"compress/gzip"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	acceptHeader   = "Accept-Encoding"
	encodingHeader = "Content-Encoding"

	brEnc = "br"
	gzEnc = "gzip"
)

func (ctx *Context) EnableGzip(level int) {
	if _, ok := ctx.ResponseWriter.(*gzRW); ok {
		return
	}
	g := getGzRW(gzip.DefaultCompression)
	g.init(ctx)
}

// TryCompressed will try serving compressed files if they exist on the disk or use on the fly gzip.
func TryCompressed(ctx *Context, fname string) error {
	gz, br := accepts(ctx.ReqHeader().Get(acceptHeader))
	ctx.SetContentType(mime.TypeByExtension(filepath.Ext(fname)))

	if br {
		if fname := fname + ".br"; fileExists(fname) {
			ctx.Header().Set(encodingHeader, brEnc)
			return ctx.File(fname)
		}
	}

	if gz {
		if fname := fname + ".gz"; fileExists(fname) {
			ctx.Header().Set(encodingHeader, gzEnc)
			return ctx.File(fname)
		}

		ctx.EnableGzip(6)
	}

	return ctx.File(fname)
}

func accepts(h string) (gz, br bool) {
	for _, s := range strings.Split(h, ",") {
		switch {
		case !gz && strings.Contains(s, "gzip"):
			gz = true
		case !br && strings.Contains(s, "br"):
			br = true
		}
	}
	return
}

func fileExists(fn string) bool {
	fi, err := os.Stat(fn)
	return err == nil && !fi.IsDir() && fi.Mode().IsRegular()
}

func Gzip(level int) Handler {
	return func(ctx *Context) Response {
		ctx.EnableGzip(level)
		return nil
	}
}

var (
	gzpools [gzip.BestCompression]sync.Pool
	gzonce  sync.Once
)

func initGzPool() {
	for i := range gzpools {
		level := i
		gzpools[i].New = func() interface{} {
			return newGzipRW(level)
		}
	}
}

func getGzRW(level int) *gzRW {
	gzonce.Do(initGzPool)

	if level < gzip.NoCompression || level > gzip.BestCompression {
		level = 6
	}

	return gzpools[level].Get().(*gzRW)
}

func newGzipRW(level int) *gzRW {
	gw, _ := gzip.NewWriterLevel(nil, level)
	return &gzRW{
		gw:    gw,
		level: level,
	}
}

type gzRW struct {
	http.ResponseWriter
	gw    *gzip.Writer
	level int
}

func (g *gzRW) init(ctx *Context) {
	g.ResponseWriter = ctx.ResponseWriter
	g.gw.Reset(g.ResponseWriter)

	ctx.Header().Set(encodingHeader, gzEnc)
	ctx.ResponseWriter = g
}

func (g *gzRW) Write(p []byte) (int, error) {
	return g.gw.Write(p)
}

func (g *gzRW) Flush() {
	g.gw.Flush()

	if hf, ok := g.ResponseWriter.(http.Flusher); ok {
		hf.Flush()
	}
}

func (g *gzRW) Reset() {
	g.gw.Close()
	g.gw.Reset(nil)
	g.ResponseWriter = nil
	gzpools[g.level].Put(g)
}
