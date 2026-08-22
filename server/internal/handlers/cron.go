package handlers

import (
	"log/slog"
	"net/http"
)

// StartScoringCron handles starting and stopping the scoring cronjob
func (h *Handler) StartScoringCron(w http.ResponseWriter, r *http.Request) {
	// Only accept PUT requests
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		encodeJson(w, map[string]string{"error": "Method not allowed"})
		return
	}

	h.CronScheduler.Start()

	slog.Info("Starting check cron")

	encodeJson(w, "Started Scoring CronJob")
}

// StopScoringCron handles starting and stopping the scoring cronjob
func (h *Handler) StopScoringCron(w http.ResponseWriter, r *http.Request) {
	// Only accept PUT requests
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		encodeJson(w, map[string]string{"error": "Method not allowed"})
		return
	}

	err := h.CronScheduler.StopJobs()

	if err != nil {
		slog.Error("Failed to stop check cronjob", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		encodeJson(w, map[string]string{"error": "Failed to stop scoring cronjob"})
		return
	}

	slog.Info("Stopping check cron")

	encodeJson(w, "Stopped Scoring CronJob")
}
