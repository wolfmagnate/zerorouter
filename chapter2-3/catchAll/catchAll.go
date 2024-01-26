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
	if len(path) < 2 || path[0] != '*' || string(path) == "*/" {
		return "", 0, fmt.Errorf("invalid catch-all parameter")
	}

	for i := 1; i < len(path); i++ {
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
			panic("invalid path parameter")
		}
		n.checkConflict_param(paramName)
		n.handle_param(path, paramName, paramLen, handle)
	case "/":
		if len(path) == 1 || string(path[1]) != "*" {
			n.checkConflict_static(path[0])
			n.handle_static(path, handle)
			return
		}
		catchAllName, pathLen, err := extractCatchAll(path[1:])
		if err != nil {
			panic("invalid path parameter")
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
		// checkConflictを通過したので子供は必ず同名のparamである
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
			child.addRoute_rune(path[1+pathLen:], handle)
			return
		}
	}
	nextNode := &catchAllNode{
		path:  "/" + catchAllName,
		nType: catchAll,
	}
	n.children = append(n.children, nextNode)
	nextNode.addRoute_rune(path[1+pathLen:], handle)
}

// ワイルドカードの制約により通常のノードを追加することが出来ないパターンを認識してpanicする
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
		// 1. 親が文字ノードで、子に文字ノードしかない場合
		if !(containsParam || containsCatchAll) {
			return
		}
		// 2. 親が文字ノードで、ちょうど1つのキャッチオールノードをもち、スラッシュ以外の文字ノードが子である場合
		// 木の構造は正しいので、containsCatchAllがtrueならば自動的にcontainsParamはfalseである
		// 木の構造は正しいので、containsCatchAllがtrueならばキャッチオールノードの個数はちょうど1つである
		if containsCatchAll && str != '/' {
			return
		}
	}

	// 3. 親がパラメータノードで、動的パターンを持たず、子がスラッシュである場合
	if n.nType == param && str == '/' {
		// 木の構造は正しいので、以下の考察が出来る
		// 1. パラメータノードの子は0または1である
		// 2. パラメータノードの子の動的パターンはキャッチオールノードのみ
		// 3. パラメータノードの子の静的パターンはスラッシュの文字ノードのみ
		if len(n.children) == 0 {
			return
		}
		if n.children[0].nType == static {
			return
		}
	}

	panic("Conflict with existing route")
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
			if c.nType == catchAll && c.children != nil && c.children[0].path != catchAllName {
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
	panic("Conflict with existing route")
}

func (n *catchAllNode) checkConflict_param(paramName string) {
	if len(n.children) == 1 && n.children[0].nType == param && n.children[0].path == paramName {
		return
	}
	if n.nType == static && len(n.children) == 0 {
		return
	}
	panic("Conflict with existing param")
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
