// Copyright 2024 進捗ゼミ. All rights reserved.
// Based on the path package, Copyright 2009 The Go Authors.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.
package chapter4

import "net/http"

var MatchedRoutePathParam = "$matchedRoutePath"

func (r *Router) saveMatchedRoutePath(path string, handle Handle) Handle {
	return func(w http.ResponseWriter, req *http.Request, ps Params) {
		ps = append(ps, Param{Key: MatchedRoutePathParam, Value: path})
		handle(w, req, ps)
	}
}
