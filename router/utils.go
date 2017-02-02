package router

import (
	"log"
	"net/http"
)

// PanicHandler is the default panic handler
func PanicHandler(w http.ResponseWriter, req *http.Request, v interface{}) {
	http.Error(w, "oops", 500)
	log.Printf("panic: %v", v)
}

func redirectByMethod(w http.ResponseWriter, req *http.Request, url string) {
	if req.Method == "GET" {
		http.Redirect(w, req, url, http.StatusMovedPermanently)
	} else {
		http.Redirect(w, req, url, http.StatusTemporaryRedirect)
	}
}

type nodePart struct {
	Name string
	Type uint8
}

func splitPathToParts(p string) (pp string, rest []nodePart, num, stars int) {
	for i, last := 0, 0; i < len(p); i++ {
		c, isEnd := p[i], i == len(p)-1
		if isEnd {
			i++
		}
		if c == '/' || isEnd {
			if len(pp) > 0 {
				n := p[last+1 : i]
				if ln := len(n); ln > 0 {
					switch c := n[0]; c {
					case ':', '*':
						if ln > 1 {
							rest = append(rest, nodePart{n[1:], c})
						}
					default:
						rest = append(rest, nodePart{n, 0})
					}
				}
			}
			last = i
		} else if c == '*' {
			num++
			stars++
			if len(pp) == 0 {
				pp = p[:i]
			}
		} else if c == ':' {
			num++
			if len(pp) == 0 {
				pp = p[:i]
			}
		}
	}
	if len(pp) == 0 {
		pp = p
	}
	return
}
