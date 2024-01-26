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
	// パスパラメータは複数文字にわたる可能性があるため、stringに変更
	path     string
	children []*catchAllNode
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
		if path[i] == '*' || path[i] == ':' {
			return "", 0, fmt.Errorf("Invalid catchAll is in pathparam name")
		}
	}
	return string(path), len(path), nil
}
func extractCatchAll(path []rune) (string, int, error) {
	if len(path) < 2 || path[0] != '*' || string(path) == "*/" {
		return "", 0, fmt.Errorf("invalid catch-all parameter")
	}

	for i := 1; i < len(path); i++ {
		if path[i] == '/' {
			return string(path[:i]), i, nil
		}
		if path[i] == '*' || path[i] == ':' {
			return "", 0, fmt.Errorf("Invalid pathparam is in catchall name")
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
		pathparam, pathLen, err := extractPathparam(path)
		if err != nil {
			panic("Invalid path parameter")
		}
		n.checkWildcardConflict_Pathparam(pathparam)
		n.handlePathparam(path, pathparam, pathLen, handle)
	case "/":
		if string(path[1]) != "*" {
			n.checkWildcardConflict()
			n.handleRegularPath(path, handle)
			return
		}
		catchAll, pathLen, err := extractCatchAll(path[1:])
		if err != nil {
			panic("Invalid path parameter")
		}
		n.checkWildcardConflict_CatchAll()
		n.handleCatchAll(path, catchAll, pathLen, handle)
	default:
		n.checkWildcardConflict()
		n.handleRegularPath(path, handle)
	}
}

func (n *catchAllNode) handleRegularPath(path []rune, handle Handle) {
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

func (n *catchAllNode) handlePathparam(path []rune, pathparam string, pathLen int, handle Handle) {
	if len(n.children) == 0 {
		nextNode := &catchAllNode{
			path:  pathparam,
			nType: param,
		}
		n.children = append(n.children, nextNode)
		nextNode.addRoute_rune(path[pathLen:], handle)
	} else {
		n.children[0].addRoute_rune(path[pathLen:], handle)
	}
}

func (n *catchAllNode) handleCatchAll(path []rune, catchAllPath string, pathLen int, handle Handle) {
	nextNode := &catchAllNode{
		path:  "/" + catchAllPath,
		nType: catchAll,
	}
	n.children = append(n.children, nextNode)
	nextNode.addRoute_rune(path[1+pathLen:], handle)
}

// ワイルドカードの制約により通常のノードを追加することが出来ないパターンを認識してpanicする
func (n *catchAllNode) checkWildcardConflict() {
	// 配下にワイルドカードがあるノードには追加できない
	if len(n.children) == 1 && (n.children[0].nType == param || n.children[0].nType == catchAll) {
		panic("Cannot insert a static node when a wildcard node is present")
	}
	// 注意点3. catch-allノードには何も追加できない
	if n.nType == catchAll {
		panic("Cannot insert a node into catch-all node")
	}
}

// ワイルドカードの制約によりCatch-Allノードを追加することが出来ないパターンを認識してpanicする
func (n *catchAllNode) checkWildcardConflict_CatchAll() {
	// 注意点：この関数がチェックしているのは、"/*wildcard"の直前のノード
	if n.nType == static || n.nType == param {
		if len(n.children) == 0 {
			return
		}
		if len(n.children) == 1 && (n.children[0].nType == catchAll || n.children[0].nType == param) {
			panic("ダメ～")
		}
		// 通常の子供でも"/"がいた場合は耐えない
		for _, child := range n.children {
			if child.path == "/" {
				panic("ダメ～")
			}
		}
	}
	if n.nType == catchAll {
		panic("Cannot insert a node into catch-all node")
	}
}

// ワイルドカードの制約によりparamノードを追加することが出来ないパターンを認識してpanicする
func (n *catchAllNode) checkWildcardConflict_Pathparam(pathparam string) {
	if len(n.children) == 1 && n.children[0].nType == param && n.children[0].path == pathparam {
		// 特例的に耐えるパターン
		return
	}
	// 基本的に子供がいるノードにパスパラメータは指定できない
	if len(n.children) > 0 {
		panic("Conflict with existing path parameter")
	}
	// 注意点3. catch-allノードには何も追加できない
	if n.nType == catchAll {
		panic("Cannot insert a node into catch-all node")
	}
	// パスパラメータの直後のパスパラメータは無理（パラメータのパースの時点で失敗する）
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
		if child.nType == catchAll {
			if next == "/" {
				// path全域にマッチする
				return child.retrieve_rune(path[len(path):])
			}
		}
	}

	return nil
}
