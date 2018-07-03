package apiserv

import (
	"mime"
	"os"
	"path/filepath"
	"strings"
)

var (
	brEnc = "br"
	gzEnc = "gzip"
)

func TryCompressed(ctx *Context, fname string) error {
	gz, br := accepts(ctx.ReqHeader().Get("Accept-Encoding"))
	ctx.SetContentType(mime.TypeByExtension(filepath.Ext(fname)))

	if br && fileExists(fname+".br") {
		ctx.Header().Set("Content-Encoding", brEnc)
		return ctx.File(fname + ".br")
	}

	if gz && fileExists(fname+".gz") {
		ctx.Header().Set("Content-Encoding", gzEnc)
		return ctx.File(fname + ".gz")
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
