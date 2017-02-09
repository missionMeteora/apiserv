package router

import "net/http"

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

// splitPathToParts takes in a path (ex: /api/v1/someEndpoint/:id/*any) and returns:
//	pp -> the longest part before the first param (/api/v1/someEndpoint/:)
//	rest -> all the params (id, any)
//	num -> number of params (probably not needed...)
//	stars -> number of stars, basically a sanity check, if it's not 0 or 1 then it's an invalid path
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
