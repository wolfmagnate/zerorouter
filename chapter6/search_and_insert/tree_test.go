package chapter6

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
	return func(http.ResponseWriter, *http.Request, Params) {
		fakeHandlerValue = val
	}
}

func checkNode(t *testing.T, n *node, expectedStr string, expectedChildren int) []*node {
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

func checkParamNode(t *testing.T, n *node, expectedParam string, expectedChildren int) []*node {
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

func checknode(t *testing.T, n *node, expectedCatchAll string) []*node {
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

func checkCatchAllNode(t *testing.T, n *node, expectedCatchAll string) []*node {
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

func checkHandler(t *testing.T, n *node, expectedValue string) {
	if n.handle == nil {
		t.Error("Expected handle at the last node to be non-nil")
	} else {
		n.handle(nil, nil, nil)
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

// 正常系テスト
// 1. 文字ノードだけで分割が正しく行われているかを検証する
func TestAddRoute_Static_1(t *testing.T) {
	n := &node{}
	n.addRoute("/abc/def", fakeHandler("dummy1"))
	n1 := n
	n1 = checkNode(t, n1, "", 1)[0]
	_ = checkNode(t, n1, "/abc/def", 0)
	checkHandler(t, n1, "dummy1")

	n.addRoute("/abc/xyz", fakeHandler("dummy2"))
	n1 = n
	n1 = checkNode(t, n1, "", 1)[0]
	branch := checkNode(t, n1, "/abc/", 2)
	n1 = branch[0]
	checkNode(t, n1, "def", 0)
	checkHandler(t, n1, "dummy1")
	n1 = branch[1]
	checkNode(t, n1, "xyz", 0)
	checkHandler(t, n1, "dummy2")

	n.addRoute("/ab/def", fakeHandler("dummy3"))
	n1 = n
	n1 = checkNode(t, n1, "", 1)[0]
	branch = checkNode(t, n1, "/ab", 2)
	n1 = branch[0]
	branch2 := checkNode(t, n1, "c/", 2)
	n2 := branch2[0]
	checkNode(t, n2, "def", 0)
	checkHandler(t, n2, "dummy1")
	n2 = branch2[1]
	checkNode(t, n2, "xyz", 0)
	checkHandler(t, n2, "dummy2")
	n1 = branch[1]
	checkNode(t, n1, "/def", 0)
	checkHandler(t, n1, "dummy3")

	n.addRoute("/ab/d", fakeHandler("dummy4"))
	n1 = n
	n1 = checkNode(t, n1, "", 1)[0]
	branch = checkNode(t, n1, "/ab", 2)
	n1 = branch[1]
	checkHandler(t, n1, "dummy4")
	n1 = checkNode(t, n1, "/d", 1)[0]
	checkHandler(t, n1, "dummy3")
	_ = checkNode(t, n1, "ef", 0)
}

// 2. パラメータノードを含む形で分割が正しく行われているかを検証する
// 2-1 パラメータノードが中盤
func TestAddRoute_Param_1(t *testing.T) {
	n := &node{}
	n.addRoute("/xy:path1", fakeHandler("dummy1"))           // パラメータノードを正常に木に変換できるか
	n.addRoute("/abc/:path2/xyz", fakeHandler("dummy2"))     // パラメータノードの次のstaticノード
	n.addRoute("/abc/:path2/xyz/abc", fakeHandler("dummy3")) // パラメータノードを含む末尾への追加
	n.addRoute("/abc/:path2/abc", fakeHandler("dummy4"))     // パラメータノードを経由した後の分割

	n = checkNode(t, n, "", 1)[0]
	branch := checkNode(t, n, "/", 2)
	n1 := branch[0]
	n1 = checkNode(t, n1, "xy", 1)[0]
	_ = checkParamNode(t, n1, ":path1", 0)
	checkHandler(t, n1, "dummy1")
	n2 := branch[1]
	n2 = checkNode(t, n2, "abc/", 1)[0]
	n2 = checkParamNode(t, n2, ":path2", 1)[0]
	branch2 := checkNode(t, n2, "/", 2)
	n3 := branch2[0]
	checkHandler(t, n3, "dummy2")
	n3 = checkNode(t, n3, "xyz", 1)[0]
	_ = checkNode(t, n3, "/abc", 0)
	checkHandler(t, n3, "dummy3")
	n4 := branch2[1]
	_ = checkNode(t, n4, "abc", 0)
	checkHandler(t, n4, "dummy4")
}

// 2-2 パラメータノードが最初
func TestAddRoute_Param_2(t *testing.T) {
	n := &node{}
	n.addRoute(":path1", fakeHandler("dummy1"))
	n.addRoute(":path1/xyz", fakeHandler("dummy2"))
	n.addRoute(":path1/xy", fakeHandler("dummy3"))

	n1 := checkNode(t, n, "", 1)[0]
	checkHandler(t, n1, "dummy1")
	n1 = checkParamNode(t, n1, ":path1", 1)[0]
	checkHandler(t, n1, "dummy3")
	n1 = checkNode(t, n1, "/xy", 1)[0]
	checkHandler(t, n1, "dummy2")
	_ = checkNode(t, n1, "z", 0)
}

// 2-3 パラメータノードが末尾
func TestAddRoute_Param_3(t *testing.T) {
	n := &node{}
	n.addRoute("/あいう:path1/xyz", fakeHandler("dummy1"))
	n.addRoute("/あいう:path1", fakeHandler("dummy2"))
	n.addRoute("/あいう:path1/xyz/:path2", fakeHandler("dummy3"))

	n1 := checkNode(t, n, "", 1)[0]
	n1 = checkNode(t, n1, "/あいう", 1)[0]
	checkHandler(t, n1, "dummy2")
	n1 = checkParamNode(t, n1, ":path1", 1)[0]
	checkHandler(t, n1, "dummy1")
	n1 = checkNode(t, n1, "/xyz", 1)[0]
	n1 = checkNode(t, n1, "/", 1)[0]
	checkHandler(t, n1, "dummy3")
	checkParamNode(t, n1, ":path2", 0)
}

// 3. キャッチオールノードを含む形で分割が正しく行われているかを検証する
func TestAddRoute_CatchAll_1(t *testing.T) {
	n := &node{}
	n.addRoute("/あいう/*everything1", fakeHandler("dummy1"))
	n.addRoute("/あい/*everything2", fakeHandler("dummy2"))
	n.addRoute("/あ/:path/*everything3", fakeHandler("dummy3"))

	n1 := checkNode(t, n, "", 1)[0]
	branch := checkNode(t, n1, "/あ", 2)
	n2 := branch[0]
	branch2 := checkNode(t, n2, "い", 2)
	n3 := branch2[0]
	n3 = checkNode(t, n3, "う", 1)[0]
	_ = checkCatchAllNode(t, n3, "/*everything1")
	checkHandler(t, n3, "dummy1")
	n4 := branch2[1]
	_ = checkCatchAllNode(t, n4, "/*everything2")
	checkHandler(t, n4, "dummy2")

	n5 := branch[1]
	n5 = checkNode(t, n5, "/", 1)[0]
	n5 = checkParamNode(t, n5, ":path", 1)[0]
	_ = checkCatchAllNode(t, n5, "/*everything3")
	checkHandler(t, n5, "dummy3")
}

// 異常系テスト
// 1. キャッチオールに子がある
func TestAddRoute_Error_1(t *testing.T) {
	n := &node{}
	recv := catchPanic(func() {
		n.addRoute("/a/*everything/b", fakeHandler("dummy"))
	})
	checkConflict(t, recv, catchAll, "/*everything", static, "/b")
}

// 2. キャッチオールノードの名前がおかしい
func TestAddRoute_Error_2(t *testing.T) {
	n := &node{}
	recv := catchPanic(func() {
		n.addRoute("/d/*everything:path", fakeHandler("dummy"))
	})
	if msg := recv.(string); msg == "invalid catchAll" {
		return
	}
	t.Errorf("Expected invalid catchAll, but got other panic")
}

// 3. 文字ノードの配下にキャッチオールとスラッシュを表す文字ノードが両方ある
func TestAddRoute_Error_3(t *testing.T) {
	n := &node{}
	recv := catchPanic(func() {
		n.addRoute("/a/*everything", fakeHandler("dummy1"))
		n.addRoute("/a/b", fakeHandler("dummy2"))
	})
	checkConflict(t, recv, static, "/a", static, "/b")
}

// 4. 文字ノードの配下にキャッチオールノードとパラメータが両方ある
func TestAddRoute_Error_4(t *testing.T) {
	n := &node{}
	recv := catchPanic(func() {
		n.addRoute("/a/*everything", fakeHandler("dummy1"))
		n.addRoute("/a:path", fakeHandler("dummy2"))
	})
	checkConflict(t, recv, static, "/a", param, ":path")
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
		handler, ps := n.retrieve(test.path)
		if handler == nil {
			if test.expectedValue != "" {
				t.Errorf("retrieve(%s) = nil, want %s", test.path, test.expectedValue)
			}
		} else {
			handler(nil, nil, nil)
			if fakeHandlerValue != test.expectedValue {
				t.Errorf("retrieve(%s) handler set fakeHandlerValue = %s, want %s", test.path, fakeHandlerValue, test.expectedValue)
			}
			if len(ps) != len(test.parameters) {
				t.Errorf("retrieve(%s) returned %d parameters; want %d", test.path, len(ps), len(test.parameters))
			}
			for _, p := range ps {
				if test.parameters[p.Key] != p.Value {
					t.Errorf("retrieve(%s) returned parameter %s with value %s; want %s", test.path, p.Key, p.Value, test.parameters[p.Key])
				}
			}
		}
	}
}
