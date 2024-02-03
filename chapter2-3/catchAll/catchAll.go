// Copyright 2024 進捗ゼミ. All rights reserved.
// Based on the path package, Copyright 2009 The Go Authors.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package chapter2

import (
	"fmt"
	"net/http"
)

type Handle func(http.ResponseWriter, *http.Request)

type nodeType uint8

const (
	static nodeType = iota
	root
	param
	catchAll
)

type catchAllNode struct {
	path     string
	children []*catchAllNode
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

func (n *catchAllNode) addRoute(path string, handle Handle) {
	n.addRoute_rune([]rune(path), handle)
}

func (n *catchAllNode) addRoute_rune(path []rune, handle Handle) {
	if len(path) == 0 {
		n.handle = handle
		return
	}

	if n.children == nil {
		n.children = make([]*catchAllNode, 0)
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

func (n *catchAllNode) handle_param(path []rune, paramName string, pathLen int, handle Handle) {
	if len(n.children) == 0 {
		nextNode := &catchAllNode{
			path:  paramName,
			nType: param,
		}
		n.children = append(n.children, nextNode)
		nextNode.addRoute_rune(path[pathLen:], handle)
	} else {
		n.children[0].addRoute_rune(path[pathLen:], handle)
	}
}

func (n *catchAllNode) handle_static(path []rune, handle Handle) {
	next := string(path[0])

	for _, child := range n.children {
		if child.path == next {
			child.addRoute_rune(path[1:], handle)
			return
		}
	}

	nextNode := &catchAllNode{
		path: next,
	}
	n.children = append(n.children, nextNode)
	nextNode.addRoute_rune(path[1:], handle)
}

func (n *catchAllNode) handle_catchAll(path []rune, catchAllName string, pathLen int, handle Handle) {
	for _, child := range n.children {
		if child.nType == catchAll && child.path == catchAllName {
			child.addRoute_rune(path[pathLen:], handle)
			return
		}
	}
	nextNode := &catchAllNode{
		path:  catchAllName,
		nType: catchAll,
	}
	n.children = append(n.children, nextNode)
	nextNode.addRoute_rune(path[pathLen:], handle)
}

type conflictPanic struct {
	targetNode *catchAllNode
	newName    string
	newType    nodeType
}

func (n *catchAllNode) checkConflict_static(str rune) {
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

func (n *catchAllNode) checkConflict_catchAll(catchAllName string) {
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

func (n *catchAllNode) checkConflict_param(paramName string) {
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

func (n *catchAllNode) retrieve(path string) Handle {
	return n.retrieve_rune([]rune(path))
}

func (n *catchAllNode) retrieve_rune(path []rune) Handle {
	if len(path) == 0 {
		return n.handle
	}

	next := string(path[0])
	for _, child := range n.children {
		switch child.nType {
		case static:
			if child.path == next {
				return child.retrieve_rune(path[1:])
			}
		case param:
			if path[0] == '/' {
				return nil
			}
			end := 1
			for end < len(path) && path[end] != '/' {
				end++
			}
			return n.children[0].retrieve_rune(path[end:])
		case catchAll:
			if next == "/" {
				return child.retrieve_rune(path[len(path):])
			}
		}
	}

	return nil
}
