package router

// Param is a key/value pair
type Param struct {
	Name  string
	Value string
}

// Params handles the named params in your url, it is *NOT* safe to be used outside of your handler.
type Params []Param

// Get returns a param by name
func (p Params) Get(name string) string {
	for i := range p {
		if v := &p[i]; v.Name == name {
			return v.Value
		}
	}
	return ""
}

// GetExt returns the value split at the last extension available, for example:
//	if :filename == "report.json", GetExt("filename") returns "report", "json"
func (p Params) GetExt(name string) (val, ext string) {
	val = p.Get(name)
	for i := len(val) - 1; i > -1; i-- {
		if val[i] == '.' {
			return val[:i], val[i+1:]
		}
	}
	return
}

// Copy returns a copy of p, required if you want to store it somewhere or use it outside of your handler.
func (p Params) Copy() Params {
	op := make(Params, len(p))
	copy(op, p)
	return op
}

// this wraps the slice to avoid an extra allocation using the pool
type paramsWrapper struct {
	p Params
}

func (pw *paramsWrapper) Params() (p Params) {
	if pw != nil {
		p = pw.p
	}
	return
}
