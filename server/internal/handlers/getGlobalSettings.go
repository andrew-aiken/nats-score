package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"server/internal/nats"
)

// GetGlobalSettings returns an object of the global settings
// from the NATS KV settings bucket (key settings)
func (h *Handler) GetChecks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	if h.NatsKVClient == nil {
		log.Printf("NATS KV client not initialized")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "NATS KV not available"})
		return
	}

	checks, err := nats.GetChecks(h.NatsKVClient)

	if err != nil {
		log.Printf("Failed to get checks: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to retrieve checks"})
		return
	}

	json.NewEncoder(w).Encode(checks)
}
