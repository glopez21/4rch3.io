package proxy

import (
	"bufio"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/4rch3/portal/internal/database"
	"github.com/4rch3/portal/internal/middleware"
	"github.com/4rch3/portal/internal/models"
)

type Proxy struct {
	db      *database.DB
	proxies map[string]*httputil.ReverseProxy
	mu      sync.RWMutex
	wsDial  func(network, addr string) (net.Conn, error)
}

func New(db *database.DB) *Proxy {
	return &Proxy{
		db:      db,
		proxies: make(map[string]*httputil.ReverseProxy),
		wsDial: (&net.Dialer{Timeout: 10 * time.Second}).Dial,
	}
}

func isWebSocketRequest(r *http.Request) bool {
	return strings.ToLower(r.Header.Get("Connection")) == "upgrade" &&
		strings.ToLower(r.Header.Get("Upgrade")) == "websocket"
}

func (p *Proxy) resolveTarget(r *http.Request) (*url.URL, *models.User, *proxyError) {
	user := middleware.GetUser(r)
	isLAN := middleware.IsLAN(r)

	if user == nil && !isLAN {
		return nil, nil, errUnauthorized
	}

	path := strings.TrimPrefix(r.URL.Path, "/svc/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		return nil, nil, errServiceNotSpecified
	}

	serviceName := parts[0]

	svcs, err := p.db.ListServices()
	if err != nil {
		return nil, nil, errInternal
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
		return nil, nil, errServiceNotFound
	}

	if len(parts) > 1 {
		r.URL.Path = "/" + parts[1]
	} else {
		r.URL.Path = "/"
	}
	r.URL.RawPath = ""

	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, nil, errInvalidTarget
	}

	return target, user, nil
}

func (p *Proxy) setForwardingHeaders(r *http.Request, user *models.User) {
	r.Header.Del("X-Forwarded-User")
	r.Header.Del("X-Forwarded-Role")
	if user != nil {
		r.Header.Set("X-Forwarded-User", user.Username)
		r.Header.Set("X-Forwarded-Role", string(user.Role))
	}
	r.Header.Add("X-Forwarded-For", r.RemoteAddr)
}

func (p *Proxy) handleWebSocket(w http.ResponseWriter, r *http.Request, target *url.URL) {
	target.Scheme = "ws"
	if target.Scheme == "https" {
		target.Scheme = "wss"
	}

	backendConn, err := p.wsDial("tcp", target.Host)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "backend connection failed")
		return
	}
	defer backendConn.Close()

	if err := r.Write(backendConn); err != nil {
		writeJSONError(w, http.StatusBadGateway, "failed to send request to backend")
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "hijacking not supported")
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "hijack failed")
		return
	}
	defer clientConn.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		br := bufio.NewReader(backendConn)
		resp, err := http.ReadResponse(br, r)
		if err == nil && resp.StatusCode == http.StatusSwitchingProtocols {
			resp.Write(clientConn)
		}
	}()

	go func() {
		defer wg.Done()
		bidirectionalCopy(clientConn, backendConn)
	}()

	wg.Wait()
}

func bidirectionalCopy(dst, src net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		transfer(dst, src)
	}()
	go func() {
		defer wg.Done()
		transfer(src, dst)
	}()

	wg.Wait()
}

func transfer(dst, src net.Conn) {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				break
			}
		}
		if err != nil {
			break
		}
	}
}

func (p *Proxy) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target, user, err := p.resolveTarget(r)
		if err != nil {
		writeJSONError(w, err.code, err.message)
		return
	}

	p.setForwardingHeaders(r, user)

		if isWebSocketRequest(r) {
			p.handleWebSocket(w, r, target)
			return
		}

		proxy := p.getProxy(target)
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