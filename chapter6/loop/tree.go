// Copyright 2024 進捗ゼミ. All rights reserved.
// Based on the path package, Copyright 2009 The Go Authors.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package chapter6

import (
	"fmt"
	"net/http"
	"strings"
)

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

func longestCommonPrefix(a, b string) int {
	i := 0
	max := min(len(a), len(b))
	for i < max && a[i] == b[i] {
		i++
	}
	return i
}

func findNearbyWildcard(path string) int {
	i1 := strings.Index(path, ":")
	i2 := strings.Index(path, "/*")
	if i1 == -1 && i2 == -1 {
		return -1
	}
	if i1 != -1 && (i2 == -1 || i1 < i2) {
		return i1
	}
	if i2 != -1 && (i1 == -1 || i2 < i1) {
		return i2
	}
	// ここは無意味なコード
	return -1
}

type Handle func(http.ResponseWriter, *http.Request, Params)

type Param struct {
	Key   string
	Value string
}
type Params []Param

type nodeType uint8

const (
	static nodeType = iota
	root
	param
	catchAll
)

type node struct {
	path     string
	children []*node
	nType    nodeType
	handle   Handle
}

func extractParam(path string) (string, int, error) {
	if len(path) < 2 || path[0] != ':' || path == ":/" {
		return "", 0, fmt.Errorf("invalid path parameter")
	}

	for i := 1; i < len(path); i++ {
		if path[i] == '/' {
			return path[:i], i, nil
		}
		if path[i] == '*' || path[i] == ':' {
			return "", 0, fmt.Errorf("invalid catchAll is in param name")
		}
	}
	return path, len(path), nil
}

func extractCatchAll(path string) (string, int, error) {
	if len(path) < 3 || path[0:2] != "/*" || path == "/*/" {
		return "", 0, fmt.Errorf("invalid catch-all parameter")
	}

	for i := 2; i < len(path); i++ {
		if path[i] == '/' {
			return path[:i], i, nil
		}
		if path[i] == '*' || path[i] == ':' {
			return "", 0, fmt.Errorf("invalid param is in catchall name")
		}
	}
	return path, len(path), nil
}

func (n *node) addRoute(path string, handle Handle) {
walk:
	if n.children == nil {
		n.children = make([]*node, 0)
	}

	i := longestCommonPrefix(path, n.path)

	// ノードの分割
	if n.nType == static {
		if i < len(n.path) {
			prefix := n.path[:i]
			suffix := n.path[i:]

			child := &node{
				path:     suffix,
				nType:    static,
				children: n.children,
				handle:   n.handle,
			}

			n.children = []*node{child}
			n.path = prefix
			n.handle = nil
		}
	}

	// パスの分割
	if i < len(path) {
		path = path[i:]

		switch path[0] {
		case ':':
			paramName, _, err := extractParam(path)
			if err != nil {
				panic("invalid parameter")
			}
			n.checkConflict_param(paramName)

			// ここから
			if len(n.children) == 0 {
				nextNode := &node{
					path:  paramName,
					nType: param,
				}
				n.children = append(n.children, nextNode)
				// nextNode.addRoute(newPath, handle)
				n = nextNode
				goto walk
			} else {
				// n.children[0].addRoute(newPath, handle)
				n = n.children[0]
				goto walk
			}
			// ここまで
		case '/':
			if len(path) == 1 || path[1] != '*' {
				n.checkConflict_static(path)
				// ここから
				next := path[0]
				for _, child := range n.children {
					if child.path[0] == next {
						// child.addRoute(path, handle)
						n = child
						goto walk
					}
				}
				i := findNearbyWildcard(path)
				if i == -1 {
					nextNode := &node{
						path:   path,
						nType:  static,
						handle: handle,
					}
					n.children = append(n.children, nextNode)
					return
				}
				nextNode := &node{
					path:  path[:i],
					nType: static,
				}
				n.children = append(n.children, nextNode)
				// nextNode.addRoute(path, handle)
				n = nextNode
				goto walk
				// ここまで
			}
			catchAllName, _, err := extractCatchAll(path)
			if err != nil {
				panic("invalid catchAll")
			}
			n.checkConflict_catchAll(catchAllName)
			// ここから
			for _, child := range n.children {
				if child.nType == catchAll && child.path == catchAllName {
					// child.addRoute(path, handle)
					n = child
					goto walk
				}
			}
			nextNode := &node{
				path:  catchAllName,
				nType: catchAll,
			}
			n.children = append(n.children, nextNode)
			// nextNode.addRoute(path, handle)
			n = nextNode
			goto walk
			// ここまで
		case '*':
			panic("catchAll pattern must be after slash")
		default:
			n.checkConflict_static(path)
			// ここから
			next := path[0]
			for _, child := range n.children {
				if child.path[0] == next {
					// child.addRoute(path, handle)
					n = child
					goto walk
				}
			}
			i := findNearbyWildcard(path)
			if i == -1 {
				nextNode := &node{
					path:   path,
					nType:  static,
					handle: handle,
				}
				n.children = append(n.children, nextNode)
				return
			}
			nextNode := &node{
				path:  path[:i],
				nType: static,
			}
			n.children = append(n.children, nextNode)
			// nextNode.addRoute(path, handle)
			n = nextNode
			goto walk
			// ここまで
		}
	} else {
		n.handle = handle
	}

}

