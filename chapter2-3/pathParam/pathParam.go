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

type pathparamNode struct {
	// パスパラメータは複数文字にわたる可能性があるため、stringに変更
	path     string
	children []*pathparamNode
	// パスパラメータor通常の文字 を判定するフラグ
	nType  nodeType
	handle Handle
}

func extractPathparam(path []rune) (string, int, error) {
	if len(path) < 2 || path[0] != ':' || string(path) == ":/" {
		return "", 0, fmt.Errorf("invalid path parameter")
	}

	for i := 1; i < len(path); i++ {
		if path[i] == '/' {
			return string(path[:i]), i, nil
		}
	}
	return string(path), len(path), nil
}

func (n *pathparamNode) addRoute(path string, handle Handle) {
	n.addRoute_rune([]rune(path), handle)
}

func (n *pathparamNode) addRoute_rune(path []rune, handle Handle) {
	if len(path) == 0 {
		n.handle = handle
		return
	}

	if n.children == nil {
		n.children = make([]*pathparamNode, 0)
	}

	if string(path[0]) == ":" {
		pathparam, pathLen, err := extractPathparam(path)
		if err != nil {
			panic("Invalid path parameter")
		}
		n.checkWildcardConflict_Pathparam(pathparam)
		n.handlePathparam(path, pathparam, pathLen, handle)
	} else {
		n.checkWildcardConflict()
		n.handleRegularPath(path, handle)
	}
}

func (n *pathparamNode) handlePathparam(path []rune, pathparam string, pathLen int, handle Handle) {
	if len(n.children) == 0 {
		nextNode := &pathparamNode{
			path:  pathparam,
			nType: param,
		}
		n.children = append(n.children, nextNode)
		nextNode.addRoute_rune(path[pathLen:], handle)
	} else {
		n.children[0].addRoute_rune(path[pathLen:], handle)
	}
}

func (n *pathparamNode) handleRegularPath(path []rune, handle Handle) {
	next := string(path[0])

	for _, child := range n.children {
		if child.path == next {
			child.addRoute_rune(path[1:], handle)
			return
		}
	}

	nextNode := &pathparamNode{
		path: next,
	}
	n.children = append(n.children, nextNode)
	nextNode.addRoute_rune(path[1:], handle)
}

func (n *pathparamNode) checkWildcardConflict_Pathparam(pathparam string) {
	if len(n.children) == 1 && n.children[0].nType == param && n.children[0].path == pathparam {
		// 特例的に耐えるパターン
		return
	}
	// 基本的に子供がいるノードにパスパラメータは指定できない
	if len(n.children) > 0 {
		panic("Conflict with existing path parameter")
	}
}

func (n *pathparamNode) checkWildcardConflict() {
	// 既にパスパラメータの子供がいる場合を除けば必ず追加できる
	if len(n.children) == 1 && n.children[0].nType == param {
		panic("Cannot insert a static node when a path parameter node is present")
	}
}

func (n *pathparamNode) retrieve(path string) Handle {
	return n.retrieve_rune([]rune(path))
}

func (n *pathparamNode) retrieve_rune(path []rune) Handle {
	if len(path) == 0 {
		return n.handle
	}

	next := string(path[0])

	for _, child := range n.children {
		if child.nType == static {
			if child.path == next {
				return child.retrieve_rune(path[1:])
			}
		}
		if n.children[0].nType == param {
			end := 1
			for end < len(path) && path[end] != '/' {
				end++
			}
			return n.children[0].retrieve_rune(path[end:])
		}
	}

	return nil
}
