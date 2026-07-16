// Package notify — disk.go : notification Discord d'alerte disque (lot ops
// 2026-07-13). Failsafe comme le reste du package : no-op si webhook absent ou
// toggle off, jamais d'erreur propagée.
package notify

import (
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
)

// Couleurs des embeds disque (convention du package : ambre = avertissement).
const (
	diskColorWarn     = 0xE0A800 // ambre
	diskColorCritical = 0xD0342C // rouge
	diskColorOK       = 0x2E8B57 // vert — rétablissement
)

// NotifyDiskAlert envoie un embed d'alerte (ou de rétablissement) disque.
// status : domain.FreshnessStatusWarn | Critical | OK (OK = recovery).
// L'anti-spam (transition / rappel 24 h) est décidé par le caller via
// ops.ShouldNotifyDisk — cette fonction envoie sans condition de fréquence.
// Retourne true si Discord a accepté le message.
func NotifyDiskAlert(cfg NotifyConfig, status, path string, freeBytes, totalBytes uint64, usedPercent float64) bool {
	if cfg.WebhookURL == "" || !cfg.NotifyDisk {
		return false
	}
	var titleKey, descKey string
	var color int
	switch status {
	case domain.FreshnessStatusCritical:
		titleKey, descKey, color = "discord_disk_critical_title", "discord_disk_alert_desc", diskColorCritical
	case domain.FreshnessStatusWarn:
		titleKey, descKey, color = "discord_disk_warn_title", "discord_disk_alert_desc", diskColorWarn
	case domain.FreshnessStatusOK:
		titleKey, descKey, color = "discord_disk_ok_title", "discord_disk_ok_desc", diskColorOK
	default:
		return false
	}
	return SendWebhook(cfg.WebhookURL, WebhookPayload{Embeds: []Embed{{
		Title: T(titleKey, cfg.Lang),
		Description: T(descKey, cfg.Lang,
			"used_pct", fmt.Sprintf("%.0f", usedPercent),
			"free", humanBytes(freeBytes),
			"total", humanBytes(totalBytes),
			"path", path,
		),
		Color:     color,
		Footer:    &EmbedFooter{Text: discordFooterText(cfg.Labels)},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}}})
}

// humanBytes rend une taille lisible (Go/Mo, base 1024) pour les embeds.
func humanBytes(b uint64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f Go", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.0f Mo", float64(b)/float64(1<<20))
	default:
		return fmt.Sprintf("%d o", b)
	}
}
