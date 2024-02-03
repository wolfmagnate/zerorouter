// Copyright 2024 進捗ゼミ. All rights reserved.
// Based on the path package, Copyright 2009 The Go Authors.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package chapter2

import (
	"net/http"
	"testing"
)

func catchPanic(testFunc func()) (recv interface{}) {
	defer func() {
		recv = recover()
	}()

	testFunc()
	return
}

var fakeHandlerValue string

func fakeHandler(val string) Handle {
	return func(http.ResponseWriter, *http.Request) {
		fakeHandlerValue = val
	}
}

func checkNode(t *testing.T, n *paramNode, expectedStr string, expectedChildren int) []*paramNode {
	if n.nType != static {
		t.Errorf("Expected node type to be static, got %v", n.nType)
	}
	if n.path != expectedStr {
		t.Errorf("Expected string %s not found in node, got %s", expectedStr, n.path)
	}
	if len(n.children) != expectedChildren {
		t.Errorf("Expected %d children, got %d", expectedChildren, len(n.children))
	}
	if expectedChildren > 0 {
		return n.children
	}
	return nil
}

func checkHandler(t *testing.T, n *paramNode, expectedValue string) {
	if n.handle == nil {
		t.Error("Expected handle at the last node to be non-nil")
	} else {
		n.handle(nil, nil)
		if fakeHandlerValue != expectedValue {
			t.Errorf("Expected fakeHandlerValue to be '%s', got '%s'", expectedValue, fakeHandlerValue)
		}
	}
}

func checkParamNode(t *testing.T, n *paramNode, expectedParam string, expectedChildren int) []*paramNode {
	if n.nType != param {
		t.Errorf("Expected node type to be param, got %v", n.nType)
	}

	if n.path != expectedParam {
		t.Errorf("Expected path parameter %s, got %s", expectedParam, n.path)
	}

	if len(n.children) != expectedChildren {
		t.Errorf("Expected %d children, got %d for path parameter %s", expectedChildren, len(n.children), expectedParam)
	}

	return n.children
}

func checkConflict(t *testing.T, recv any, parentType nodeType, parentName string, childType nodeType, childName string) {
	conflict := recv.(conflictPanic)
	if conflict.newName != childName {
		t.Errorf("Expected childName to be '%s', got '%s'", childName, conflict.newName)
	}
	if conflict.newType != childType {
		t.Errorf("Expected childType to be '%v', got '%v'", childType, conflict.newType)
	}
	if conflict.targetNode.path != parentName {
		t.Errorf("Expected parentName to be '%s', got '%s'", parentName, conflict.targetNode.path)
	}
	if conflict.targetNode.nType != parentType {
		t.Errorf("Expected parentType to be '%v', got '%v'", parentType, conflict.targetNode.nType)
	}
}

// 親：static
// 子：static
func TestCheckConflict_1(t *testing.T) {
	p := &paramNode{
		path:  "a",
		nType: static,
	}
	p.children = make([]*paramNode, 0)
	p.checkConflict_static('b')
}

// 親：static
// 子：param
func TestCheckConflict_2(t *testing.T) {
	p := &paramNode{
		path:  "a",
		nType: static,
	}
	p.children = make([]*paramNode, 0)
	p.checkConflict_param(":path")
}

// 親：static
// 子：複数のstatic
func TestCheckConflict_3(t *testing.T) {
	p := &paramNode{
		path:  "a",
		nType: static,
	}
	p.children = make([]*paramNode, 0)
	c1 := &paramNode{
		path:  "b",
		nType: static,
	}
	p.children = append(p.children, c1)
	p.checkConflict_static('c')
}

// 親：static
// 子：static, param
func TestCheckConflict_4(t *testing.T) {
	p := &paramNode{
		path:  "a",
		nType: static,
	}
	p.children = make([]*paramNode, 0)
	c1 := &paramNode{
		path:  "b",
		nType: static,
	}
	p.children = append(p.children, c1)
	recv := catchPanic(func() {
		p.checkConflict_param(":path")
	})
	checkConflict(t, recv, static, "a", param, ":path")
}

// 親：static
// 子：static, param
func TestCheckConflict_5(t *testing.T) {
	p := &paramNode{
		path:  "a",
		nType: static,
	}
	p.children = make([]*paramNode, 0)
	c1 := &paramNode{
		path:  ":path",
		nType: param,
	}
	p.children = append(p.children, c1)
	p.checkConflict_param(":path")
	recv := catchPanic(func() {
		p.checkConflict_param(":another")
	})
	checkConflict(t, recv, static, "a", param, ":another")
}

// 親：param
// 子：static
func TestCheckConflict_6(t *testing.T) {
	p := &paramNode{
		path:  ":path",
		nType: param,
	}
	p.children = make([]*paramNode, 0)
	p.checkConflict_static('/')
	recv := catchPanic(func() {
		p.checkConflict_static('a')
	})
	checkConflict(t, recv, param, ":path", static, "a")
}

// 親：param
// 子：param
func TestCheckConflict_7(t *testing.T) {
	p := &paramNode{
		path:  ":path",
		nType: param,
	}
	p.children = make([]*paramNode, 0)
	recv := catchPanic(func() {
		p.checkConflict_param(":another")
	})
	checkConflict(t, recv, param, ":path", param, ":another")
}

// 親：param
// 子：static, param
func TestCheckConflict_8(t *testing.T) {
	p := &paramNode{
		path:  ":path",
		nType: param,
	}
	p.children = make([]*paramNode, 0)
	p.checkConflict_static('/')
	c := &paramNode{
		path:  "/",
		nType: static,
	}
	p.children = append(p.children, c)
	recv := catchPanic(func() {
		p.checkConflict_param(":path2")
	})
	checkConflict(t, recv, param, ":path", param, ":path2")
}

