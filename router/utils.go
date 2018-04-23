package router

import (
	"fmt"
	"net/http"
	"regexp"
)

func redirectByMethod(w http.ResponseWriter, req *http.Request, url string) {
	if req.Method == "GET" {
		http.Redirect(w, req, url, http.StatusMovedPermanently)
	} else {
		http.Redirect(w, req, url, http.StatusTemporaryRedirect)
	}
}

type nodePart string

func (np nodePart) Name() string { return string(np[1:]) }
func (np nodePart) Type() uint8  { return np[0] }
func (np nodePart) String() string {
	if np.Type() == '/' {
		return fmt.Sprintf("{%s}", np.Name())
	}
	return fmt.Sprintf("{%s '%c'}", np.Name(), np.Type())
}

var re = regexp.MustCompile(`([:*/]?[^:*]+)`)

// splitPathToParts takes in a path (ex: /api/v1/someEndpoint/:id/*any) and returns:
//	pp -> the longest part before the first param (/api/v1/someEndpoint/:)
//	rest -> all the params (id, any)
//	num -> number of params (probably not needed...)
//	stars -> number of stars, basically a sanity check, if it's not 0 or 1 then it's an invalid path
func splitPathToParts(p string) (pp string, rest []nodePart, num, stars int) {
	parts := re.FindAllString(p, -1)
	if len(parts) < 2 {
		pp = p
		return
	}

	pp = parts[0]
	for _, part := range parts[1:] {
		splitPathFn(part, '/', func(sp string, _, _ int) bool {
			switch c := sp[0]; c {
			case '*':
				stars++
				fallthrough
			case ':':
				num++
				fallthrough
			case '/':
				rest = append(rest, nodePart(sp))
			}
			return false
		})
	}
	return
}

func splitPathFn(s string, sep uint8, fn func(p string, pidx, idx int) bool) bool {
	for i, pi, last := 0, 0, 0; i < len(s); i++ {
		if s[i] != sep {
			if i < len(s)-1 {
				continue
			}
			i = len(s)
		}

		if ss := s[last:i]; ss != "" {
			if fn(ss, pi, i) {
				return true
			}
			last = i
			pi++
		}
	}

	return false
}

func revSplitPathFn(s string, sep uint8, fn func(p string, pidx, idx int) bool) bool {
	for i, pi, last := len(s)-1, 0, len(s); i > -1; i-- {
		if s[i] != sep {
			continue
		}
		if ss := s[i:last]; ss != "" {
			if fn(ss, pi, last) {
				return true
			}
			last = i
			pi++
		}
	}

	return false
}

// based on https://github.com/gin-gonic/gin/blob/a8fa424ae529397d4a0f2a1f9fda8031851a3269/path.go#L21
// cleanPath is the URL version of path.Clean, it returns a canonical URL path
// for p, eliminating . and .. elements.
//
// The following rules are applied iteratively until no further processing can
// be done:
//	1. Replace multiple slashes with a single slash.
//	2. Eliminate each . path name element (the current directory).
//	3. Eliminate each inner .. path name element (the parent directory)
//	   along with the non-.. element that precedes it.
//	4. Eliminate .. elements that begin a rooted path:
//	   that is, replace "/.." by "/" at the beginning of a path.
//
// If the result of this process is an empty string, "/" is returned.
func cleanPath(p string) (_ string, modified bool) {
	// Turn empty string into "/"
	if p == "" {
		return "/", false
	}

	n := len(p)
	var buf []byte

	// Invariants:
	//      reading from path; r is index of next byte to process.
	//      writing to buf; w is index of next byte to write.

	// path must start with '/'
	r := 1
	w := 1

	if p[0] != '/' {
		r = 0
		buf = make([]byte, n+1)
		buf[0] = '/'
	}

	trailing := n > 2 && p[n-1] == '/'

	// A bit more clunky without a 'lazybuf' like the path package, but the loop
	// gets completely inlined (bufApp). So in contrast to the path package this
	// loop has no expensive function calls (except 1x make)

	for r < n {
		switch {
		case p[r] == '/':
			// empty path element, trailing slash is added after the end
			r++

		case p[r] == '.' && r+1 == n:
			trailing = true
			r++

		case p[r] == '.' && p[r+1] == '/':
			// . element
			r++

		case p[r] == '.' && p[r+1] == '.' && (r+2 == n || p[r+2] == '/'):
			// .. element: remove to last /
			r += 2

			if w > 1 {
				// can backtrack
				w--

				if buf == nil {
					for w > 1 && p[w] != '/' {
						w--
					}
				} else {
					for w > 1 && buf[w] != '/' {
						w--
					}
				}
			}

		default:
			// real path element.
			// add slash if needed
			if w > 1 {
				bufApp(&buf, p, w, '/')
				w++
			}

			// copy element
			for r < n && p[r] != '/' {
				bufApp(&buf, p, w, p[r])
				w++
				r++
			}
		}
	}

	// re-append trailing slash
	if trailing && w > 1 {
		bufApp(&buf, p, w, '/')
		w++
	}

	if buf == nil {
		return p[:w], w < len(p)
	}

	return string(buf[:w]), true
}

// internal helper to lazily create a buffer if necessary.
func bufApp(buf *[]byte, s string, w int, c byte) {
	if *buf == nil {
		if s[w] == c {
			return
		}

		*buf = make([]byte, len(s))
		copy(*buf, s[:w])
	}
	(*buf)[w] = c
}
