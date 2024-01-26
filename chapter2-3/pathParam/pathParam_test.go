// Copyright 2024 進捗ゼミ. All rights reserved.
// Based on the path package, Copyright 2009 The Go Authors.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package chapter2

import (
	"net/http"
	"testing"
)

var fakeHandlerValue string

func fakeHandler(val string) Handle {
	return func(http.ResponseWriter, *http.Request) {
		fakeHandlerValue = val
	}
}

func checkNode_Pathparam(t *testing.T, n *pathparamNode, expectedStr string, expectedChildren int) []*pathparamNode {
	runes := []rune(expectedStr)
	if len(runes) > 1 {
		t.Errorf("Invalid expectedStr. It must have one rune.")
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

func checkHandler_Pathparam(t *testing.T, n *pathparamNode, expectedValue string) {
	if n.handle == nil {
		t.Error("Expected handle at the last node to be non-nil")
	} else {
		n.handle(nil, nil)
		if fakeHandlerValue != expectedValue {
			t.Errorf("Expected fakeHandlerValue to be '%s', got '%s'", expectedValue, fakeHandlerValue)
		}
	}
}

func checkParamNode_Pathparam(t *testing.T, n *pathparamNode, expectedParam string, expectedChildren int) []*pathparamNode {
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

func TestAddRoute_Pathparam_SinglePath(t *testing.T) {
	n := &pathparamNode{}
	n.addRoute("/ab/cd", fakeHandler("dummy"))

	n = checkNode_Pathparam(t, n, "", 1)[0]
	n = checkNode_Pathparam(t, n, "/", 1)[0]
	n = checkNode_Pathparam(t, n, "a", 1)[0]
	n = checkNode_Pathparam(t, n, "b", 1)[0]
	n = checkNode_Pathparam(t, n, "/", 1)[0]
	n = checkNode_Pathparam(t, n, "c", 1)[0]
	_ = checkNode_Pathparam(t, n, "d", 0)

	checkHandler_Pathparam(t, n, "dummy")
}

func TestAddRoute_Pathparam_MultiPath(t *testing.T) {
	n := &pathparamNode{}
	n.addRoute("/ab/cd", fakeHandler("dummy1"))
	n.addRoute("/ab/ed", fakeHandler("dummy2"))

	n = checkNode_Pathparam(t, n, "", 1)[0]
	n = checkNode_Pathparam(t, n, "/", 1)[0]
	n = checkNode_Pathparam(t, n, "a", 1)[0]
	n = checkNode_Pathparam(t, n, "b", 1)[0]
	branch := checkNode_Pathparam(t, n, "/", 2)

	n1 := branch[0]
	n1 = checkNode_Pathparam(t, n1, "c", 1)[0]
	_ = checkNode_Pathparam(t, n1, "d", 0)
	checkHandler_Pathparam(t, n1, "dummy1")

	n2 := branch[1]
	n2 = checkNode_Pathparam(t, n2, "e", 1)[0]
	_ = checkNode_Pathparam(t, n2, "d", 0)
	checkHandler_Pathparam(t, n2, "dummy2")
}

func TestAddRoute_Pathparam_SingleParam(t *testing.T) {
	n := &pathparamNode{}
	n.addRoute("/ab/:path/xyz", fakeHandler("dummy"))

	n = checkNode_Pathparam(t, n, "", 1)[0]
	n = checkNode_Pathparam(t, n, "/", 1)[0]
	n = checkNode_Pathparam(t, n, "a", 1)[0]
	n = checkNode_Pathparam(t, n, "b", 1)[0]
	n = checkNode_Pathparam(t, n, "/", 1)[0]
	n = checkParamNode_Pathparam(t, n, ":path", 1)[0]
	n = checkNode_Pathparam(t, n, "/", 1)[0]
	n = checkNode_Pathparam(t, n, "x", 1)[0]
	n = checkNode_Pathparam(t, n, "y", 1)[0]
	_ = checkNode_Pathparam(t, n, "z", 0)

	checkHandler_Pathparam(t, n, "dummy")
}

func TestAddRoute_Pathparam_MultiParam(t *testing.T) {
	n := &pathparamNode{}
	n.addRoute("/ab/:path1/xyz", fakeHandler("dummy1"))
	n.addRoute("/a/:path2", fakeHandler("dummy2"))

	n = checkNode_Pathparam(t, n, "", 1)[0]
	n = checkNode_Pathparam(t, n, "/", 1)[0]
	branch := checkNode_Pathparam(t, n, "a", 2)

	n1 := branch[0]
	n1 = checkNode_Pathparam(t, n1, "b", 1)[0]
	n1 = checkNode_Pathparam(t, n1, "/", 1)[0]
	n1 = checkParamNode_Pathparam(t, n1, ":path1", 1)[0]
	n1 = checkNode_Pathparam(t, n1, "/", 1)[0]
	n1 = checkNode_Pathparam(t, n1, "x", 1)[0]
	n1 = checkNode_Pathparam(t, n1, "y", 1)[0]
	_ = checkNode_Pathparam(t, n1, "z", 0)
	checkHandler_Pathparam(t, n1, "dummy1")

	n2 := branch[1]
	n2 = checkNode_Pathparam(t, n2, "/", 1)[0]
	_ = checkParamNode_Pathparam(t, n2, ":path2", 0)
	checkHandler_Pathparam(t, n2, "dummy2")
}

func TestAddRoute_Pathparam_MultiPathForSingleParam(t *testing.T) {
	n := &pathparamNode{}
	n.addRoute("/ab/:path/xyz", fakeHandler("dummy1"))
	n.addRoute("/ab/:path/mn", fakeHandler("dummy2"))

	n = checkNode_Pathparam(t, n, "", 1)[0]
	n = checkNode_Pathparam(t, n, "/", 1)[0]
	n = checkNode_Pathparam(t, n, "a", 1)[0]
	n = checkNode_Pathparam(t, n, "b", 1)[0]
	n = checkNode_Pathparam(t, n, "/", 1)[0]
	n = checkParamNode_Pathparam(t, n, ":path", 1)[0]
	branch := checkNode_Pathparam(t, n, "/", 2)

	n1 := branch[0]
	n1 = checkNode_Pathparam(t, n1, "x", 1)[0]
	n1 = checkNode_Pathparam(t, n1, "y", 1)[0]
	_ = checkNode_Pathparam(t, n1, "z", 0)
	checkHandler_Pathparam(t, n1, "dummy1")

	n2 := branch[1]
	n2 = checkNode_Pathparam(t, n2, "m", 1)[0]
	_ = checkNode_Pathparam(t, n2, "n", 0)
	checkHandler_Pathparam(t, n2, "dummy2")
}
