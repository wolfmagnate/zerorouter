// Copyright 2024 進捗ゼミ. All rights reserved.
// Based on the path package, Copyright 2009 The Go Authors.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package chapter4

import (
	"fmt"
	"unicode"
)

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

func extractParam(path []rune) (string, int, error) {
	if len(path) < 2 || path[0] != ':' || string(path) == ":/" {
		return "", 0, fmt.Errorf("invalid path parameter")
	}

	for i := 1; i < len(path); i++ {
		if path[i] == '/' {
			return string(path[:i]), i, nil
		}
		if path[i] == '*' || path[i] == ':' {
			return "", 0, fmt.Errorf("invalid catchAll is in param name")
		}
	}
	return string(path), len(path), nil
}
func extractCatchAll(path []rune) (string, int, error) {
	if len(path) < 3 || string(path[0:2]) != "/*" || string(path) == "/*/" {
		return "", 0, fmt.Errorf("invalid catch-all parameter")
	}

	for i := 2; i < len(path); i++ {
		if path[i] == '/' {
			return string(path[:i]), i, nil
		}
		if path[i] == '*' || path[i] == ':' {
			return "", 0, fmt.Errorf("invalid param is in catchall name")
		}
	}
	return string(path), len(path), nil
}

func (n *node) addRoute(path string, handle Handle) {
	n.addRoute_rune([]rune(path), handle)
}

func (n *node) addRoute_rune(path []rune, handle Handle) {
	if len(path) == 0 {
		n.handle = handle
		return
	}

	if n.children == nil {
		n.children = make([]*node, 0)
	}

	switch string(path[0]) {
	case ":":
		paramName, paramLen, err := extractParam(path)
		if err != nil {
			panic("invalid parameter")
		}
		n.checkConflict_param(paramName)
		n.handle_param(path, paramName, paramLen, handle)
	case "/":
		if len(path) == 1 || string(path[1]) != "*" {
			n.checkConflict_static(path[0])
			n.handle_static(path, handle)
			return
		}
		catchAllName, pathLen, err := extractCatchAll(path)
		if err != nil {
			panic("invalid catchAll")
		}
		n.checkConflict_catchAll(catchAllName)
		n.handle_catchAll(path, catchAllName, pathLen, handle)
	case "*":
		panic("catchAll pattern must be after slash")
	default:
		n.checkConflict_static(path[0])
		n.handle_static(path, handle)
	}
}

func (n *node) handle_param(path []rune, paramName string, pathLen int, handle Handle) {
	if len(n.children) == 0 {
		nextNode := &node{
			path:  paramName,
			nType: param,
		}
		n.children = append(n.children, nextNode)
		nextNode.addRoute_rune(path[pathLen:], handle)
	} else {
		n.children[0].addRoute_rune(path[pathLen:], handle)
	}
}

func (n *node) handle_static(path []rune, handle Handle) {
	next := string(path[0])

	for _, child := range n.children {
		if child.path == next {
			child.addRoute_rune(path[1:], handle)
			return
		}
	}

	nextNode := &node{
		path: next,
	}
	n.children = append(n.children, nextNode)
	nextNode.addRoute_rune(path[1:], handle)
}

func (n *node) handle_catchAll(path []rune, catchAllName string, pathLen int, handle Handle) {
	for _, child := range n.children {
		if child.nType == catchAll && child.path == catchAllName {
			child.addRoute_rune(path[pathLen:], handle)
			return
		}
	}
	nextNode := &node{
		path:  catchAllName,
		nType: catchAll,
	}
	n.children = append(n.children, nextNode)
	nextNode.addRoute_rune(path[pathLen:], handle)
}

type conflictPanic struct {
	targetNode *node
	newName    string
	newType    nodeType
}

func (n *node) checkConflict_static(str rune) {
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
		if containsCatchAll && str != '/' {
			return
		}
	}

	if n.nType == param && str == '/' {
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
			if c.nType == param || (c.nType == static && c.path == "/") {
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
	return n.retrieve_rune([]rune(path), make(Params, 0))
}

func (n *node) retrieve_rune(path []rune, ps Params) (Handle, Params) {
	if len(path) == 0 {
		return n.handle, ps
	}
	next := string(path[0])
	for _, child := range n.children {
		switch child.nType {
		case static:
			if child.path == next {
				return child.retrieve_rune(path[1:], ps)
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
			return child.retrieve_rune(path[end:], ps)
		case catchAll:
			if next == "/" {
				ps = append(ps, Param{
					Key:   child.path[2:],
					Value: string(path),
				})
				return child.retrieve_rune(path[len(path):], ps)
			}
		}
	}
	return nil, nil
}

func (n *node) retrieve_caseInsensitive(path string) (Handle, Params) {
	return n.retrieve_caseInsensitive_rune([]rune(path), make(Params, 0))
}
func (n *node) retrieve_caseInsensitive_rune(path []rune, ps Params) (Handle, Params) {
	if len(path) == 0 {
		return n.handle, ps
	}

	next_u := string(unicode.ToUpper(path[0]))
	next_l := string(unicode.ToLower(path[0]))
	for _, child := range n.children {
		switch child.nType {
		case static:
			var handle_u, handle_l Handle
			var ps_u, ps_l Params
			if child.path == next_u {
				handle_u, ps_u = child.retrieve_caseInsensitive_rune(path[1:], make(Params, 0))
			}
			if child.path == next_l {
				handle_l, ps_l = child.retrieve_caseInsensitive_rune(path[1:], make(Params, 0))
			}
			if handle_u != nil {
				ps = append(ps, ps_u...)
				return handle_u, ps
			}
			if handle_l != nil {
				ps = append(ps, ps_l...)
				return handle_l, ps
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
				Key:   n.path[1:],
				Value: string(path[0:end]),
			})
			return child.retrieve_caseInsensitive_rune(path[end:], ps)
		case catchAll:
			if next_u == "/" || next_l == "/" {
				ps = append(ps, Param{
					Key:   n.path[2:],
					Value: string(path),
				})
				return child.retrieve_caseInsensitive_rune(path[len(path):], ps)
			}
		}
	}

	return nil, nil
}
