package handlers

import (
	"log/slog"
	"net/http"
	"github.com/andrew-aiken/score/internal/nats"
)

// GetGlobalSettings returns an object of the global settings
// from the NATS KV settings bucket (key settings)
func (h *Handler) ChecksJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	checks, err := nats.GetChecks(h.NatsKVClient)

	if err != nil {
		slog.Warn("Failed to get checks", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		encodeJson(w, map[string]string{"error": "Failed to retrieve checks"})
		return
	}

	encodeJson(w, checks)
}
