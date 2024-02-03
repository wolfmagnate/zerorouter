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

type paramNode struct {
	// パスパラメータは複数文字にわたる可能性があるため、stringに変更
	path     string
	children []*paramNode
	// パスパラメータor通常の文字 を判定するフラグ
	nType  nodeType
	handle Handle
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
			return "", 0, fmt.Errorf("invalid character is in param name")
		}
	}
	return string(path), len(path), nil
}

func (n *paramNode) addRoute(path string, handle Handle) {
	n.addRoute_rune([]rune(path), handle)
}

func (n *paramNode) addRoute_rune(path []rune, handle Handle) {
	if len(path) == 0 {
		n.handle = handle
		return
	}

	if n.children == nil {
		n.children = make([]*paramNode, 0)
	}

	if string(path[0]) == ":" {
		paramName, paramLen, err := extractParam(path)
		if err != nil {
			panic("Invalid path parameter")
		}
		n.checkConflict_param(paramName)
		n.handle_param(path, paramName, paramLen, handle)
	} else {
		n.checkConflict_static(path[0])
		n.handle_static(path, handle)
	}
}

func (n *paramNode) handle_param(path []rune, paramName string, pathLen int, handle Handle) {
	if len(n.children) == 0 {
		nextNode := &paramNode{
			path:  paramName,
			nType: param,
		}
		n.children = append(n.children, nextNode)
		nextNode.addRoute_rune(path[pathLen:], handle)
	} else {
		n.children[0].addRoute_rune(path[pathLen:], handle)
	}
}

func (n *paramNode) handle_static(path []rune, handle Handle) {
	next := string(path[0])

	for _, child := range n.children {
		if child.path == next {
			child.addRoute_rune(path[1:], handle)
			return
		}
	}

	nextNode := &paramNode{
		path: next,
	}
	n.children = append(n.children, nextNode)
	nextNode.addRoute_rune(path[1:], handle)
}

type conflictPanic struct {
	targetNode *paramNode
	newName    string
	newType    nodeType
}

func (n *paramNode) checkConflict_param(paramName string) {
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

func (n *paramNode) checkConflict_static(str rune) {
	if n.nType == static {
		if !(len(n.children) == 1 && n.children[0].nType == param) {
			return
		}
	}
	if n.nType == param {
		if str == '/' {
			return
		}
	}
	panic(conflictPanic{
		targetNode: n,
		newName:    string(str),
		newType:    static,
	})
}

func (n *paramNode) retrieve(path string) Handle {
	return n.retrieve_rune([]rune(path))
}

func (n *paramNode) retrieve_rune(path []rune) Handle {
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
			return child.retrieve_rune(path[end:])
		}
	}

	return nil
}
