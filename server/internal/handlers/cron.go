package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// StartScoringCron handles starting and stopping the scoring cronjob
func (h *Handler) StartScoringCron(w http.ResponseWriter, r *http.Request) {
	// Only accept PUT requests
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	h.CronScheduler.Start()

	slog.Info("Starting check cron")

	json.NewEncoder(w).Encode("Started Scoring CronJob")
}

// StopScoringCron handles starting and stopping the scoring cronjob
func (h *Handler) StopScoringCron(w http.ResponseWriter, r *http.Request) {
	// Only accept PUT requests
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	err := h.CronScheduler.StopJobs()

	if err != nil {
		slog.Error("Failed to stop check cronjob", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to stop scoring cronjob"})
		return
	}

	slog.Info("Stopping check cron")

	json.NewEncoder(w).Encode("Stopped Scoring CronJob")
}
