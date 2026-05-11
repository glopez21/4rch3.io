package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"

	"github.com/4rch3/portal/internal/database"
	"github.com/4rch3/portal/internal/middleware"
)

type Proxy struct {
	db       *database.DB
	proxies  map[string]*httputil.ReverseProxy
	mu       sync.RWMutex
}

func New(db *database.DB) *Proxy {
	return &Proxy{
		db:      db,
		proxies: make(map[string]*httputil.ReverseProxy),
	}
}

func (p *Proxy) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := middleware.GetUser(r)
		isLAN := middleware.IsLAN(r)

		if user == nil && !isLAN {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/svc/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 0 || parts[0] == "" {
			http.Error(w, "service not specified", http.StatusBadRequest)
			return
		}

		serviceName := parts[0]

		svcs, err := p.db.ListServices()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		var targetURL string
		for _, svc := range svcs {
			if svc.Name == serviceName && svc.Enabled {
				if svc.InternalURL != "" {
					targetURL = svc.InternalURL
				} else {
					targetURL = svc.URL
				}
				break
			}
		}

		if targetURL == "" {
			http.Error(w, "service not found", http.StatusNotFound)
			return
		}

		target, err := url.Parse(targetURL)
		if err != nil {
			http.Error(w, "invalid target", http.StatusInternalServerError)
			return
		}

		proxy := p.getProxy(target)

		rewrite := &httputil.ProxyRequest{
			In:  r,
			Out: r.Clone(r.Context()),
		}
		rewrite.SetURL(target)
		if len(parts) > 1 {
			rewrite.Out.URL.Path = "/" + parts[1]
		} else {
			rewrite.Out.URL.Path = "/"
		}
		rewrite.Out.URL.RawPath = ""
		rewrite.Out.Host = target.Host

		if user != nil {
			rewrite.Out.Header.Set("X-Forwarded-User", user.Username)
			rewrite.Out.Header.Set("X-Forwarded-Role", string(user.Role))
		}
		rewrite.Out.Header.Set("X-Forwarded-For", r.RemoteAddr)

		proxy.ServeHTTP(w, r)
	})
}

func (p *Proxy) getProxy(target *url.URL) *httputil.ReverseProxy {
	p.mu.RLock()
	proxy, ok := p.proxies[target.Host]
	p.mu.RUnlock()

	if ok {
		return proxy
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if proxy, ok := p.proxies[target.Host]; ok {
		return proxy
	}

	proxy = httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorLog = log.Default()
	p.proxies[target.Host] = proxy

	return proxy
}
