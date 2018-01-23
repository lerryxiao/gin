// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package gin

import (
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/lerryxiao/gin/render"
)

// Version is Framework's version.
const (
	Version                = "v1.2"
	defaultMultipartMemory = 32 << 20 // 32 MB
)

var default404Body = []byte("404 page not found")
var default405Body = []byte("405 method not allowed")
var defaultAppEngine bool

// HandlerFunc 回调函数
type HandlerFunc func(*Context)

// HandlersChain 回调函数数组
type HandlersChain []HandlerFunc

// Last returns the last handler in the chain. ie. the last handler is the main own.
func (c HandlersChain) Last() HandlerFunc {
	if length := len(c); length > 0 {
		return c[length-1]
	}
	return nil
}

// RouteInfo 路由信息
type RouteInfo struct {
	Method  string
	Path    string
	Handler string
}

// RoutesInfo 路由数组
type RoutesInfo []RouteInfo

// Engine is the framework's instance, it contains the muxer, middleware and configuration settings.
// Create an instance of Engine, by using New() or Default()
type Engine struct {
	RouterGroup
	delims           render.Delims
	secureJSONPrefix string
	HTMLRender       render.HTMLRender
	FuncMap          template.FuncMap
	allNoRoute       HandlersChain
	allNoMethod      HandlersChain
	noRoute          HandlersChain
	noMethod         HandlersChain
	pool             sync.Pool
	trees            MethodTrees

	// Enables automatic redirection if the current route can't be matched but a
	// handler for the path with (without) the trailing slash exists.
	// For example if /foo/ is requested but a route only exists for /foo, the
	// client is redirected to /foo with http status code 301 for GET requests
	// and 307 for all other request methods.
	RedirectTrailingSlash bool

	// If enabled, the router tries to fix the current request path, if no
	// handle is registered for it.
	// First superfluous path elements like ../ or // are removed.
	// Afterwards the router does a case-insensitive lookup of the cleaned path.
	// If a handle can be found for this route, the router makes a redirection
	// to the corrected path with status code 301 for GET requests and 307 for
	// all other request methods.
	// For example /FOO and /..//Foo could be redirected to /foo.
	// RedirectTrailingSlash is independent of this option.
	RedirectFixedPath bool

	// If enabled, the router checks if another method is allowed for the
	// current route, if the current request can not be routed.
	// If this is the case, the request is answered with 'Method Not Allowed'
	// and HTTP status code 405.
	// If no other Method is allowed, the request is delegated to the NotFound
	// handler.
	HandleMethodNotAllowed bool
	ForwardedByClientIP    bool

	// #726 #755 If enabled, it will thrust some headers starting with
	// 'X-AppEngine...' for better integration with that PaaS.
	AppEngine bool

	// If enabled, the url.RawPath will be used to find parameters.
	UseRawPath bool
	// If true, the path value will be unescaped.
	// If UseRawPath is false (by default), the UnescapePathValues effectively is true,
	// as url.Path gonna be used, which is already unescaped.
	UnescapePathValues bool

	// Value of 'maxMemory' param that is given to http.Request's ParseMultipartForm
	// method call.
	MaxMultipartMemory int64

	// mark info when call Group func
	groupRouter []string
}

var _ IRouter = &Engine{}

// New returns a new blank Engine instance without any middleware attached.
// By default the configuration is:
// - RedirectTrailingSlash:  true
// - RedirectFixedPath:      false
// - HandleMethodNotAllowed: false
// - ForwardedByClientIP:    true
// - UseRawPath:             false
// - UnescapePathValues:     true
func New() *Engine {
	debugPrintWARNINGNew()
	engine := &Engine{
		RouterGroup: RouterGroup{
			Handlers: nil,
			basePath: "/",
			root:     true,
		},
		FuncMap:                template.FuncMap{},
		RedirectTrailingSlash:  true,
		RedirectFixedPath:      false,
		HandleMethodNotAllowed: false,
		ForwardedByClientIP:    true,
		AppEngine:              defaultAppEngine,
		UseRawPath:             false,
		UnescapePathValues:     true,
		MaxMultipartMemory:     defaultMultipartMemory,
		trees:                  make(MethodTrees, 0, 9),
		delims:                 render.Delims{Left: "{{", Right: "}}"},
		secureJSONPrefix:       "while(1);",
	}
	engine.RouterGroup.engine = engine
	engine.pool.New = func() interface{} {
		return engine.allocateContext()
	}
	return engine
}

// Default returns an Engine instance with the Logger and Recovery middleware already attached.
func Default() *Engine {
	engine := New()
	engine.Use(Logger(), Recovery())
	return engine
}

// Trees 目录表
func (engin *Engine) Trees() *MethodTrees {
	return &engin.trees
}

func (engin *Engine) allocateContext() *Context {
	return &Context{engine: engin}
}

// Delims 切分
func (engin *Engine) Delims(left, right string) *Engine {
	engin.delims = render.Delims{Left: left, Right: right}
	return engin
}

// SecureJSONPrefix json前缀
func (engin *Engine) SecureJSONPrefix(prefix string) *Engine {
	engin.secureJSONPrefix = prefix
	return engin
}

