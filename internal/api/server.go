package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/harrylincoln/taper/internal/throttle"
)

type Server struct {
	addr    string
	manager *throttle.Manager
	httpSrv *http.Server
}

func (s *Server) HttpHandler() http.Handler {
	return s.httpSrv.Handler
}

func NewServer(addr string, manager *throttle.Manager) *Server {
	s := &Server{
		addr:    addr,
		manager: manager,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/status", s.withCORS(s.handleStatus))
	mux.HandleFunc("/level", s.withCORS(s.handleLevel))

	s.httpSrv = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return s
}

// Simple CORS wrapper so the extension can call the API
func (s *Server) withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

func (s *Server) Start() error {
	return s.httpSrv.ListenAndServe()
}

func (s *Server) Shutdown() {
	_ = s.httpSrv.Close()
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	prof := s.manager.GetProfile()
	res := map[string]interface{}{
		"level":   s.manager.CurrentLevel(),
		"profile": prof,
	}
	_ = json.NewEncoder(w).Encode(res)
}

func (s *Server) handleLevel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Level int `json:"level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	s.manager.SetLevel(body.Level)
	log.Println("Level set to", body.Level)

	w.WriteHeader(http.StatusNoContent)
}
