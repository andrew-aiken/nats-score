package handlers

import (
	"log/slog"
	"net/http"
	"server/internal/nats"
)

// GetGlobalSettings returns an object of the global settings
// from the NATS KV settings bucket (key settings)
func (h *Handler) GetChecks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		encodeJson(w, map[string]string{"error": "Method not allowed"})
		return
	}

	if h.NatsKVClient == nil {
		slog.Warn("NATS KV client not initialized")
		w.WriteHeader(http.StatusServiceUnavailable)
		encodeJson(w, map[string]string{"error": "NATS KV not available"})
		return
	}

	checks, err := nats.GetChecks(h.NatsKVClient)

	if err != nil {
		slog.Warn("Failed to get checks", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		encodeJson(w, map[string]string{"error": "Failed to retrieve checks"})
		return
	}

	encodeJson(w, checks)
}
