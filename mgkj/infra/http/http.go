package http

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"
)

type Http struct {
	server *http.Server
	router *Router
}

type PathGroup struct {
	path   string
	router *Router
}

type Handler func(context.Context, http.ResponseWriter, *http.Request)

type Filter func(context.Context, http.ResponseWriter, *http.Request) bool

type Router struct {
	mux        map[string]func(context.Context, http.ResponseWriter, *http.Request)
	preFilters map[string][]Filter
}

func (r *Router) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if f, ok := r.mux[req.URL.Path]; ok {
		ctx := context.Background()
		preFilters, isPreFilterExist := r.preFilters[req.URL.Path]
		if isPreFilterExist == true {
			for _, pre := range preFilters {
				isOk := pre(ctx, resp, req)
				if isOk == false {
					return
				}
			}
		}
		f(ctx, resp, req)
		return
	}
	http.Error(resp, "error url:"+req.URL.String(), http.StatusBadRequest)
}

func (h *Http) Init(host, port string) {

	h.router = &Router{
		mux:        make(map[string]func(context.Context, http.ResponseWriter, *http.Request)),
		preFilters: make(map[string][]Filter),
	}

	h.server = &http.Server{
		Addr:        host + ":" + port,
		Handler:     h.router,
		ReadTimeout: 5 * time.Second}

	go func() {
		log.Println("http server listen to: ", h.server.Addr)
		err := h.server.ListenAndServe()
		if err != nil {
			log.Fatal(err)
		}
	}()
}

func (h *Http) Post(path string, handler Handler, pre Filter) {
	h.router.mux[path] = handler
	filters := []Filter{post}
	h.router.preFilters[path] = filters
	if pre != nil {
		filters = append(filters, pre)
		h.router.preFilters[path] = filters
	}
}

func (h *Http) Get(path string, handler Handler, pre Filter) {
	h.router.mux[path] = handler
	filters := []Filter{get}
	h.router.preFilters[path] = filters
	if pre != nil {
		filters = append(filters, pre)
		h.router.preFilters[path] = filters
	}
}

func (h *Http) Group(path string, handler Handler, pre Filter) *PathGroup {
	h.router.mux[path] = handler
	g := &PathGroup{path: path, router: h.router}
	return g
}

func (g *PathGroup) Path() string {
	return g.path
}

func (g *PathGroup) Post(path string, handler Handler, pre Filter) {
	p := g.path + path
	g.router.mux[p] = handler
	filters := []Filter{post}
	g.router.preFilters[p] = filters
	if pre != nil {
		filters = append(filters, pre)
		g.router.preFilters[p] = filters
	}
}

func (g *PathGroup) Get(path string, handler Handler, pre Filter) {
	p := g.path + path
	g.router.mux[p] = handler
	filters := []Filter{get}
	g.router.preFilters[p] = filters
	if pre != nil {
		filters = append(filters, pre)
		g.router.preFilters[p] = filters
	}
}

func (h *Http) AddPreFilter(path string, filter Filter) {
	filters, ok := h.router.preFilters[path]
	if ok == false {
		return
	}
	filters = append(filters, filter)
	h.router.preFilters[path] = filters
}

func post(ctx context.Context, w http.ResponseWriter, req *http.Request) bool {
	if strings.ToUpper(req.Method) == "POST" {
		return true
	}
	w.WriteHeader(405) //method not allowed
	return false
}

func get(ctx context.Context, w http.ResponseWriter, req *http.Request) bool {
	if strings.ToUpper(req.Method) == "GET" {
		return true
	}
	w.WriteHeader(405) //method not allowed
	return false
}
