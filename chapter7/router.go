// Copyright 2024 進捗ゼミ. All rights reserved.
// Based on the path package, Copyright 2009 The Go Authors.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.
package chapter7

import (
	"net/http"
	"path"
	"sync"
)

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
