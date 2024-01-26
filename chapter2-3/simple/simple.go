// Copyright 2024 進捗ゼミ. All rights reserved.
// Based on the path package, Copyright 2009 The Go Authors.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package chapter2

import "net/http"

type Handle func(http.ResponseWriter, *http.Request)

type simpleNode struct {
	path     rune
	children []*simpleNode
	handle   Handle
}

// nをルートとしたtrie木を構築する
// pathに対応した葉ノードと、ルートから葉までのノードとエッジを追加する
// pathに対応する葉にhandleを追加する
// pathは空文字以外の文字列
func (n *simpleNode) addRoute(path string, handle Handle) {
	n.addRoute_rune([]rune(path), handle)
}

func (n *simpleNode) addRoute_rune(path []rune, handle Handle) {
	if len(path) == 0 {
		n.handle = handle
		return
	}

	if n.children == nil {
		n.children = make([]*simpleNode, 0)
	}
	next := path[0]

	for _, c := range n.children {
		if c.path == next {
			c.addRoute_rune(path[1:], handle)
			return
		}
	}

	next_node := &simpleNode{
		path: next,
	}
	n.children = append(n.children, next_node)
	next_node.addRoute_rune(path[1:], handle)
}

// nをルートとする木について、pathに対応するHandleを検索する
// 見つからなかった場合にはnilを返す
func (n *simpleNode) retrieve(path string) Handle {
	return n.retrieve_rune([]rune(path))
}

func (n *simpleNode) retrieve_rune(path []rune) Handle {
	if len(path) == 0 {
		return n.handle
	}

	next := path[0]

	for _, child := range n.children {
		if child.path == next {
			return child.retrieve_rune(path[1:])
		}
	}

	return nil
}