type conflictPanic struct {
	targetNode *node
	newName    string
	newType    nodeType
}

func (n *node) checkConflict_static(str string) {
	if n.nType == static {
		containsParam := false
		containsCatchAll := false
		for _, c := range n.children {
			if c.nType == param {
				containsParam = true
			}
			if c.nType == catchAll {
				containsCatchAll = true
			}
		}
		if !(containsParam || containsCatchAll) {
			return
		}
		if containsCatchAll && str[0] != '/' {
			return
		}
	}

	if n.nType == param && str[0] == '/' {
		if len(n.children) == 0 {
			return
		}
		if n.children[0].nType == static {
			return
		}
	}

	panic(conflictPanic{
		targetNode: n,
		newName:    string(str),
		newType:    static,
	})
}

func (n *node) checkConflict_catchAll(catchAllName string) {
	if n.nType == static {
		if len(n.children) == 0 {
			return
		}
		if len(n.children) == 1 && n.children[0].nType == catchAll && n.children[0].path == catchAllName {
			return
		}
		containsSlashOrParam := false
		containsAnotherCatchAll := false
		for _, c := range n.children {
			if c.nType == param || (c.nType == static && c.path[0] == '/') {
				containsSlashOrParam = true
			}
			if c.nType == catchAll && c.path != catchAllName {
				containsAnotherCatchAll = true
			}
		}
		if !containsSlashOrParam && !containsAnotherCatchAll {
			return
		}
	}
	if n.nType == param {
		if len(n.children) == 0 {
			return
		}
		if len(n.children) == 1 && n.children[0].nType == catchAll && n.children[0].path == catchAllName {
			return
		}
	}
	panic(conflictPanic{
		targetNode: n,
		newName:    catchAllName,
		newType:    catchAll,
	})
}

func (n *node) checkConflict_param(paramName string) {
	if len(n.children) == 1 && n.children[0].nType == param && n.children[0].path == paramName {
		return
	}
	if n.nType == static && len(n.children) == 0 {
		return
	}
	panic(conflictPanic{
		targetNode: n,
		newName:    paramName,
		newType:    param,
	})
}

func (n *node) retrieve(path string) (Handle, Params) {
	return n.retrieve_loop(path, make(Params, 0))
}

func (n *node) retrieve_loop(path string, ps Params) (Handle, Params) {
walk:
	if len(path) == 0 {
		return n.handle, ps
	}
	for _, child := range n.children {
		switch child.nType {
		case static:
			if child.path == path[:len(child.path)] {
				n = child
				path = path[len(child.path):]
				goto walk
			}
		case param:
			if path[0] == '/' {
				return nil, nil
			}
			end := 1
			for end < len(path) && path[end] != '/' {
				end++
			}
			ps = append(ps, Param{
				Key:   child.path[1:],
				Value: string(path[0:end]),
			})
			n = child
			path = path[end:]
			goto walk
		case catchAll:
			if path[0] == '/' {
				ps = append(ps, Param{
					Key:   child.path[2:],
					Value: string(path),
				})
				n = child
				path = path[len(path):]
				goto walk
			}
		}
	}
	return nil, nil
}
