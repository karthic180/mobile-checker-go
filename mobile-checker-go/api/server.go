// Package api provides a lightweight HTTP REST API for the mobile checker.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/yourusername/mobile-checker/internal/checker"
)

// Server is the HTTP API server.
type Server struct {
	checker *checker.Checker
}

// NewServer creates a new API Server.
func NewServer(dataDir string) *Server {
	return &Server{checker: checker.New(dataDir)}
}

// Routes registers all API routes.
func (s *Server) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/mobile/bulk", s.handleBulk)
	mux.HandleFunc("/api/mobile/", s.handleMobile)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "UK Mobile Coverage API"})
}

// GET /api/mobile/{postcode}
func (s *Server) handleMobile(w http.ResponseWriter, r *http.Request) {
	pc := strings.TrimPrefix(r.URL.Path, "/api/mobile/")
	if pc == "" {
		writeError(w, http.StatusBadRequest, "postcode required")
		return
	}
	result := s.checker.Check(pc)
	if result.Error != "" {
		writeError(w, http.StatusNotFound, result.Error)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "result": result})
}

// POST /api/mobile/bulk — {"postcodes": ["SW1A1AA", "EC1A1BB"]}
func (s *Server) handleBulk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}
	var body struct {
		Postcodes []string `json:"postcodes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(body.Postcodes) == 0 || len(body.Postcodes) > 50 {
		writeError(w, http.StatusBadRequest, "provide between 1 and 50 postcodes")
		return
	}
	results := s.checker.CheckMultiple(body.Postcodes)
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "results": results})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"status": "error", "message": msg})
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	s.Routes(mux)
	fmt.Printf("UK Mobile Coverage API listening on http://%s\n", addr)
	fmt.Println("  GET  /health")
	fmt.Println("  GET  /api/mobile/{postcode}")
	fmt.Println("  POST /api/mobile/bulk")
	return http.ListenAndServe(addr, mux)
}
