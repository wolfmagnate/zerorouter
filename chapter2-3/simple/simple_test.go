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

func checkNode(t *testing.T, n *simpleNode, expectedStr string, expectedChildren int) []*simpleNode {
	runes := []rune(expectedStr)
	var expectedRune rune
	if len(runes) == 0 {
		expectedRune = rune(0)
	} else {
		expectedRune = runes[0]
	}

	if n.path != expectedRune {
		t.Errorf("Expected rune %c not found in node, got %c", expectedRune, n.path)
	}
	if len(n.children) != expectedChildren {
		t.Errorf("Expected %d children, got %d", expectedChildren, len(n.children))
	}
	if expectedChildren > 0 {
		return n.children
	}
	return nil
}

func checkHandler(t *testing.T, n *simpleNode, expectedValue string) {
	if n.handle == nil {
		t.Error("Expected handle at the last node to be non-nil")
	} else {
		n.handle(nil, nil)
		if fakeHandlerValue != expectedValue {
			t.Errorf("Expected fakeHandlerValue to be '%s', got '%s'", expectedValue, fakeHandlerValue)
		}
	}
}

func TestAddRoute_SinglePath(t *testing.T) {
	n := &simpleNode{}
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
	n := &simpleNode{}
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

func TestRetrieve(t *testing.T) {
	n := &simpleNode{}
	n.addRoute("/a", fakeHandler("dummy1"))
	n.addRoute("/xy", fakeHandler("dummy2"))
	n.addRoute("/xy/z", fakeHandler("dummy3"))

	tests := []struct {
		path          string
		expectedValue string
	}{
		{"/a", "dummy1"},
		{"/xy", "dummy2"},
		{"/xy/z", "dummy3"},
		{"/x", ""},
		{"/b", ""},
	}

	for _, test := range tests {
		handler := n.retrieve(test.path)
		if handler == nil {
			if test.expectedValue != "" {
				t.Errorf("retrieve(%s) = nil, want %s", test.path, test.expectedValue)
			}
		} else {
			handler(nil, nil) // Call the handler to set the fakeHandlerValue
			if fakeHandlerValue != test.expectedValue {
				t.Errorf("retrieve(%s) handler set fakeHandlerValue = %s, want %s", test.path, fakeHandlerValue, test.expectedValue)
			}
		}
	}
}
