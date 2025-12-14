package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

// GetGlobalSettings returns an object of the global settings
// from the NATS KV settings bucket (key settings)
func (h *Handler) GetGlobalSettings(w http.ResponseWriter, r *http.Request) {
	if h.NatsKVClient == nil {
		log.Printf("NATS KV client not initialized")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "NATS KV not available"})
		return
	}

	settings, err := h.NatsKVClient.GetSettings()

	if err != nil {
		log.Printf("Failed to get mutable fields: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to retrieve mutable fields"})
		return
	}

	json.NewEncoder(w).Encode(settings)
}
