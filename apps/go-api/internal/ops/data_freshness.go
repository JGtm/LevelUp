// Package ops — data_freshness.go : évaluation PURE de la fraîcheur des
// données par joueur suivi (plan monitoring A4, seuils DC-3).
//
// Sémantique DC-3 : un joueur est signalé quand ses données sont VIEILLES
// **ET** que le moteur de sync ne tourne plus — un joueur qui ne joue pas mais
// dont le sync réussit régulièrement est à jour (statut ok). Si l'âge du
// dernier cycle sync réussi est INCONNU (titre hors scheduler V2 — ex. Halo 5
// via liveRunner), l'âge du dernier match persisté fait foi seul : c'est
// précisément le trou de visibilité que ce panneau couvre.
//
// Les seuils viennent d'app_settings.json (FreshnessThresholdsFromSettings),
// jamais en dur dans les callers.
package ops

import (
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
)

// Défauts DC-3 (surclassables par app_settings.json).
const (
	defaultFreshnessWarnMatchHours    = 48
	defaultFreshnessWarnSyncHours     = 6
	defaultFreshnessCriticalMatchDays = 7
)

// Clés app_settings.json des seuils de fraîcheur.
const (
	settingFreshnessWarnMatchHours    = "freshness_warn_match_hours"
	settingFreshnessWarnSyncHours     = "freshness_warn_sync_hours"
	settingFreshnessCriticalMatchDays = "freshness_critical_match_days"
)

// FreshnessThresholds paramètre l'évaluation (DC-3).
type FreshnessThresholds struct {
	// WarnMatchAge : âge du dernier match au-delà duquel on passe warn
	// (si le sync ne tourne plus). Défaut 48 h.
	WarnMatchAge time.Duration
	// WarnSyncAge : âge du dernier cycle sync réussi en-deçà duquel le joueur
	// est considéré à jour quoi qu'il arrive. Défaut 6 h.
	WarnSyncAge time.Duration
	// CriticalMatchAge : âge du dernier match au-delà duquel on passe critical.
	// Défaut 7 j.
	CriticalMatchAge time.Duration
}

// DefaultFreshnessThresholds retourne les seuils DC-3 par défaut.
func DefaultFreshnessThresholds() FreshnessThresholds {
	return FreshnessThresholds{
		WarnMatchAge:     defaultFreshnessWarnMatchHours * time.Hour,
		WarnSyncAge:      defaultFreshnessWarnSyncHours * time.Hour,
		CriticalMatchAge: defaultFreshnessCriticalMatchDays * 24 * time.Hour,
	}
}

// FreshnessThresholdsFromSettings surcharge les défauts depuis app_settings.json
// (valeurs numériques JSON → float64). Clés absentes/invalides = défaut.
func FreshnessThresholdsFromSettings(settings map[string]interface{}) FreshnessThresholds {
	th := DefaultFreshnessThresholds()
	if h, ok := settingHours(settings, settingFreshnessWarnMatchHours); ok {
		th.WarnMatchAge = h
	}
	if h, ok := settingHours(settings, settingFreshnessWarnSyncHours); ok {
		th.WarnSyncAge = h
	}
	if h, ok := settingHours(settings, settingFreshnessCriticalMatchDays); ok {
		th.CriticalMatchAge = h * 24
	}
	return th
}

// settingHours lit une clé numérique (heures ou jours selon la clé) > 0.
func settingHours(settings map[string]interface{}, key string) (time.Duration, bool) {
	raw, ok := settings[key]
	if !ok {
		return 0, false
	}
	switch v := raw.(type) {
	case float64:
		if v > 0 {
			return time.Duration(v * float64(time.Hour)), true
		}
	case int:
		if v > 0 {
			return time.Duration(v) * time.Hour, true
		}
	}
	return 0, false
}

// PlayerFreshnessInput est l'entrée PURE de l'évaluation d'un joueur.
type PlayerFreshnessInput struct {
	Gamertag string
	XUID     string
	// LastMatchAt : dernier match persisté (timestamp canonique) ; nil = aucun.
	LastMatchAt *time.Time
	// LastSyncOKAt : dernier cycle sync réussi pour ce joueur ; nil = inconnu
	// (titre hors scheduler, ou jamais couru depuis le boot).
	LastSyncOKAt *time.Time
	// CheckError : DB inaccessible pour ce joueur (statut unknown).
	CheckError string
}

// EvaluatePlayerFreshness applique DC-3 à un joueur. Pure (now injecté).
func EvaluatePlayerFreshness(in PlayerFreshnessInput, now time.Time, th FreshnessThresholds) domain.PlayerFreshness {
	out := domain.PlayerFreshness{
		Gamertag:   in.Gamertag,
		XUID:       in.XUID,
		CheckError: in.CheckError,
	}
	if in.LastMatchAt != nil {
		out.LastMatchAt = in.LastMatchAt.UTC().Format(time.RFC3339)
		out.MatchAgeSeconds = int64(now.Sub(*in.LastMatchAt).Seconds())
	}
	if in.LastSyncOKAt != nil {
		out.LastSyncOKAt = in.LastSyncOKAt.UTC().Format(time.RFC3339)
		out.SyncAgeSeconds = int64(now.Sub(*in.LastSyncOKAt).Seconds())
	}
	if in.CheckError != "" {
		out.Status = domain.FreshnessStatusUnknown
		return out
	}
	// Moteur vivant = données à jour, même si le joueur ne joue pas (DC-3 : le
	// WARN exige match vieux ET sync vieux).
	if in.LastSyncOKAt != nil && now.Sub(*in.LastSyncOKAt) <= th.WarnSyncAge {
		out.Status = domain.FreshnessStatusOK
		return out
	}
	// Sync vieux ou inconnu : l'âge du dernier match fait foi.
	switch {
	case in.LastMatchAt == nil:
		out.Status = domain.FreshnessStatusCritical
		out.Reason = "aucun match persisté (jamais synchronisé)"
	case now.Sub(*in.LastMatchAt) > th.CriticalMatchAge:
		out.Status = domain.FreshnessStatusCritical
		out.Reason = fmt.Sprintf("dernier match > %s et sync inactif", th.CriticalMatchAge)
	case now.Sub(*in.LastMatchAt) > th.WarnMatchAge:
		out.Status = domain.FreshnessStatusWarn
		out.Reason = fmt.Sprintf("dernier match > %s et sync inactif", th.WarnMatchAge)
	default:
		out.Status = domain.FreshnessStatusOK
	}
	return out
}