func TestAddRoute_SinglePath(t *testing.T) {
	n := &paramNode{}
	n.addRoute("/ab/cd", fakeHandler("dummy"))

	n = checkNode(t, n, "", 1)[0]
	n = checkNode(t, n, "/", 1)[0]
	n = checkNode(t, n, "a", 1)[0]
	n = checkNode(t, n, "b", 1)[0]
	n = checkNode(t, n, "/", 1)[0]
	n = checkNode(t, n, "c", 1)[0]
	_ = checkNode(t, n, "d", 0)

	checkHandler(t, n, "dummy")
}

func TestAddRoute_MultiPath(t *testing.T) {
	n := &paramNode{}
	n.addRoute("/ab/cd", fakeHandler("dummy1"))
	n.addRoute("/ab/ed", fakeHandler("dummy2"))

	n = checkNode(t, n, "", 1)[0]
	n = checkNode(t, n, "/", 1)[0]
	n = checkNode(t, n, "a", 1)[0]
	n = checkNode(t, n, "b", 1)[0]
	branch := checkNode(t, n, "/", 2)

	n1 := branch[0]
	n1 = checkNode(t, n1, "c", 1)[0]
	_ = checkNode(t, n1, "d", 0)
	checkHandler(t, n1, "dummy1")

	n2 := branch[1]
	n2 = checkNode(t, n2, "e", 1)[0]
	_ = checkNode(t, n2, "d", 0)
	checkHandler(t, n2, "dummy2")
}

func TestAddRoute_SingleParam(t *testing.T) {
	n := &paramNode{}
	n.addRoute("/ab/:path/xyz", fakeHandler("dummy"))

	n = checkNode(t, n, "", 1)[0]
	n = checkNode(t, n, "/", 1)[0]
	n = checkNode(t, n, "a", 1)[0]
	n = checkNode(t, n, "b", 1)[0]
	n = checkNode(t, n, "/", 1)[0]
	n = checkParamNode(t, n, ":path", 1)[0]
	n = checkNode(t, n, "/", 1)[0]
	n = checkNode(t, n, "x", 1)[0]
	n = checkNode(t, n, "y", 1)[0]
	_ = checkNode(t, n, "z", 0)

	checkHandler(t, n, "dummy")
}

func TestAddRoute_MultiParam(t *testing.T) {
	n := &paramNode{}
	n.addRoute("/ab/:path1/xyz", fakeHandler("dummy1"))
	n.addRoute("/a/:path2", fakeHandler("dummy2"))

	n = checkNode(t, n, "", 1)[0]
	n = checkNode(t, n, "/", 1)[0]
	branch := checkNode(t, n, "a", 2)

	n1 := branch[0]
	n1 = checkNode(t, n1, "b", 1)[0]
	n1 = checkNode(t, n1, "/", 1)[0]
	n1 = checkParamNode(t, n1, ":path1", 1)[0]
	n1 = checkNode(t, n1, "/", 1)[0]
	n1 = checkNode(t, n1, "x", 1)[0]
	n1 = checkNode(t, n1, "y", 1)[0]
	_ = checkNode(t, n1, "z", 0)
	checkHandler(t, n1, "dummy1")

	n2 := branch[1]
	n2 = checkNode(t, n2, "/", 1)[0]
	_ = checkParamNode(t, n2, ":path2", 0)
	checkHandler(t, n2, "dummy2")
}

func TestAddRoute_MultiPathForSingleParam(t *testing.T) {
	n := &paramNode{}
	n.addRoute("/ab/:path/xyz", fakeHandler("dummy1"))
	n.addRoute("/ab/:path/mn", fakeHandler("dummy2"))

	n = checkNode(t, n, "", 1)[0]
	n = checkNode(t, n, "/", 1)[0]
	n = checkNode(t, n, "a", 1)[0]
	n = checkNode(t, n, "b", 1)[0]
	n = checkNode(t, n, "/", 1)[0]
	n = checkParamNode(t, n, ":path", 1)[0]
	branch := checkNode(t, n, "/", 2)

	n1 := branch[0]
	n1 = checkNode(t, n1, "x", 1)[0]
	n1 = checkNode(t, n1, "y", 1)[0]
	_ = checkNode(t, n1, "z", 0)
	checkHandler(t, n1, "dummy1")

	n2 := branch[1]
	n2 = checkNode(t, n2, "m", 1)[0]
	_ = checkNode(t, n2, "n", 0)
	checkHandler(t, n2, "dummy2")
}

func TestRetrieve(t *testing.T) {
	n := &paramNode{}
	n.addRoute("/x/:user/y", fakeHandler("dummy1"))
	n.addRoute("/xy:id/z", fakeHandler("dummy2"))
	n.addRoute("/a/:id/:name", fakeHandler("dummy3"))

	tests := []struct {
		path          string
		expectedValue string
	}{
		{"/x", ""},
		{"/x/taro/y", "dummy1"},
		{"/x/taro/z/y", ""},
		{"/xyz/z", "dummy2"},
		{"/xyloooooong/z", "dummy2"},
		{"/a//", ""},
		{"/a//hello", ""},
		{"/a/b/c", "dummy3"},
	}

	for _, test := range tests {
		handler := n.retrieve(test.path)
		if handler == nil {
			if test.expectedValue != "" {
				t.Errorf("retrieve(%s) = nil, want %s", test.path, test.expectedValue)
			}
		} else {
			handler(nil, nil)
			if fakeHandlerValue != test.expectedValue {
				t.Errorf("retrieve(%s) handler set fakeHandlerValue = %s, want %s", test.path, fakeHandlerValue, test.expectedValue)
			}
		}
	}
}
