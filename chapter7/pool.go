package chapter7

func (r *Router) getParams() *Params {
	ps, _ := r.paramsPool.Get().(*Params)
	*ps = (*ps)[0:0]
	return ps
}

func (r *Router) putParams(ps *Params) {
	if ps != nil {
		r.paramsPool.Put(ps)
	}
}

func countParams(path string) int {
	var n int
	for i := range []byte(path) {
		switch path[i] {
		case ':', '*':
			n++
		}
	}
	return n
}
