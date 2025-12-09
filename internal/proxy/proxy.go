package proxy

import (
	"log"
	"net"
	"net/http"
	"time"

	"github.com/harrylincoln/taper/internal/throttle"
)

type Server struct {
	addr    string
	manager *throttle.Manager
	httpSrv *http.Server
}

func NewServer(addr string, manager *throttle.Manager) *Server {
	s := &Server{
		addr:    addr,
		manager: manager,
	}

	handler := http.HandlerFunc(s.handle)

	s.httpSrv = &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	return s
}

func (s *Server) Start() error {
	return s.httpSrv.ListenAndServe()
}

func (s *Server) Shutdown() {
	_ = s.httpSrv.Close()
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		s.handleHTTPS(w, r)
	} else {
		s.handleHTTP(w, r)
	}
}

func (s *Server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	prof := s.manager.GetProfile()

	// Optional latency
	if prof.LatencyMs > 0 {
		time.Sleep(time.Duration(prof.LatencyMs) * time.Millisecond)
	}

	// Forward the request
	req, err := http.NewRequest(r.Method, r.RequestURI, r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	req.Header = r.Header.Clone()

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// copy headers
	for k, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	_, _ = throttle.ThrottledCopy(w, resp.Body, int64(prof.DownloadBytesPerSec))
}

func (s *Server) handleHTTPS(w http.ResponseWriter, r *http.Request) {
	prof := s.manager.GetProfile()

	if prof.LatencyMs > 0 {
		time.Sleep(time.Duration(prof.LatencyMs) * time.Millisecond)
	}

	// Hijack the connection
	h, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := h.Hijack()
	if err != nil {
		log.Println("hijack error:", err)
		return
	}

	// Dial target
	targetConn, err := net.Dial("tcp", r.Host)
	if err != nil {
		log.Println("dial error:", err)
		_ = clientConn.Close()
		return
	}

	// Send 200 connection established
	_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// Copy data both ways with throttling
	go func() {
		defer clientConn.Close()
		defer targetConn.Close()
		_, _ = throttle.ThrottledCopy(targetConn, clientConn, int64(prof.UploadBytesPerSec))
	}()

	go func() {
		defer clientConn.Close()
		defer targetConn.Close()
		_, _ = throttle.ThrottledCopy(clientConn, targetConn, int64(prof.DownloadBytesPerSec))
	}()
}
