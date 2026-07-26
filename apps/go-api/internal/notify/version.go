// version.go — Notification de nouvelle version avec anti-spam robuste.
//
// Correction du bug Python : le guard session_state Streamlit se remettait
// à zéro à chaque refresh → spam Discord. En Go on évite tout état mémoire :
// on lit TOUJOURS app_settings.json pour comparer, et on ne met à jour
// last_notified_version QUE si Discord confirme la réception (HTTP 200/204).
package notify

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"levelup/go-api/internal/platform/atomicfile"
)

const maxDiscordBody = 1900 // Discord limite les embeds à 2048 chars

// NotifyNewVersion envoie une notification Discord si la version a changé
// (major ou minor). Retourne true si la notification a été envoyée avec succès.
//
// Anti-spam garanti :
//  1. Lit last_notified_version depuis app_settings.json à chaque appel.
//  2. Si last_notified est vide → initialise sans notifier (premier démarrage).
//  3. Si last_notified == currentVersion → skip (déjà notifié).
//  4. Si seul le patch change (ex: 1.2.3 → 1.2.4) → skip.
//  5. N'écrit last_notified_version QUE si Discord confirme (HTTP 200/204).
//  6. Écriture via atomicfile.WriteFile : atomique (tmp → rename) quand
//     l'environnement le permet, in-place quand la cible est bind-montée fichier
//     (conteneur prod, rename EBUSY) — cf. limites du repli dans ce package.
//  7. Opt-in prod : la variable d'environnement LEVELUP_NOTIFY_VERSIONS=1
//     est nécessaire pour envoyer (évite le spam en dev/staging).
func NotifyNewVersion(cfg NotifyConfig, currentVersion string) bool {
	// NotifyNewVersion est appele depuis le boot — pas de ctx propage. On
	// utilise context.Background() comme fallback canonique (revue 2026-04-29
	// P3.5 — migration log.Printf -> slog).
	ctx := context.Background()
	if !cfg.NotifyVersion {
		return false
	}
	if os.Getenv("LEVELUP_NOTIFY_VERSIONS") != "1" {
		slog.DebugContext(ctx, "discord_version_skip_flag_off", "op", "version", "reason", "LEVELUP_NOTIFY_VERSIONS != 1")
		return false
	}
	if cfg.WebhookURL == "" {
		return false
	}

	// 1. Lire l'état persisté depuis app_settings.json
	lastNotified := readLastNotifiedVersion(cfg.SettingsPath)

	// 2. Premier démarrage : initialiser sans notifier
	if lastNotified == "" {
		slog.InfoContext(ctx, "discord_version_initialized", "op", "version", "current_version", currentVersion)
		_ = writeLastNotifiedVersion(cfg.SettingsPath, currentVersion)
		return false
	}

	// 3. Déjà notifié pour cette version exacte
	if lastNotified == currentVersion {
		slog.DebugContext(ctx, "discord_version_skip_already_notified", "op", "version", "current_version", currentVersion)
		return false
	}

	// 4. Changement patch seul : skip
	if !isMajorMinorChange(lastNotified, currentVersion) {
		slog.DebugContext(ctx, "discord_version_skip_patch_only", "op", "version", "from", lastNotified, "to", currentVersion)
		// On met quand même à jour pour éviter une accumulation de patches
		_ = writeLastNotifiedVersion(cfg.SettingsPath, currentVersion)
		return false
	}

	// 5. Construire et envoyer l'embed
	changelog := extractWhatsNew(currentVersion, cfg.SettingsPath)
	embed := BuildVersionEmbed(currentVersion, changelog, cfg.Lang)

	ok := SendWebhook(cfg.WebhookURL, WebhookPayload{Embeds: []Embed{embed}})
	if !ok {
		slog.WarnContext(ctx, "discord_version_send_failed", "op", "version", "current_version", currentVersion)
		return false
	}

	// 6. N'écrire QUE si l'envoi a réussi
	if err := writeLastNotifiedVersion(cfg.SettingsPath, currentVersion); err != nil {
		slog.ErrorContext(ctx, "discord_version_persist_failed", "op", "version", "err", err, "current_version", currentVersion)
	}
	slog.InfoContext(ctx, "discord_version_sent", "op", "version", "current_version", currentVersion)
	return true
}

