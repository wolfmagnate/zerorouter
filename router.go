// Copyright 2024 進捗ゼミ. All rights reserved.
// Based on the path package, Copyright 2009 The Go Authors.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.
package zerorouter

import (
	"context"
	"net/http"
	"path"
	"slices"
	"strings"
	"sync"
)

func (r *Router) GET(path string, handle Handle) {
	r.Handle(http.MethodGet, path, handle)
}

func (r *Router) HEAD(path string, handle Handle) {
	r.Handle(http.MethodHead, path, handle)
}

func (r *Router) OPTIONS(path string, handle Handle) {
	r.Handle(http.MethodOptions, path, handle)
}

func (r *Router) POST(path string, handle Handle) {
	r.Handle(http.MethodPost, path, handle)
}

func (r *Router) PUT(path string, handle Handle) {
	r.Handle(http.MethodPut, path, handle)
}

func (r *Router) PATCH(path string, handle Handle) {
	r.Handle(http.MethodPatch, path, handle)
}

func (r *Router) DELETE(path string, handle Handle) {
	r.Handle(http.MethodDelete, path, handle)
}

func (r *Router) Handle(method, path string, handle Handle) {
	if method == "" {
		panic("method must not be empty")
	}
	if len(path) < 1 || path[0] != '/' {
		panic("path must begin with '/' in path '" + path + "'")
	}
	if handle == nil {
		panic("handle must not be nil")
	}

	varsCount := 0
	if r.SaveMatchedRoutePath {
		varsCount++
		handle = r.saveMatchedRoutePath(path, handle)
	}

	if r.trees == nil {
		r.trees = make(map[string]*node)
	}

	root := r.trees[method]
	if root == nil {
		root = new(node)
		r.trees[method] = root
	}

	root.addRoute(path, handle)

	if paramsCount := countParams(path); paramsCount+varsCount > r.maxParams {
		r.maxParams = paramsCount + varsCount
	}

	if r.paramsPool.New == nil && r.maxParams > 0 {
		r.paramsPool.New = func() interface{} {
			ps := make(Params, 0, r.maxParams)
			return &ps
		}
	}
}

func (ps Params) ByName(name string) string {
	for _, p := range ps {
		if p.Key == name {
			return p.Value
		}
	}
	return ""
}

type paramsKey struct{}

var ParamsKey = paramsKey{}

func ParamsFromContext(ctx context.Context) Params {
	p, _ := ctx.Value(ParamsKey).(Params)
	return p
}

func (r *Router) Handler(method, path string, handler http.Handler) {
	r.Handle(method, path,
		func(w http.ResponseWriter, req *http.Request, p Params) {
			if len(p) > 0 {
				ctx := req.Context()
				ctx = context.WithValue(ctx, ParamsKey, p)
				req = req.WithContext(ctx)
			}
			handler.ServeHTTP(w, req)
		},
	)
}

func (r *Router) HandlerFunc(method, path string, handler http.HandlerFunc) {
	r.Handler(method, path, handler)
}

func (r *Router) allowed(path string) string {
	allowed := make([]string, 0)

	if path == "*" {
		for method := range r.trees {
			if method == http.MethodOptions {
				continue
			}
			allowed = append(allowed, method)
		}
	} else {
		for method := range r.trees {
			if method == http.MethodOptions {
				continue
			}
			handle := r.trees[method].retrieve_noparam(path)
			if handle != nil {
				allowed = append(allowed, method)
			}
		}
	}

	if len(allowed) > 0 {
		allowed = append(allowed, http.MethodOptions)
		slices.Sort(allowed)
		return strings.Join(allowed, ", ")
	}

	return ""
}

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

func (r *Router) recv(w http.ResponseWriter, req *http.Request) {
	if rcv := recover(); rcv != nil {
		r.PanicHandler(w, req, rcv)
	}
}

type Router struct {
	trees                  map[string]*node
	PanicHandler           func(http.ResponseWriter, *http.Request, interface{})
	HandleOPTIONS          bool
	HandleMethodNotAllowed bool
	SaveMatchedRoutePath   bool
	RedirectFixedPath      bool
	paramsPool             sync.Pool
	maxParams              int
}

func New() *Router {
	return &Router{}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if r.PanicHandler != nil {
		defer r.recv(w, req)
	}
	urlPath := req.URL.Path
	if root := r.trees[req.Method]; root != nil {
		if handle, ps := root.retrieve(urlPath, r.getParams); handle != nil {
			if ps != nil {
				handle(w, req, *ps)
				r.putParams(ps)
			} else {
				handle(w, req, nil)
			}
			return
		} else if urlPath != "/" {
			code := http.StatusMovedPermanently
			if req.Method != http.MethodGet {
				code = http.StatusPermanentRedirect
			}
			if r.RedirectFixedPath {
				fixedPath := path.Clean(urlPath)
				if handle := root.retrieve_noparam(fixedPath); handle != nil {
					req.URL.Path = fixedPath
					http.Redirect(w, req, req.URL.String(), code)
					return
				}
			}
		}
	}
	if req.Method == http.MethodOptions && r.HandleOPTIONS {
		if allow := r.allowed(urlPath); allow != "" {
			w.Header().Set("Allow", allow)
			return
		}
	} else if r.HandleMethodNotAllowed {
		if allow := r.allowed(urlPath); allow != "" {
			w.Header().Set("Allow", allow)
			http.Error(w,
				http.StatusText(http.StatusMethodNotAllowed),
				http.StatusMethodNotAllowed,
			)
			return
		}
	}
	http.NotFound(w, req)
}

var MatchedRoutePathParam = "$matchedRoutePath"

func (r *Router) saveMatchedRoutePath(path string, handle Handle) Handle {
	return func(w http.ResponseWriter, req *http.Request, ps Params) {
		if ps == nil {
			psp := r.getParams()
			ps = (*psp)[0:1]
			ps[0] = Param{Key: MatchedRoutePathParam, Value: path}
			handle(w, req, ps)
			r.putParams(psp)
		} else {
			i := len(ps)
			ps = ps[:i+1]
			ps[i] = Param{Key: MatchedRoutePathParam, Value: path}
			handle(w, req, ps)
		}
	}
}
