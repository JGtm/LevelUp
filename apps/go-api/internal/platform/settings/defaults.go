// Package settings — defaults.go : valeurs par défaut d'app_settings.json et
// réapplication des défauts pour les clés ABSENTES du fichier.
//
// Extrait de store.go le 2026-09-15 (ADR 0035 D5) : le mode « défauts sûrs »
// ajoute de la logique de décision à ce point, qui n'a rien à voir avec les I/O
// du store. L'extraction ramène store.go de 556 à 516 lignes — la dette de
// seuil (CLAUDE.md règle 5, ≤ 500 L) est réduite, pas résorbée.
package settings

import (
	"encoding/json"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
)

// absentDefaults : ce qui décide la valeur d'une clé ABSENTE d'app_settings.json au-delà des
// défauts historiques.
type absentDefaults struct {
	// enforced : l'instance applique la propriété des joueurs (WithEnforcedDefaults).
	enforced bool
	// system : fuseau et langue du système, posés sous le profil desktop (WithSystemDefaults).
	system config.SystemLocale
}

// WithEnforcedDefaults déclare que l'instance applique la propriété des joueurs
// (authz.Enforced : ni démo, ni auth_mode ouvert). Dans ce mode, une clé ABSENTE
// d'app_settings.json prend sa valeur SÛRE et non sa valeur permissive :
// `instance_locked` absent ⇒ true, `can_self_provision` absent ⇒ false.
//
// POURQUOI (ADR 0035) : le 2026-07-23, une instance de production dont le fichier
// ne portait aucune des deux clés a laissé un compte Xbox inconnu se provisionner
// par SSO. Un déploiement multi-utilisateur doit être fermé tant que son
// administrateur ne l'ouvre pas explicitement.
//
// Une valeur EXPLICITE du fichier gagne toujours (aucune réécriture du fichier :
// la production est verrouillée par son administrateur, pas par un déploiement).
// Hors mode appliqué (mono-utilisateur, auth_mode=none, démo), les défauts
// historiques restent inchangés (false / true).
func (s *Store) WithEnforcedDefaults(enforced bool) *Store {
	s.absent.enforced = enforced
	return s
}

// WithSystemDefaults déclare le fuseau et la langue du système comme défaut des clés
// `user_timezone` et `lang` ABSENTES d'app_settings.json (profil desktop : un premier lancement
// hors de France n'hérite plus d'Europe/Paris ni de l'anglais par défaut). Une valeur explicite
// du fichier gagne toujours ; un champ vide de sys laisse le défaut historique. Aucune
// réécriture du fichier : la valeur du système suit le système tant que l'utilisateur n'a rien
// enregistré.
func (s *Store) WithSystemDefaults(sys config.SystemLocale) *Store {
	s.absent.system = sys
	return s
}

// applyAbsentDefaults réapplique les défauts « clé absente → true » sur un
// AppSettings dérivé de la map raw donnée (rétrocompatibilité fichiers existants).
// Partagé entre Load et ResolveForTitle pour garder une seule source de vérité.
//
// `d.enforced` (cf. Store.WithEnforcedDefaults, ADR 0035 D5) bascule les deux clés
// de sécurité sur leur défaut SÛR quand la propriété des joueurs est appliquée.
// Une clé PRÉSENTE dans le fichier n'est jamais touchée, dans un mode comme dans
// l'autre.
func applyAbsentDefaults(cfg *AppSettings, raw map[string]json.RawMessage, d absentDefaults) {
	enforced := d.enforced
	// Langue et fuseau : une clé vide vaut absente (config.DeclaredString), comme pour /bootstrap.
	if !declaredString(raw, "lang") {
		cfg.Lang = d.system.LangOr(defaultLang)
	}
	if !declaredString(raw, "user_timezone") {
		cfg.UserTimezone = d.system.TimezoneOr(defaultTimezone)
	}
	if _, ok := raw["instance_locked"]; !ok && enforced {
		cfg.InstanceLocked = true
	}
	if _, ok := raw["can_self_provision"]; !ok {
		cfg.CanSelfProvision = !enforced
	}
	if _, ok := raw["can_start_initial_sync"]; !ok {
		cfg.CanStartInitialSync = true
	}
	if _, ok := raw["show_progression"]; !ok {
		cfg.ShowProgression = true
	}
	if _, ok := raw["coach_proactive_mode"]; !ok {
		cfg.CoachProactiveMode = true // DEC-2 : défaut ON (bascule 2026-07-22)
	}
	if _, ok := raw["archive_films"]; !ok {
		cfg.ArchiveFilms = true // D-5 : archiver par défaut (un film expire côté serveur Halo)
	}
	if _, ok := raw["replay_sound_variation_percent"]; !ok {
		// 100 = les fourchettes du jeu telles quelles. 0 est un réglage LÉGITIME
		// (variation coupée) : sans ce ré-application, un fichier sans la clé serait
		// indiscernable d'un opérateur ayant délibérément mis 0.
		cfg.ReplaySoundVariationPercent = 100
	}
}

// declaredString dit si la clé chaîne est ÉCRITE dans le fichier (règle config.DeclaredString).
func declaredString(raw map[string]json.RawMessage, key string) bool {
	m, ok := raw[key]
	if !ok {
		return false
	}
	var v any
	if err := json.Unmarshal(m, &v); err != nil {
		return false
	}
	_, declared := config.DeclaredString(v)
	return declared
}

// Défauts historiques de la langue et du fuseau (clé non écrite, sans valeur du système).
const (
	defaultLang     = "en"
	defaultTimezone = "Europe/Paris"
)

// Defaults retourne les valeurs par défaut de app_settings.json.
func Defaults() *AppSettings {
	return defaultSettings()
}

// defaultSettings retourne les valeurs par défaut de app_settings.json.
func defaultSettings() *AppSettings {
	return &AppSettings{
		Lang:                defaultLang,
		DiscordLang:         "fr",
		UserTimezone:        defaultTimezone,
		MediaBufferMinutes:  2,
		CanSelfProvision:    true,
		CanStartInitialSync: true,
		// Règles de sessions
		SessionGapMinutes:     120,       // 2 heures — historique Python
		SessionTeamChangeMode: "friends", // amis seulement — moins sensible aux randoms
		// Règles de badges narratifs
		OutcomeExcludeBotMatchesFromBadges:  true,       // bots faussent les scores adverses
		OutcomeExcludeBotMatchesFromRecords: false,      // pas de changement de comportement par défaut
		OutcomeBadgeSensitivity:             "standard", // seuils historiques Python
		// Affichage Objectifs/Prestige activé par défaut
		ShowProgression: true,
		// Coach proactif activé par défaut (DEC-2, bascule 2026-07-22).
		CoachProactiveMode: true,
		// Sons du rejeu 2D : variation du jeu telle quelle, aucune distance.
		ReplaySoundVariationPercent: 100,
		ReplaySoundDistancePercent:  0,
		FilmArchiveSettings:         domain.FilmArchiveSettings{ArchiveFilms: true},
	}
}