// BuildVersionEmbed construit l'embed Rich Discord pour une nouvelle version.
func BuildVersionEmbed(version, changelog, lang string) Embed {
	cleanVer := strings.TrimPrefix(version, "v")
	title := T("discord_version_title", lang, "version", cleanVer)

	desc := changelog
	if desc == "" {
		desc = fmt.Sprintf("v%s", cleanVer)
	}
	// Respecter la limite Discord
	if len([]rune(desc)) > maxDiscordBody {
		runes := []rune(desc)
		desc = string(runes[:maxDiscordBody]) + "…"
	}

	return Embed{
		Title:       title,
		Description: desc,
		Color:       colorVersion,
		Footer:      &EmbedFooter{Text: T(keyDiscordVersionFooter, lang)},
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// isMajorMinorChange — compare major.minor seulement
// ─────────────────────────────────────────────────────────────────────────────

// isMajorMinorChange retourne true si X ou Y ont changé dans vX.Y.Z.
// Retourne false si old est vide (premier démarrage) pour éviter le spam.
func isMajorMinorChange(old, newVer string) bool {
	if old == "" {
		return false
	}
	oMaj, oMin, oOK := parseMajorMinor(old)
	nMaj, nMin, nOK := parseMajorMinor(newVer)
	if !oOK || !nOK {
		// Fallback : comparaison string brute
		return strings.TrimSpace(old) != strings.TrimSpace(newVer)
	}
	return oMaj != nMaj || oMin != nMin
}

func parseMajorMinor(v string) (int, int, bool) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return 0, 0, false
	}
	maj, err1 := strconv.Atoi(parts[0])
	min, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return maj, min, true
}

// ─────────────────────────────────────────────────────────────────────────────
// Persistance last_notified_version (écriture atomique)
// ─────────────────────────────────────────────────────────────────────────────

func readLastNotifiedVersion(settingsPath string) string {
	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		return ""
	}
	var s map[string]any
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	v, _ := s["last_notified_version"].(string)
	return strings.TrimSpace(v)
}

// writeLastNotifiedVersion persiste last_notified_version dans app_settings.json.
//
// L'écriture passe par atomicfile.WriteFile : atomique (temp + rename) quand
// l'environnement le permet, in-place en repli quand la cible est bind-montée
// FICHIER dans un conteneur — cas de la PRODUCTION, où le rename répondait EBUSY
// et où la version notifiée n'était donc JAMAIS persistée (notification Discord
// « nouvelle version » rejouée à chaque redémarrage, constat deploy v7.2.0).
func writeLastNotifiedVersion(settingsPath, version string) error {
	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		// Créer un settings minimal si le fichier n'existe pas encore
		raw = []byte("{}")
	}
	var s map[string]any
	if err := json.Unmarshal(raw, &s); err != nil {
		s = map[string]any{}
	}
	s["last_notified_version"] = version

	updated, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal app_settings: %w", err)
	}

	return atomicfile.WriteFile(settingsPath, updated, 0o644)
}

// ─────────────────────────────────────────────────────────────────────────────
// extractWhatsNew — extrait la section changelog du README.md
// ─────────────────────────────────────────────────────────────────────────────

// extractWhatsNew cherche "**vX.Y" dans README.md (même répertoire que settingsPath)
// et retourne les bullet points jusqu'au prochain bloc **vX.Y ou ## heading.
func extractWhatsNew(version, settingsPath string) string {
	readmePath := filepath.Join(filepath.Dir(settingsPath), "README.md")
	f, err := os.Open(readmePath)
	if err != nil {
		return ""
	}
	defer f.Close()

	parts := strings.SplitN(strings.TrimPrefix(strings.TrimSpace(version), "v"), ".", 3)
	if len(parts) < 2 {
		return ""
	}
	prefix := fmt.Sprintf("**v%s.%s", parts[0], parts[1])

	scanner := bufio.NewScanner(f)
	var collected []string
	inSection := false

	for scanner.Scan() {
		line := scanner.Text()
		stripped := strings.TrimSpace(line)

		if !inSection {
			if strings.HasPrefix(stripped, prefix) {
				inSection = true
				collected = append(collected, stripped)
			}
			continue
		}

		// Fin de section : nouveau bloc **vX.Y ou heading ##
		if strings.HasPrefix(stripped, "**v") && len(collected) > 0 && stripped != collected[0] {
			break
		}
		if strings.HasPrefix(stripped, "## ") {
			break
		}
		collected = append(collected, stripped)
	}

	return strings.TrimSpace(strings.Join(collected, "\n"))
}