// LoadHTMLGlob 加载html
func (engin *Engine) LoadHTMLGlob(pattern string) {
	if IsDebugging() {
		debugPrintLoadTemplate(template.Must(template.New("").Delims(engin.delims.Left, engin.delims.Right).Funcs(engin.FuncMap).ParseGlob(pattern)))
		engin.HTMLRender = render.HTMLDebug{Glob: pattern, FuncMap: engin.FuncMap, Delims: engin.delims}
		return
	}

	templ := template.Must(template.New("").Delims(engin.delims.Left, engin.delims.Right).Funcs(engin.FuncMap).ParseGlob(pattern))
	engin.SetHTMLTemplate(templ)
}

// LoadHTMLFiles 加载html文件
func (engin *Engine) LoadHTMLFiles(files ...string) {
	if IsDebugging() {
		engin.HTMLRender = render.HTMLDebug{Files: files, FuncMap: engin.FuncMap, Delims: engin.delims}
		return
	}

	templ := template.Must(template.New("").Delims(engin.delims.Left, engin.delims.Right).Funcs(engin.FuncMap).ParseFiles(files...))
	engin.SetHTMLTemplate(templ)
}

// SetHTMLTemplate 设置html模板
func (engin *Engine) SetHTMLTemplate(templ *template.Template) {
	if len(engin.trees) > 0 {
		debugPrintWARNINGSetHTMLTemplate()
	}
	engin.HTMLRender = render.HTMLProduction{Template: templ.Funcs(engin.FuncMap)}
}

// SetFuncMap 设置函数表
func (engin *Engine) SetFuncMap(funcMap template.FuncMap) {
	engin.FuncMap = funcMap
}

// NoRoute adds handlers for NoRoute. It return a 404 code by default.
func (engin *Engine) NoRoute(handlers ...HandlerFunc) {
	engin.noRoute = handlers
	engin.rebuild404Handlers()
}

// NoMethod sets the handlers called when... TODO.
func (engin *Engine) NoMethod(handlers ...HandlerFunc) {
	engin.noMethod = handlers
	engin.rebuild405Handlers()
}

// Use attachs a global middleware to the router. ie. the middleware attached though Use() will be
// included in the handlers chain for every single request. Even 404, 405, static files...
// For example, this is the right place for a logger or error management middleware.
func (engin *Engine) Use(middleware ...HandlerFunc) IRoutes {
	engin.RouterGroup.Use(middleware...)
	engin.rebuild404Handlers()
	engin.rebuild405Handlers()
	return engin
}

func (engin *Engine) rebuild404Handlers() {
	engin.allNoRoute = engin.combineHandlers(engin.noRoute)
}

func (engin *Engine) rebuild405Handlers() {
	engin.allNoMethod = engin.combineHandlers(engin.noMethod)
}

// AddRoute register mentod path with handlers
func (engin *Engine) AddRoute(method, path string, handlers HandlersChain) {
	assert1(path[0] == '/', "path must begin with '/'")
	assert1(len(method) > 0, "HTTP method can not be empty")
	assert1(len(handlers) > 0, "there must be at least one handler")

	debugPrintRoute(method, path, handlers)
	root := engin.trees.get(method)
	if root == nil {
		root = new(Node)
		engin.trees = append(engin.trees, MethodTree{method: method, root: root})
	}
	root.AddRoute(path, handlers)
}

// GetHandlers check has mentod path registed
func (engin *Engine) GetHandlers(method, path string) HandlersChain {
	if len(method) <= 0 || len(path) <= 0 || path[0] != '/' {
		return nil
	}
	root := engin.trees.get(method)
	if root == nil {
		return nil
	}
	return root.GetHandlers(path)
}

// DelRoute check has mentod path registed
func (engin *Engine) DelRoute(method, path string, handlers HandlersChain) {
	if len(method) <= 0 || len(path) <= 0 || path[0] != '/' || len(handlers) <= 0 {
		return
	}
	root := engin.trees.get(method)
	if root == nil {
		return
	}
	root.DelRoute(path, handlers)
}

// Routes returns a slice of registered routes, including some useful information, such as:
// the http method, path and the handler name.
func (engin *Engine) Routes() (routes RoutesInfo) {
	for _, tree := range engin.trees {
		routes = iterate("", tree.method, routes, tree.root)
	}
	return routes
}

func iterate(path, method string, routes RoutesInfo, root *Node) RoutesInfo {
	path += root.path
	if len(root.handlers) > 0 {
		routes = append(routes, RouteInfo{
			Method:  method,
			Path:    path,
			Handler: nameOfFunction(root.handlers.Last()),
		})
	}
	for _, child := range root.children {
		routes = iterate(path, method, routes, child)
	}
	return routes
}

// Run attaches the router to a http.Server and starts listening and serving HTTP requests.
// It is a shortcut for http.ListenAndServe(addr, router)
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (engin *Engine) Run(addr ...string) (err error) {
	defer func() { debugPrintError(err) }()

	address := resolveAddress(addr)
	debugPrint("Listening and serving HTTP on %s\n", address)
	err = http.ListenAndServe(address, engin)
	return
}

