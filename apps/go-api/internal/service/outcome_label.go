// Package service — LA CLÉ DE L'ISSUE D'UN MATCH, résolue par le TITRE.
//
// Le libellé d'une victoire, d'une défaite, d'une égalité ou d'un abandon sortait d'une map Go
// écrite en FRANÇAIS. Sous UI anglaise, l'en-tête de la Match View et l'écran de fin du rejeu
// annonçaient donc « Victoire » — pendant que le même panneau, vu depuis un adversaire, prenait
// son titre dans `outcomes.toml` et disait « Loss ». Deux vocabulaires sur une seule surface.
//
// Décision D5 (2026-09-07) : le Go ne sert JAMAIS de texte pour l'issue d'un match — il sert la
// CLÉ CANONIQUE (`win` | `loss` | `tie` | `dnf`, MT-06), obtenue via `mappings.Canonical(rawCode)`
// sur le jeu d'outcomes du titre (`config/titles/{slug}/mappings/outcomes.toml`). Le web localise
// avec `useOutcomeLabel` / `useOutcomeMapping` (apps/web/src/lib/i18n/fieldMappings.ts). L'ancienne
// option transitoire (le Go localise lui-même depuis le TOML, `resolveOutcomeLabel`) est abandonnée
// avec son kill-switch et la map FR de repli.
//
// SEULE EXCEPTION : l'export CSV de l'historique (handlers/match_history.go, Export) est un
// fichier rendu SERVEUR sans JS pour localiser — il a besoin d'un texte. `outcomeText` sert
// cette unique surface, en lisant le libellé DEPUIS l'adapter sémantique du titre (jamais une
// map Go) via `MatchHistoryService.OutcomeText`.
package service

import (
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/mappings"
)

// outcomesOf extrait le jeu d'outcomes de l'adapter sémantique d'un titre. nil-safe des deux
// côtés : adapter non câblé ou titre sans `outcomes.toml` → nil, et outcomeKey/outcomeText
// dégradent proprement (Canonical/Get sont nil-safe sur *mappings.OutcomeMappingSet).
func outcomesOf(semantic games.TitleSemanticAdapter) *mappings.OutcomeMappingSet {
	if semantic == nil {
		return nil
	}
	return semantic.Outcomes()
}

// outcomeKey traduit un code brut d'issue en CLÉ CANONIQUE (win|loss|tie|dnf) via le jeu
// d'outcomes du TITRE — le chokepoint unique du dépôt pour tout DTO qui expose l'issue d'un
// match. "" si le titre n'a pas de jeu d'outcomes câblé ou que le code brut n'y est pas mappé :
// ce n'est pas un repli, il n'y a rien à traduire (omitempty côté JSON) — le web dégrade son
// affichage, jamais un tiret ni un mot fabriqué côté serveur.
func outcomeKey(outcomes *mappings.OutcomeMappingSet, code int) string {
	key, ok := outcomes.Canonical(code)
	if !ok {
		return ""
	}
	return string(key)
}

// outcomeText résout le TEXTE de l'issue depuis le titre, dans la locale donnée — réservé à
// l'export CSV (seule surface de ce lot qui rend du texte côté serveur). Code brut → clé
// canonique → libellé du TOML. Titre sans mapping ou code inconnu → "" (dégradation propre,
// jamais un mot français en dur) ; clé mappée sans libellé exploitable → la clé elle-même.
func outcomeText(outcomes *mappings.OutcomeMappingSet, locale string, code int) string {
	key, ok := outcomes.Canonical(code)
	if !ok {
		return ""
	}
	return outcomeTextByKey(outcomes, locale, string(key))
}

// outcomeTextByKey résout le texte d'une clé canonique DÉJÀ CONNUE (pas de retraversée
// Canonical(rawCode)) — utilisé par l'accueil, où canonical.Outcome est déjà porté par la
// ligne (canonical.PlayerMatchRow.Self.Outcome). "" si key est vide ; la clé elle-même si le
// titre ne mappe pas cette clé ou n'a pas de libellé exploitable dans la locale.
func outcomeTextByKey(outcomes *mappings.OutcomeMappingSet, locale, key string) string {
	if key == "" {
		return ""
	}
	mapping, found := outcomes.Get(key)
	if !found {
		return key
	}
	if label, _ := mapping.Label(locale); label != "" {
		return label
	}
	return key
}

// outcomeKeyFromHaloCode traduit un code Halo BRUT (domain.Outcome*) en clé canonique MT-06,
// pour les DTO dont le service n'a PAS (encore) d'adapter sémantique câblé (CareerService,
// ExplorerService — dette multi-titre existante, hors périmètre de ce lot : cf.
// .ai/PLAN_LIBELLES_EN_DUR_GO_2026-09-07.md §9). Halo uniquement : un futur titre aux codes
// différents devra câbler son adapter sémantique avant d'appeler ces DTO.
func outcomeKeyFromHaloCode(code int) string {
	switch code {
	case domain.OutcomeWin:
		return string(canonical.OutcomeWin)
	case domain.OutcomeLoss:
		return string(canonical.OutcomeLoss)
	case domain.OutcomeDraw:
		return string(canonical.OutcomeTie)
	case domain.OutcomeDNF:
		return string(canonical.OutcomeDNF)
	default:
		return ""
	}
}
