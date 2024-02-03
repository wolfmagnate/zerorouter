package chapter4

import "net/http"

var MatchedRoutePathParam = "$matchedRoutePath"

func (r *Router) saveMatchedRoutePath(path string, handle Handle) Handle {
	return func(w http.ResponseWriter, req *http.Request, ps Params) {
		ps = append(ps, Param{Key: MatchedRoutePathParam, Value: path})
		handle(w, req, ps)
	}
}
