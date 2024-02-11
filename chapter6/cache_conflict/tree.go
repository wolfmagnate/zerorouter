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

func findWildcard(path string) (wildcard string, i int, valid bool) {
	i1 := strings.Index(path, ":")
	i2 := strings.Index(path, "/*")

	if i1 == -1 && i2 == -1 {
		return "", -1, true
	}

	if i1 != -1 && (i2 == -1 || i1 < i2) {
		wildcard, _, err := extractParam(path[i1:])
		return wildcard, i1, err == nil
	}

	if i2 != -1 && (i1 == -1 || i2 < i1) {
		wildcard, _, err := extractCatchAll(path[i2:])
		return wildcard, i2, err == nil
	}

	// ここは無意味なコード
	return "", -1, false
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
	path             string
	children         []*node
	nType            nodeType
	handle           Handle
	hasParamChild    bool
	hasCatchAllChild bool
	hasSlashChild    bool
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
				path:             suffix,
				nType:            static,
				children:         n.children,
				handle:           n.handle,
				hasParamChild:    n.hasParamChild,
				hasCatchAllChild: n.hasCatchAllChild,
				hasSlashChild:    n.hasSlashChild,
			}

			n.children = []*node{child}
			n.path = prefix
			n.handle = nil
			n.hasParamChild = false
			n.hasCatchAllChild = false
			n.hasSlashChild = suffix[0] == '/'
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

			if len(n.children) == 0 {
				// 挿入
				n.insertChild(path, handle)
			} else {
				// 探索
				n = n.children[0]
				goto walk
			}
		case '/':
			if len(path) == 1 || path[1] != '*' {
				n.checkConflict_static(path)
				next := path[0]
				for _, child := range n.children {
					if child.path[0] == next {
						// 探索
						n = child
						goto walk
					}
				}
				// 挿入
				n.insertChild(path, handle)
				return
			}
			catchAllName, _, err := extractCatchAll(path)
			if err != nil {
				panic("invalid catchAll")
			}
			n.checkConflict_catchAll(catchAllName)
			for _, child := range n.children {
				if child.nType == catchAll && child.path == catchAllName {
					// 探索
					n = child
					goto walk
				}
			}
			// 挿入
			n.insertChild(path, handle)
		case '*':
			panic("catchAll pattern must be after slash")
		default:
			n.checkConflict_static(path)
			next := path[0]
			for _, child := range n.children {
				if child.path[0] == next {
					// 探索
					n = child
					goto walk
				}
			}
			// 挿入
			n.insertChild(path, handle)
		}
	} else {
		n.handle = handle
	}
}

func (n *node) insertChild(path string, handle Handle) {
	child := &node{}
	n.children = append(n.children, child)
	parent := n
	n = child
	for {
		wildcard, i, valid := findWildcard(path)
		if i < 0 {
			if path[0] == '/' {
				parent.hasSlashChild = true
			}
			n.path = path
			n.handle = handle
			return
		}
		if !valid {
			panic("invalid wildcard found")
		}

		if wildcard[0] == ':' {
			if i > 0 {
				n.path = path[:i]
				path = path[i:]
				child := &node{
					nType: param,
					path:  wildcard,
				}
				n.children = []*node{child}
				n.hasParamChild = true
				if path[0] == '/' {
					parent.hasSlashChild = true
				}
				parent = n
				n = child
			} else {
				// nがparamになる
				n.nType = param
				n.path = wildcard
				parent.hasParamChild = true
			}

			// パラメータノードより深く行くとき
			if len(wildcard) < len(path) {
				path = path[len(wildcard):]
				child := &node{}
				n.children = []*node{child}
				parent = n
				n = child
				continue
			} else {
				n.handle = handle
			}
			return
		} else if wildcard[0:2] == "/*" {
			// catchAll以降があったらエラー
			if i+len(wildcard) != len(path) {
				panic("catchAll must be the last pattern")
			}
			if i > 0 {
				n.path = path[:i]
				child := &node{
					nType:  catchAll,
					path:   wildcard,
					handle: handle,
				}
				n.children = []*node{child}
				n.hasCatchAllChild = true
				if path[0] == '/' {
					parent.hasSlashChild = true
				}
			} else {
				n.path = wildcard
				n.nType = catchAll
				n.handle = handle
				parent.hasCatchAllChild = true
			}
			return
		}
	}
}

type conflictPanic struct {
	targetNode *node
	newName    string
	newType    nodeType
}

func (n *node) checkConflict_static(str string) {
	if n.nType == static {
		if !(n.hasParamChild || n.hasCatchAllChild) {
			return
		}
		if n.hasCatchAllChild && str[0] != '/' {
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
		newName:    str,
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
		if !n.hasSlashChild && !n.hasParamChild && !n.hasCatchAllChild {
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
				Value: path[0:end],
			})
			n = child
			path = path[end:]
			goto walk
		case catchAll:
			if path[0] == '/' {
				ps = append(ps, Param{
					Key:   child.path[2:],
					Value: path,
				})
				n = child
				path = path[len(path):]
				goto walk
			}
		}
	}
	return nil, nil
}
