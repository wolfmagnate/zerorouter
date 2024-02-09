// Copyright 2024 進捗ゼミ. All rights reserved.
// Based on the path package, Copyright 2009 The Go Authors.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package chapter7

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
	indices          string
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
				indices:          n.indices,
				handle:           n.handle,
				hasParamChild:    n.hasParamChild,
				hasCatchAllChild: n.hasCatchAllChild,
				hasSlashChild:    n.hasSlashChild,
			}

			n.children = []*node{child}
			n.indices = string(suffix[0])
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
				for i, c := range []byte(n.indices) {
					if c == next {
						// 探索
						n = n.children[i]
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
			for i, c := range []byte(n.indices) {
				if c == '*' && n.children[i].path == catchAllName {
					// 探索
					n = n.children[i]
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
			for i, c := range []byte(n.indices) {
				if c == next {
					// 探索
					n = n.children[i]
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
	// 前提条件：空っぽのノードに対してpathを入れていくぞ！
	// コンフリクトが絶対に起きないため、めちゃくちゃ条件判定を省ける
	for {
		wildcard, i, valid := findWildcard(path)
		if i < 0 {
			break
		}
		if !valid {
			panic("invalid wildcard found")
		}

		if wildcard[0] == ':' {
			if i > 0 {
				n.path = path[:i]
				parent.indices += string(path[0])
				path = path[i:]
				child := &node{
					nType: param,
					path:  wildcard,
				}
				n.children = []*node{child}
				n.indices = ":"
				n.hasParamChild = true
				parent = n
				n = child
			} else {
				// nがparamになる
				n.nType = param
				n.path = wildcard
				parent.hasParamChild = true
				parent.indices += ":"
			}

			// パラメータノードより深く行くとき
			if len(wildcard) < len(path) {
				path = path[len(wildcard):]
				child := &node{}
				n.children = []*node{child}
				// この時点ではchildが何になるか分からないからindicesは設定できない
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
				parent.indices += string(path[0])
				child := &node{
					nType:  catchAll,
					path:   wildcard,
					handle: handle,
				}
				n.children = []*node{child}
				n.indices = "*"
				n.hasCatchAllChild = true
			} else {
				n.path = wildcard
				n.nType = catchAll
				n.handle = handle
				parent.hasCatchAllChild = true
				parent.indices += "*"
			}
			return
		}
	}
	if path[0] == '/' {
		parent.hasSlashChild = true
	}
	parent.indices += string(path[0])
	n.path = path
	n.handle = handle
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

type paramsProvider interface {
	add(Param)
	getParams() *Params
}

type funcParamsProvider struct {
	ps          *Params
	provideFunc func() *Params
}

func (f *funcParamsProvider) add(p Param) {
	if f.ps == nil {
		f.ps = f.provideFunc()
	}
	i := len(*(f.ps))
	*f.ps = (*(f.ps))[:i+1]
	(*(f.ps))[i] = p
}

func (f *funcParamsProvider) getParams() *Params {
	return f.ps
}

func (n *node) retrieve(path string, params func() *Params) (handle Handle, ps *Params) {
	return n._retrieve(path, &funcParamsProvider{
		provideFunc: params,
	})
}

type nilParamsProvider struct{}

func (n *nilParamsProvider) add(p Param)        {}
func (n *nilParamsProvider) getParams() *Params { return nil }

func (n *node) retrieve_nil(path string) Handle {
	handle, _ := n._retrieve(path, &nilParamsProvider{})
	return handle
}

func (n *node) _retrieve(path string, provider paramsProvider) (handle Handle, ps *Params) {
walk:
	if len(path) == 0 {
		return n.handle, provider.getParams()
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
			provider.add(Param{
				Key:   child.path[1:],
				Value: path[:end],
			})
			n = child
			path = path[end:]
			goto walk
		case catchAll:
			if path[0] == '/' {
				provider.add(Param{
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
