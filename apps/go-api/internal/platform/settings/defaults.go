// Package settings — defaults.go : valeurs par défaut d'app_settings.json et
// réapplication des défauts pour les clés ABSENTES du fichier.
//
// Extrait de store.go le 2026-09-15 (ADR 0035 D5) : le mode « défauts sûrs »
// ajoute de la logique de décision à ce point, qui n'a rien à voir avec les I/O
// du store (seuil fichier ≤ 500 L, CLAUDE.md règle 5).
package settings

import "encoding/json"

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
	s.enforcedDefaults = enforced
	return s
}

// applyAbsentDefaults réapplique les défauts « clé absente → true » sur un
// AppSettings dérivé de la map raw donnée (rétrocompatibilité fichiers existants).
// Partagé entre Load et ResolveForTitle pour garder une seule source de vérité.
//
// `enforced` (cf. Store.WithEnforcedDefaults, ADR 0035 D5) bascule les deux clés
// de sécurité sur leur défaut SÛR quand la propriété des joueurs est appliquée.
// Une clé PRÉSENTE dans le fichier n'est jamais touchée, dans un mode comme dans
// l'autre.
func applyAbsentDefaults(cfg *AppSettings, raw map[string]json.RawMessage, enforced bool) {
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
	if _, ok := raw["replay_sound_variation_percent"]; !ok {
		// 100 = les fourchettes du jeu telles quelles. 0 est un réglage LÉGITIME
		// (variation coupée) : sans ce ré-application, un fichier sans la clé serait
		// indiscernable d'un opérateur ayant délibérément mis 0.
		cfg.ReplaySoundVariationPercent = 100
	}
}

// Defaults retourne les valeurs par défaut de app_settings.json.
func Defaults() *AppSettings {
	return defaultSettings()
}

// defaultSettings retourne les valeurs par défaut de app_settings.json.
func defaultSettings() *AppSettings {
	return &AppSettings{
		Lang:                "en",
		DiscordLang:         "fr",
		UserTimezone:        "Europe/Paris",
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
	}
}
