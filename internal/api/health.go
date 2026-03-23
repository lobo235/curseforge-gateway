package api

import (
	"net/http"
)

type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// healthHandler returns a handler for GET /health.
// It pings CurseForge and returns 200 if reachable, 503 otherwise.
// This endpoint intentionally skips Bearer auth — it is called by Nomad's health check.
func (s *Server) healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.curseforge.Ping(r.Context()); err != nil {
			s.log.Error("curseforge ping failed", "error", err)
			writeJSON(w, http.StatusServiceUnavailable, healthResponse{Status: "unavailable", Version: s.version})
			return
		}
		writeJSON(w, http.StatusOK, healthResponse{Status: "ok", Version: s.version})
	}
}
