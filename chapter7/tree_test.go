// Copyright 2024 進捗ゼミ. All rights reserved.
// Based on the path package, Copyright 2009 The Go Authors.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.
package chapter7

import (
	"net/http"
	"testing"
)

var fakeHandlerValue string

func fakeHandler(val string) Handle {
	return func(http.ResponseWriter, *http.Request, Params) {
		fakeHandlerValue = val
	}
}

type testParamsProvider struct {
	ps *Params
}

func (t *testParamsProvider) add(p Param) {
	if t.ps == nil {
		ary := make(Params, 0)
		t.ps = &ary
	}
	*t.ps = append(*t.ps, p)
}
func (t *testParamsProvider) getParams() *Params {
	return t.ps
}

func TestRetrieve(t *testing.T) {
	n := &node{}
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
		parameters    map[string]string
	}{
		{"/a", "dummy1", nil},
		{"/a/012", "dummy2", map[string]string{
			"path": "012",
		}},
		{"/a/012/yeah", "dummy3", map[string]string{
			"path":       "012",
			"everything": "/yeah",
		}},
		{"/a/b/yeah:good", "dummy3", map[string]string{
			"path":       "b",
			"everything": "/yeah:good",
		}},
		{"/x", "dummy4", nil},
		{"/xy", "dummy5", nil},
		{"/xz", "dummy6", nil},
		{"/xz/", "dummy7", map[string]string{
			"file": "/",
		}},
		{"/xz/hoge/fuga", "dummy7", map[string]string{
			"file": "/hoge/fuga",
		}},
		{"/xzz", "dummy8", nil},
		{"/xyz", "dummy9", map[string]string{
			"id": "z",
		}},
		{"/xyzzz/n", "dummy10", map[string]string{
			"id": "zzz",
		}},
	}

	for _, test := range tests {
		handler, ps := n._retrieve(test.path, &testParamsProvider{})
		if handler == nil {
			if test.expectedValue != "" {
				t.Errorf("retrieve(%s) = nil, want %s", test.path, test.expectedValue)
			}
		} else {
			handler(nil, nil, nil)
			if fakeHandlerValue != test.expectedValue {
				t.Errorf("retrieve(%s) handler set fakeHandlerValue = %s, want %s", test.path, fakeHandlerValue, test.expectedValue)
			}
			if ps == nil && len(test.parameters) == 0 {
				continue
			}
			if len(*ps) != len(test.parameters) {
				t.Errorf("retrieve(%s) returned %d parameters; want %d", test.path, len(*ps), len(test.parameters))
			}
			for _, p := range *ps {
				if test.parameters[p.Key] != p.Value {
					t.Errorf("retrieve(%s) returned parameter %s with value %s; want %s", test.path, p.Key, p.Value, test.parameters[p.Key])
				}
			}
		}
	}
}
