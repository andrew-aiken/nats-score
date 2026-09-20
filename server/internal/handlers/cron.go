package handlers

import (
	"log/slog"
	"net/http"
)

// StartScoringCron handles starting the scoring cronjob
func (h *Handler) StartScoringCron(w http.ResponseWriter, r *http.Request) {
	// Only accept PUT requests
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		encodeJson(w, map[string]string{"error": "Method not allowed"})
		return
	}

	if h.CronScheduler == nil {
		slog.Warn("cron scheduler not initialized")
		w.WriteHeader(http.StatusServiceUnavailable)
		encodeJson(w, map[string]string{"error": "cron scheduler not initialized"})
		return
	}

	h.CronScheduler.Start()

	slog.Info("Starting check cron")

	encodeJson(w, "Started Scoring CronJob") // TODO: switch to json encoding
}

// StopScoringCron handles stopping the scoring cronjob
func (h *Handler) StopScoringCron(w http.ResponseWriter, r *http.Request) {
	// Only accept PUT requests
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		encodeJson(w, map[string]string{"error": "Method not allowed"})
		return
	}

	if h.CronScheduler == nil {
		slog.Warn("cron scheduler not initialized")
		w.WriteHeader(http.StatusServiceUnavailable)
		encodeJson(w, map[string]string{"error": "cron scheduler not initialized"})
		return
	}

	err := h.CronScheduler.StopJobs()

	if err != nil {
		slog.Error("Failed to stop check cronjob", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		encodeJson(w, map[string]string{"error": "Failed to stop scoring cronjob"})
		return
	}

	slog.Info("Stopping check cron")

	encodeJson(w, "Stopped Scoring CronJob") // TODO: switch to json encoding
}
