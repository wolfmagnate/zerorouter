// Copyright 2024 進捗ゼミ. All rights reserved.
// Based on the path package, Copyright 2009 The Go Authors.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.
package chapter7

import (
	"net/http"
	"slices"
	"strings"
)

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