// RunTLS attaches the router to a http.Server and starts listening and serving HTTPS (secure) requests.
// It is a shortcut for http.ListenAndServeTLS(addr, certFile, keyFile, router)
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (engin *Engine) RunTLS(addr, certFile, keyFile string) (err error) {
	debugPrint("Listening and serving HTTPS on %s\n", addr)
	defer func() { debugPrintError(err) }()

	err = http.ListenAndServeTLS(addr, certFile, keyFile, engin)
	return
}

// RunUnix attaches the router to a http.Server and starts listening and serving HTTP requests
// through the specified unix socket (ie. a file).
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (engin *Engine) RunUnix(file string) (err error) {
	debugPrint("Listening and serving HTTP on unix:/%s", file)
	defer func() { debugPrintError(err) }()

	os.Remove(file)
	listener, err := net.Listen("unix", file)
	if err != nil {
		return
	}
	defer listener.Close()
	err = http.Serve(listener, engin)
	return
}

// ServeHTTP conforms to the http.Handler interface.
func (engin *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := engin.pool.Get().(*Context)
	c.writermem.reset(w)
	c.Request = req
	c.reset()

	engin.handleHTTPRequest(c)
	engin.pool.Put(c)
}

// HandleContext re-enter a context that has been rewritten.
// This can be done by setting c.Request.Path to your new target.
// Disclaimer: You can loop yourself to death with this, use wisely.
func (engin *Engine) HandleContext(c *Context) {
	c.reset()
	engin.handleHTTPRequest(c)
	engin.pool.Put(c)
}

func (engin *Engine) handleHTTPRequest(context *Context) {
	httpMethod := context.Request.Method
	path := context.Request.URL.Path
	unescape := false
	if engin.UseRawPath && len(context.Request.URL.RawPath) > 0 {
		path = context.Request.URL.RawPath
		unescape = engin.UnescapePathValues
	}

	// Find root of the tree for the given HTTP method
	t := engin.trees
	for i, tl := 0, len(t); i < tl; i++ {
		if t[i].method == httpMethod {
			root := t[i].root
			// Find route in tree
			handlers, params, tsr := root.getValue(path, context.Params, unescape)
			if handlers != nil {
				context.handlers = handlers
				context.Params = params
				context.Next()
				context.writermem.WriteHeaderNow()
				return
			}
			if httpMethod != "CONNECT" && path != "/" {
				if tsr && engin.RedirectTrailingSlash {
					redirectTrailingSlash(context)
					return
				}
				if engin.RedirectFixedPath && redirectFixedPath(context, root, engin.RedirectFixedPath) {
					return
				}
			}
			break
		}
	}

	if engin.HandleMethodNotAllowed {
		for _, tree := range engin.trees {
			if tree.method != httpMethod {
				if handlers, _, _ := tree.root.getValue(path, nil, unescape); handlers != nil {
					context.handlers = engin.allNoMethod
					serveError(context, 405, default405Body)
					return
				}
			}
		}
	}
	context.handlers = engin.allNoRoute
	serveError(context, 404, default404Body)
}

func (engin *Engine) markRoute(path string, group bool) {
	if len(path) > 0 {
		if group == true {
			engin.groupRouter = append(engin.groupRouter, fmt.Sprintf("^~ %s", path))
		} else {
			engin.groupRouter = append(engin.groupRouter, path)
		}
	}
}

func (engin *Engine) getGroupRoute() *[]string {
	return &engin.groupRouter
}

var mimePlain = []string{MIMEPlain}

func serveError(c *Context, code int, defaultMessage []byte) {
	c.writermem.status = code
	c.Next()
	if !c.writermem.Written() {
		if c.writermem.Status() == code {
			c.writermem.Header()["Content-Type"] = mimePlain
			c.Writer.Write(defaultMessage)
		} else {
			c.writermem.WriteHeaderNow()
		}
	}
}

func redirectTrailingSlash(c *Context) {
	req := c.Request
	path := req.URL.Path
	code := 301 // Permanent redirect, request with GET method
	if req.Method != "GET" {
		code = 307
	}

	if len(path) > 1 && path[len(path)-1] == '/' {
		req.URL.Path = path[:len(path)-1]
	} else {
		req.URL.Path = path + "/"
	}
	debugPrint("redirecting request %d: %s --> %s", code, path, req.URL.String())
	http.Redirect(c.Writer, req, req.URL.String(), code)
	c.writermem.WriteHeaderNow()
}

func redirectFixedPath(c *Context, root *Node, trailingSlash bool) bool {
	req := c.Request
	path := req.URL.Path

	fixedPath, found := root.findCaseInsensitivePath(
		cleanPath(path),
		trailingSlash,
	)
	if found {
		code := 301 // Permanent redirect, request with GET method
		if req.Method != "GET" {
			code = 307
		}
		req.URL.Path = string(fixedPath)
		debugPrint("redirecting request %d: %s --> %s", code, path, req.URL.String())
		http.Redirect(c.Writer, req, req.URL.String(), code)
		c.writermem.WriteHeaderNow()
		return true
	}
	return false
}
