// Package middleware — error_tracker.go : suivi des erreurs et alerting Discord.
//
// Sprint 40 T2+T3 :
//
//	T2 : capture les réponses HTTP 500 et envoie un webhook Discord (fire-and-forget).
//	T3 : calcule le taux d'erreur sur une fenêtre glissante de 1 minute.
//	     Si taux > 5 %, envoie une alerte Discord (max 1 alerte toutes les 5 min).
//
// Pas de dépendance externe : utilise net/http stdlib pour le webhook.
package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// ErrorTrackerConfig configure le middleware d'error tracking.
type ErrorTrackerConfig struct {
	// WebhookURL est l'URL Discord webhook. Vide = alerting désactivé.
	WebhookURL string
	// AlertThreshold est le pourcentage d'erreurs déclenchant l'alerte (défaut : 5.0).
	AlertThreshold float64
	// AlertCooldown est le délai minimum entre deux alertes (défaut : 5 min).
	AlertCooldown time.Duration
}

// ErrorTracker est un middleware qui suit le taux d'erreurs HTTP.
// Utiliser NewErrorTracker pour créer une instance.
type ErrorTracker struct {
	cfg ErrorTrackerConfig

	mu          sync.Mutex
	windowStart time.Time
	lastAlertAt time.Time

	// Compteurs atomiques pour la fenêtre courante (lecture sans verrou).
	atomicTotal  atomic.Int64
	atomicErrors atomic.Int64
}

// NewErrorTracker crée un ErrorTracker avec la configuration donnée.
// Si AlertThreshold <= 0, on utilise 5.0.
// Si AlertCooldown <= 0, on utilise 5 minutes.
func NewErrorTracker(cfg ErrorTrackerConfig) *ErrorTracker {
	if cfg.AlertThreshold <= 0 {
		cfg.AlertThreshold = 5.0
	}
	if cfg.AlertCooldown <= 0 {
		cfg.AlertCooldown = 5 * time.Minute
	}
	return &ErrorTracker{
		cfg:         cfg,
		windowStart: time.Now(),
	}
}

// Middleware retourne le http.Handler middleware.
// DÉSACTIVÉ EN DUR — l'alerting Discord 500/taux d'erreur n'est pas souhaité.
// Réactiver en supprimant le return immédiat ci-dessous.
func (et *ErrorTracker) Middleware(next http.Handler) http.Handler {
	return next
}

// middlewareDisabled est conservé pour référence future.
//
//nolint:unused
func (et *ErrorTracker) middlewareDisabled(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		etw := &errorTrackWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(etw, r)

		et.atomicTotal.Add(1)
		if etw.status >= 500 {
			et.atomicErrors.Add(1)
			if et.cfg.WebhookURL != "" {
				go et.notifyError(r, etw.status)
			}
		}

		et.checkWindow()
	})
}

// checkWindow vérifie si la fenêtre de 1 minute est expirée et calcule le taux.
func (et *ErrorTracker) checkWindow() {
	et.mu.Lock()
	defer et.mu.Unlock()

	if time.Since(et.windowStart) < time.Minute {
		return
	}

	// Snapshot atomique puis reset.
	totalSnap := et.atomicTotal.Swap(0)
	errorsSnap := et.atomicErrors.Swap(0)
	et.windowStart = time.Now()

	if totalSnap == 0 || et.cfg.WebhookURL == "" {
		return
	}

	rate := float64(errorsSnap) / float64(totalSnap) * 100
	if rate > et.cfg.AlertThreshold {
		// Anti-spam : respecter le cooldown.
		if time.Since(et.lastAlertAt) >= et.cfg.AlertCooldown {
			et.lastAlertAt = time.Now()
			go et.notifyRateAlert(rate, int(errorsSnap), int(totalSnap))
		}
	}
}

// notifyError envoie une notification Discord pour une erreur 5xx.
func (et *ErrorTracker) notifyError(r *http.Request, status int) {
	msg := fmt.Sprintf(":warning: **HTTP %d** sur `%s %s`", status, r.Method, r.URL.Path)
	et.postDiscord(discordSimplePayload(msg))
}

// notifyRateAlert envoie une alerte Discord pour un taux d'erreurs élevé.
func (et *ErrorTracker) notifyRateAlert(rate float64, errors, total int) {
	msg := fmt.Sprintf(
		":rotating_light: **Taux d'erreurs élevé** : %.1f%% (%d/%d requêtes sur la dernière minute) — seuil = %.0f%%",
		rate, errors, total, et.cfg.AlertThreshold,
	)
	et.postDiscord(discordSimplePayload(msg))
}

// postDiscord envoie un payload JSON au webhook Discord.
func (et *ErrorTracker) postDiscord(payload map[string]any) {
	data, err := json.Marshal(payload)
	if err != nil {
		slog.Warn("error_tracker: marshal webhook payload", "err", err)
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, et.cfg.WebhookURL, bytes.NewReader(data))
	if err != nil {
		slog.Warn("error_tracker: create webhook request", "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		slog.Warn("error_tracker: webhook request failed", "err", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		slog.Warn("error_tracker: webhook HTTP error", "status", resp.StatusCode)
	}
}

// discordSimplePayload construit un payload Discord minimal avec juste du texte.
func discordSimplePayload(content string) map[string]any {
	return map[string]any{"content": content}
}

// errorTrackWriter capture le code de statut HTTP.
type errorTrackWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (ew *errorTrackWriter) WriteHeader(code int) {
	if !ew.wrote {
		ew.status = code
		ew.wrote = true
	}
	ew.ResponseWriter.WriteHeader(code)
}

func (ew *errorTrackWriter) Write(b []byte) (int, error) {
	if !ew.wrote {
		ew.status = http.StatusOK
		ew.wrote = true
	}
	return ew.ResponseWriter.Write(b)
}
