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

func checkNode(t *testing.T, n *catchAllNode, expectedStr string, expectedChildren int) []*catchAllNode {
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

func checkParamNode(t *testing.T, n *catchAllNode, expectedParam string, expectedChildren int) []*catchAllNode {
	if n.nType != param {
		t.Errorf("Expected node type to be param, got %v", n.nType)
	}
	if n.path != expectedParam {
		t.Errorf("Expected param %s, got %s", expectedParam, n.path)
	}
	if len(n.children) != expectedChildren {
		t.Errorf("Expected %d children, got %d for path parameter %s", expectedChildren, len(n.children), expectedParam)
	}
	return n.children
}

func checkCatchAllNode(t *testing.T, n *catchAllNode, expectedCatchAll string) []*catchAllNode {
	if n.nType != catchAll {
		t.Errorf("Expected node type to be catchAll, got %v", n.nType)
	}
	if n.path != expectedCatchAll {
		t.Errorf("Expected catchAll %s, got %s", expectedCatchAll, n.path)
	}
	if len(n.children) != 0 {
		t.Errorf("Expected no children, got %d for catchAll %s", len(n.children), expectedCatchAll)
	}
	return n.children
}

func checkHandler(t *testing.T, n *catchAllNode, expectedValue string) {
	if n.handle == nil {
		t.Error("Expected handle at the last node to be non-nil")
	} else {
		n.handle(nil, nil)
		if fakeHandlerValue != expectedValue {
			t.Errorf("Expected fakeHandlerValue to be '%s', got '%s'", expectedValue, fakeHandlerValue)
		}
	}
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

// テストケース
// 3つの制約判定について7パターンについてちゃんと全部正しい動作をすること
func TestCheckConflict_1(t *testing.T) {
	p := &catchAllNode{
		path:  "a",
		nType: static,
	}
	p.children = make([]*catchAllNode, 0)
	p.checkConflict_static('b')
}
func TestCheckConflict_2(t *testing.T) {
	p := &catchAllNode{
		path:  "a",
		nType: static,
	}
	p.children = make([]*catchAllNode, 0)
	p.checkConflict_param(":path")
}

func TestCheckConflict_3(t *testing.T) {
	p := &catchAllNode{
		path:  "a",
		nType: static,
	}
	p.children = make([]*catchAllNode, 0)
	p.checkConflict_catchAll("/*everything")
}

func TestCheckConflict_4(t *testing.T) {
	p := &catchAllNode{
		path:  "a",
		nType: static,
	}
	p.children = make([]*catchAllNode, 0)
	c1 := &catchAllNode{
		path:  "b",
		nType: static,
	}
	p.children = append(p.children, c1)
	p.checkConflict_static('c')
}

func TestCheckConflict_5(t *testing.T) {
	p := &catchAllNode{
		path:  "a",
		nType: static,
	}
	p.children = make([]*catchAllNode, 0)
	c1 := &catchAllNode{
		path:  "b",
		nType: static,
	}
	p.children = append(p.children, c1)
	recv := catchPanic(func() {
		p.checkConflict_param(":path")
	})
	checkConflict(t, recv, static, "a", param, ":path")
}

func TestCheckConflict_6(t *testing.T) {
	p := &catchAllNode{
		path:  "a",
		nType: static,
	}
	p.children = make([]*catchAllNode, 0)
	c1 := &catchAllNode{
		path:  "b",
		nType: static,
	}
	p.children = append(p.children, c1)
	p.checkConflict_catchAll("/*everything")
}

func TestCheckConflict_7(t *testing.T) {
	p := &catchAllNode{
		path:  "a",
		nType: static,
	}
	p.children = make([]*catchAllNode, 0)
	c1 := &catchAllNode{
		path:  "/",
		nType: static,
	}
	p.children = append(p.children, c1)
	recv := catchPanic(func() {
		p.checkConflict_catchAll("/*everything")
	})
	checkConflict(t, recv, static, "a", catchAll, "/*everything")
}

func TestCheckConflict_8(t *testing.T) {
	p := &catchAllNode{
		path:  "a",
		nType: static,
	}
	p.children = make([]*catchAllNode, 0)
	c1 := &catchAllNode{
		path:  ":path",
		nType: param,
	}
	p.children = append(p.children, c1)
	recv := catchPanic(func() {
		p.checkConflict_catchAll("/*everything")
	})
	checkConflict(t, recv, static, "a", catchAll, "/*everything")
}

func TestCheckConflict_9(t *testing.T) {
	p := &catchAllNode{
		path:  "a",
		nType: static,
	}
	p.children = make([]*catchAllNode, 0)
	c1 := &catchAllNode{
		path:  "b",
		nType: static,
	}
	p.children = append(p.children, c1)
	p.checkConflict_catchAll("/*everything")
	c2 := &catchAllNode{
		path:  "/*everything",
		nType: catchAll,
	}
	p.children = append(p.children, c2)
	recv := catchPanic(func() {
		p.checkConflict_param(":path")
	})
	checkConflict(t, recv, static, "a", param, ":path")
}
func TestCheckConflict_10(t *testing.T) {
	p := &catchAllNode{
		path:  "a",
		nType: static,
	}
	p.children = make([]*catchAllNode, 0)
	c1 := &catchAllNode{
		path:  "b",
		nType: static,
	}
	p.children = append(p.children, c1)
	p.checkConflict_catchAll("/*everything")
	c2 := &catchAllNode{
		path:  "/*everything",
		nType: catchAll,
	}
	p.children = append(p.children, c2)
	p.checkConflict_catchAll("/*everything")
	recv := catchPanic(func() {
		p.checkConflict_catchAll("/*everything2")
	})
	checkConflict(t, recv, static, "a", catchAll, "/*everything2")
}

func TestCheckConflict_11(t *testing.T) {
	p := &catchAllNode{
		path:  "a",
		nType: static,
	}
	p.children = make([]*catchAllNode, 0)
	c1 := &catchAllNode{
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
func TestCheckConflict_12(t *testing.T) {
	p := &catchAllNode{
		path:  ":path",
		nType: param,
	}
	p.children = make([]*catchAllNode, 0)
	p.checkConflict_static('/')
	recv := catchPanic(func() {
		p.checkConflict_static('a')
	})
	checkConflict(t, recv, param, ":path", static, "a")
}

// 親：param
// 子：param
func TestCheckConflict_13(t *testing.T) {
	p := &catchAllNode{
		path:  ":path",
		nType: param,
	}
	p.children = make([]*catchAllNode, 0)
	recv := catchPanic(func() {
		p.checkConflict_param(":another")
	})
	checkConflict(t, recv, param, ":path", param, ":another")
}

// 親：param
// 子：catchAll
func TestCheckConflict_14(t *testing.T) {
	p := &catchAllNode{
		path:  ":path",
		nType: param,
	}
	p.children = make([]*catchAllNode, 0)
	p.checkConflict_catchAll("/*everything")
	c := &catchAllNode{
		path:  "/*everything",
		nType: catchAll,
	}
	p.children = append(p.children, c)
	recv := catchPanic(func() {
		p.checkConflict_catchAll("/*another")
	})
	checkConflict(t, recv, param, ":path", catchAll, "/*another")
}

// 親：param
// 子：static, param
func TestCheckConflict_15(t *testing.T) {
	p := &catchAllNode{
		path:  ":path",
		nType: param,
	}
	p.children = make([]*catchAllNode, 0)
	p.checkConflict_static('/')
	c := &catchAllNode{
		path:  "/",
		nType: static,
	}
	p.children = append(p.children, c)
	recv := catchPanic(func() {
		p.checkConflict_param(":path2")
	})
	checkConflict(t, recv, param, ":path", param, ":path2")
}

// 親：param
// 子：static, catchAll
func TestCheckConflict_16(t *testing.T) {
	p := &catchAllNode{
		path:  ":path",
		nType: param,
	}
	p.children = make([]*catchAllNode, 0)
	p.checkConflict_static('/')
	c := &catchAllNode{
		path:  "/",
		nType: static,
	}
	p.children = append(p.children, c)
	recv := catchPanic(func() {
		p.checkConflict_catchAll("/*everything")
	})
	checkConflict(t, recv, param, ":path", catchAll, "/*everything")
}

// 親：param
// 子：param, catchAll
func TestCheckConflict_17(t *testing.T) {
	// paramの子にparamが来るとNGなので自明にNG
	// c.f. TestCheckConflict_13
	p := &catchAllNode{
		path:  ":path",
		nType: param,
	}
	p.children = make([]*catchAllNode, 0)
	c := &catchAllNode{
		path:  "/*everything",
		nType: catchAll,
	}
	p.children = append(p.children, c)
	recv := catchPanic(func() {
		p.checkConflict_param(":something")
	})
	checkConflict(t, recv, param, ":path", param, ":something")
}

// 親：param
// 子：static, param, catchAll
func TestCheckConflict_18(t *testing.T) {
	// paramの子にparamが来るとNGなので自明にNG
	// c.f. TestCheckConflict_13

	// staticとcatchAllは同時にparamの子になれないので自明にNG
	// c.f. TestCheckConflict_16
}

// 親：catchAll
// 子：static
func TestCheckConflict_19(t *testing.T) {
	p := &catchAllNode{
		path:  "/*everything",
		nType: catchAll,
	}
	recv := catchPanic(func() {
		p.checkConflict_static('a')
	})
	checkConflict(t, recv, catchAll, "/*everything", static, "a")
}

// 親：catchAll
// 子：param
func TestCheckConflict_20(t *testing.T) {
	p := &catchAllNode{
		path:  "/*everything",
		nType: catchAll,
	}
	recv := catchPanic(func() {
		p.checkConflict_param(":path")
	})
	checkConflict(t, recv, catchAll, "/*everything", param, ":path")
}

// 親：catchAll
// 子：catchAll
func TestCheckConflict_21(t *testing.T) {
	p := &catchAllNode{
		path:  "/*everything",
		nType: catchAll,
	}

	recv := catchPanic(func() {
		p.checkConflict_catchAll("/*anypath")
	})
	checkConflict(t, recv, catchAll, "/*everything", catchAll, "/*anypath")
}

// テストケース
// 1. 単純なパターン
func TestAddRoute_CatchAll_1(t *testing.T) {
	n := &catchAllNode{}
	n.addRoute("/a/*everything", fakeHandler("dummy"))

	n = checkNode(t, n, "", 1)[0]
	n = checkNode(t, n, "/", 1)[0]
	n = checkNode(t, n, "a", 1)[0]
	_ = checkCatchAllNode(t, n, "/*everything")

	checkHandler(t, n, "dummy")
}

// 2. パラメータと同居
// /b/:pathと/a/*everything
func TestAddRoute_CatchAll_2(t *testing.T) {
	n := &catchAllNode{}
	n.addRoute("/a/:path", fakeHandler("dummy1"))
	n.addRoute("/b/*everything", fakeHandler("dummy2"))

	n = checkNode(t, n, "", 1)[0]
	branch := checkNode(t, n, "/", 2)

	n1 := branch[0]
	n1 = checkNode(t, n1, "a", 1)[0]
	n1 = checkNode(t, n1, "/", 1)[0]
	_ = checkParamNode(t, n1, ":path", 0)
	checkHandler(t, n1, "dummy1")

	n2 := branch[1]
	n2 = checkNode(t, n2, "b", 1)[0]
	_ = checkCatchAllNode(t, n2, "/*everything")
	checkHandler(t, n2, "dummy2")
}

// /a/:path/*everything
func TestAddRoute_CatchAll_3(t *testing.T) {
	n := &catchAllNode{}
	n.addRoute("/a/:path/*everything", fakeHandler("dummy"))

	n = checkNode(t, n, "", 1)[0]
	n = checkNode(t, n, "/", 1)[0]
	n = checkNode(t, n, "a", 1)[0]
	n = checkNode(t, n, "/", 1)[0]
	n = checkParamNode(t, n, ":path", 1)[0]
	_ = checkCatchAllNode(t, n, "/*everything")

	checkHandler(t, n, "dummy")
}

// 3. 文字ノードの配下にキャッチオールとスラッシュ以外の文字ノード
// /abcと/a/*everythingと/a
// /a/*everythingと/a:user/b
func TestAddRoute_CatchAll_4(t *testing.T) {
	n := &catchAllNode{}
	n.addRoute("/abc", fakeHandler("dummy1"))
	n.addRoute("/a/*everything", fakeHandler("dummy2"))
	n.addRoute("/a", fakeHandler("dummy3"))

	n = checkNode(t, n, "", 1)[0]
	n = checkNode(t, n, "/", 1)[0]
	checkHandler(t, n, "dummy3")
	branch := checkNode(t, n, "a", 2)

	n1 := branch[0]
	n1 = checkNode(t, n1, "b", 1)[0]
	_ = checkNode(t, n1, "c", 0)
	checkHandler(t, n1, "dummy1")

	n2 := branch[1]
	_ = checkCatchAllNode(t, n2, "/*everything")
	checkHandler(t, n2, "dummy2")
}

// 4. パラメータと近接
func TestAddRoute_CatchAll_5(t *testing.T) {
	n := &catchAllNode{}
	n.addRoute("/xy:path", fakeHandler("dummy1"))
	n.addRoute("/x/*everything", fakeHandler("dummy2"))

	n = checkNode(t, n, "", 1)[0]
	n = checkNode(t, n, "/", 1)[0]
	branch := checkNode(t, n, "x", 2)

	n1 := branch[0]
	n1 = checkNode(t, n1, "y", 1)[0]
	_ = checkParamNode(t, n1, ":path", 0)
	checkHandler(t, n1, "dummy1")

	n2 := branch[1]
	_ = checkCatchAllNode(t, n2, "/*everything")
	checkHandler(t, n2, "dummy2")
}

// 5. キャッチオールに子があるパターン
func TestAddRoute_CatchAll_6(t *testing.T) {
	n := &catchAllNode{}
	recv := catchPanic(func() {
		n.addRoute("/a/*everything/b", fakeHandler("dummy"))
	})
	checkConflict(t, recv, catchAll, "/*everything", static, "/")
}

func TestAddRoute_CatchAll_7(t *testing.T) {
	n := &catchAllNode{}
	recv := catchPanic(func() {
		n.addRoute("/d/*everything:path", fakeHandler("dummy"))
	})
	if msg := recv.(string); msg == "invalid catchAll" {
		return
	}
	t.Errorf("Expected invalid catchAll, but got other panic")
}

// 6. 文字ノードの配下にキャッチオールとスラッシュを表す文字ノード
func TestAddRoute_CatchAll_8(t *testing.T) {
	n := &catchAllNode{}
	recv := catchPanic(func() {
		n.addRoute("/a/*everything", fakeHandler("dummy1"))
		n.addRoute("/a/b", fakeHandler("dummy2"))
	})
	checkConflict(t, recv, static, "a", static, "/")
}
func TestAddRoute_CatchAll_9(t *testing.T) {
	n := &catchAllNode{}
	recv := catchPanic(func() {
		n.addRoute("/a/*everything", fakeHandler("dummy1"))
		n.addRoute("/a:path", fakeHandler("dummy2"))
	})
	checkConflict(t, recv, static, "a", param, ":path")
}

// 7. パラメータの配下にキャッチオール
func TestAddRoute_CatchAll_10(t *testing.T) {
	n := &catchAllNode{}
	n.addRoute("/:path/*everything", fakeHandler("dummy"))

	n = checkNode(t, n, "", 1)[0]
	n = checkNode(t, n, "/", 1)[0]
	n = checkParamNode(t, n, ":path", 1)[0]
	_ = checkCatchAllNode(t, n, "/*everything")
	checkHandler(t, n, "dummy")
}

func TestRetrieve(t *testing.T) {
	n := &catchAllNode{}
	n.addRoute("/a", fakeHandler("dummy1"))
	n.addRoute("/a/:path", fakeHandler("dummy2"))
	n.addRoute("/a/:path/*everything", fakeHandler("dummy3"))
	n.addRoute("/x", fakeHandler("dummy4"))
	n.addRoute("/xy", fakeHandler("dummy5"))
	n.addRoute("/xz", fakeHandler("dummy6"))
	n.addRoute("/xz/*file", fakeHandler("dummy7"))
	n.addRoute("/xzz", fakeHandler("dummy8"))
	n.addRoute("/xy:id", fakeHandler("dummy9"))
	n.addRoute("/xy:id/n", fakeHandler("dummy10"))

	tests := []struct {
		path          string
		expectedValue string
	}{
		{"/a", "dummy1"},
		{"/a/012", "dummy2"},
		{"/a/012/yeah", "dummy3"},
		{"/a/b/yeah:good", "dummy3"},
		{"/x", "dummy4"},
		{"/xy", "dummy5"},
		{"/xz", "dummy6"},
		{"/xz/", "dummy7"},
		{"/xz/hoge/fuga", "dummy7"},
		{"/xzz", "dummy8"},
		{"/xyz", "dummy9"},
		{"/xyzzz/n", "dummy10"},
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
