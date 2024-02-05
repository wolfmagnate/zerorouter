package chapter4

import (
	"net/http"
	"path"
)

type Router struct {
	trees map[string]*node

	// ハンドル関数のpanicへの対処 で追加
	PanicHandler func(http.ResponseWriter, *http.Request, interface{})

	// OPTIONメソッド で追加
	HandleOPTIONS bool

	// 405 Method Not Allowed で追加
	HandleMethodNotAllowed bool

	// マッチしたパスの取得 で追加
	SaveMatchedRoutePath bool

	// 正規化したパスへのリダイレクト で追加
	RedirectFixedPath bool
}

func New() *Router {
	return &Router{}
}

func (r *Router) ServeHTTP_1(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	if root := r.trees[req.Method]; root != nil {
		if handle, ps := root.retrieve(path); handle != nil {
			handle(w, req, ps)
			return
		}
	}
	http.NotFound(w, req)
}

// ハンドル関数のpanicへの対処 でのServeHTTP実装
func (r *Router) ServeHTTP_2(w http.ResponseWriter, req *http.Request) {
	if r.PanicHandler != nil {
		defer r.recv(w, req)
	}
	path := req.URL.Path
	if root := r.trees[req.Method]; root != nil {
		if handle, ps := root.retrieve(path); handle != nil {
			handle(w, req, ps)
			return
		}
	}
	http.NotFound(w, req)
}

func (r *Router) ServeHTTP_3(w http.ResponseWriter, req *http.Request) {
	if r.PanicHandler != nil {
		defer r.recv(w, req)
	}
	path := req.URL.Path
	if root := r.trees[req.Method]; root != nil {
		if handle, ps := root.retrieve(path); handle != nil {
			handle(w, req, ps)
			return
		}
	}
	if req.Method == http.MethodOptions && r.HandleOPTIONS {
		if allow := r.allowed(path); allow != "" {
			w.Header().Set("Allow", allow)
			return
		}
	}
	http.NotFound(w, req)
}

func (r *Router) ServeHTTP_4(w http.ResponseWriter, req *http.Request) {
	if r.PanicHandler != nil {
		defer r.recv(w, req)
	}
	path := req.URL.Path
	if root := r.trees[req.Method]; root != nil {
		if handle, ps := root.retrieve(path); handle != nil {
			handle(w, req, ps)
			return
		}
	}
	if req.Method == http.MethodOptions && r.HandleOPTIONS {
		if allow := r.allowed(path); allow != "" {
			w.Header().Set("Allow", allow)
			return
		}
	} else if r.HandleMethodNotAllowed {
		if allow := r.allowed(path); allow != "" {
			w.Header().Set("Allow", allow)
			http.Error(w,
				http.StatusText(http.StatusMethodNotAllowed),
				http.StatusMethodNotAllowed,
			)
			return
		}
	}
	http.NotFound(w, req)
}

func (r *Router) ServeHTTP_5(w http.ResponseWriter, req *http.Request) {
	if r.PanicHandler != nil {
		defer r.recv(w, req)
	}
	urlPath := req.URL.Path
	if root := r.trees[req.Method]; root != nil {
		if handle, ps := root.retrieve(urlPath); handle != nil {
			handle(w, req, ps)
			return
		} else if urlPath != "/" {
			code := http.StatusMovedPermanently
			if req.Method != http.MethodGet {
				code = http.StatusPermanentRedirect
			}
			if r.RedirectFixedPath {
				fixedPath := path.Clean(urlPath)
				if handle, _ := root.retrieve(fixedPath); handle != nil {
					req.URL.Path = fixedPath
					http.Redirect(w, req, req.URL.String(), code)
					return
				}
			}
		}
	}
	if req.Method == http.MethodOptions && r.HandleOPTIONS {
		if allow := r.allowed(urlPath); allow != "" {
			w.Header().Set("Allow", allow)
			return
		}
	} else if r.HandleMethodNotAllowed {
		if allow := r.allowed(urlPath); allow != "" {
			w.Header().Set("Allow", allow)
			http.Error(w,
				http.StatusText(http.StatusMethodNotAllowed),
				http.StatusMethodNotAllowed,
			)
			return
		}
	}
	http.NotFound(w, req)
}
