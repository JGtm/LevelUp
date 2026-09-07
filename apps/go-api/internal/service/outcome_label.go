// Package service — LE MOT DE L'ISSUE D'UN MATCH, résolu par le TITRE.
//
// Le libellé d'une victoire, d'une défaite, d'une égalité ou d'un abandon sortait d'une map Go
// écrite en FRANÇAIS. Sous UI anglaise, l'en-tête de la Match View et l'écran de fin du rejeu
// annonçaient donc « Victoire » — pendant que le même panneau, vu depuis un adversaire, prenait
// son titre dans `outcomes.toml` et disait « Loss ». Deux vocabulaires sur une seule surface.
//
// La source de vérité est le TOML du titre (`config/titles/{slug}/mappings/outcomes.toml`,
// projeté en `mappings.OutcomeMappingSet`) : le code brut du titre y est traduit en issue
// canonique (`Canonical`), puis l'issue en libellé de la locale demandée (`Label`). Aucun
// libellé FR/EN en dur, aucune comparaison de slug — la table EST le titre.
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/mappings"
)

// outcomeLabelUnknown est le tiret servi quand NI le titre NI le repli ne connaissent le code
// (0 = pas d'issue enregistrée sur la ligne). Ce n'est pas un repli : il n'y a rien à traduire.
const outcomeLabelUnknown = "-"

// outcomeLabels — REPLI FRANÇAIS du libellé d'issue, conservé pour les titres qui n'exposent
// pas (encore) de `raw_code` dans leur `outcomes.toml`, et pour les surfaces qui n'ont pas
// d'adapter sémantique câblé. Les quatre valeurs sont à l'octet celles de la colonne `fr` du
// TOML d'Halo Infinite : tant que le repli sert, rien ne bouge à l'écran en français.
//
// KILL-SWITCH — bascule du défaut : 2026-09-07 (le titre devient la source, ce bloc n'est plus
// que le filet). Retrait cible : 2026-12-01. Critère mesurable : 0 occurrence du log
// « outcome_label: repli sur la map FR » sur 30 jours de logs prod (le repli est instrumenté
// dans resolveOutcomeLabel). Le jour où le compteur est à zéro, ce bloc et outcomeLabel()
// partent avec leurs derniers appelants (career, explorer — cf. le rapport de lot).
var outcomeLabels = map[int]string{
	1: "Égalité",
	2: "Victoire",
	3: "Défaite",
	4: "Abandon",
}

// outcomesOf extrait le jeu d'outcomes de l'adapter sémantique d'un titre. nil-safe des deux
// côtés : adapter non câblé ou titre sans `outcomes.toml` → nil, et resolveOutcomeLabel replie.
func outcomesOf(semantic games.TitleSemanticAdapter) *mappings.OutcomeMappingSet {
	if semantic == nil {
		return nil
	}
	return semantic.Outcomes()
}

// resolveOutcomeLabel est LE chokepoint du libellé d'issue servi par l'API : le mot du titre,
// dans la locale de la requête (`ctxkeys.Locale`, « fr » par défaut).
//
// Dégradation gracieuse, jamais de panic ni de chaîne vide : titre sans jeu d'outcomes, code
// brut non mappé, ou libellé vide dans le TOML → repli sur la map FR, journalisé pour que le
// critère de retrait du kill-switch soit mesurable. Un code inconnu des DEUX (0, valeur
// aberrante) rend le tiret d'avant — ce n'est pas un repli, rien n'a été perdu.
func resolveOutcomeLabel(ctx context.Context, outcomes *mappings.OutcomeMappingSet, code int) string {
	if key, ok := outcomes.Canonical(code); ok {
		if mapping, found := outcomes.Get(string(key)); found {
			// Le fallback interne de Label (locale → en → clé) est volontairement ignoré ici :
			// « Win » vaut mieux qu'un « win » brut, et le loader exige déjà en + fr.
			if label, _ := mapping.Label(ctxkeys.Locale(ctx)); label != "" {
				return label
			}
		}
	}
	fallback := outcomeLabel(code)
	if fallback != outcomeLabelUnknown {
		slog.DebugContext(ctx, "outcome_label: repli sur la map FR",
			"code", code,
			"titleSlug", ctxkeys.TitleSlug(ctx),
			"locale", ctxkeys.Locale(ctx))
	}
	return fallback
}

// outcomeLabel est le REPLI seul (cf. outcomeLabels) : il ne connaît que le français et ne doit
// plus être appelé directement par une surface nouvelle — passer par resolveOutcomeLabel.
func outcomeLabel(code int) string {
	if label, ok := outcomeLabels[code]; ok {
		return label
	}
	return outcomeLabelUnknown
}
