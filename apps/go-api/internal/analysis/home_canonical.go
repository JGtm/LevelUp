// Package analysis — home_canonical.go : entry-points canonical-aware pour la
// page Home (P4.3b, ADR 0011).
//
// Les *FromCanonical sont les entry-points de la page Home : ils operent
// DIRECTEMENT sur les types canonical. Il n'y a PLUS de delegation a une couche
// legacy -- les anciennes fonctions publiques (ComputeKPIs, ComputeTrend,
// BuildHeroCard, BuildHighlights, BuildRecentMatches*, BuildSessionSummaries),
// jadis "source de verite legacy", ont ete SUPPRIMEES (G2, 2026-07-03) : plus
// aucun caller, les *FromCanonical etant devenus autonomes.
//
//   - Le converter HomeMatchRowFromCanonical reste encapsule dans le package
//     (invisible cote service/home_service.go, qui consomme uniquement les
//     *FromCanonical).
//   - Residu legacy conserve : types legacymatch.HomeMatchRow / HomeSessionRow +
//     helpers partages (dominantKey, selectHighlightWindow, mapImageURLFromRegistry,
//     distinctSessionLabels...) tant que port.HomeRepository.LoadHomeMatches
//     retourne du legacy et que des callers paralleles (squad/teammates) les consomment.
//
// **Découpage thématique** (split god-file 2026-05) :
//   - home_canonical_converters.go : conversion canonical → legacy.
//   - home_canonical_kpis.go       : KPIs, tendance, hero card.
//   - home_canonical_highlights.go : 8 tuiles highlights.
//   - home_canonical_recent.go     : BuildRecentMatchesWithFavorites + score label.
//   - home_canonical_sessions.go   : BuildSessionSummary[ies].
//   - home_canonical_skill.go      : InferHomeSkillHistory + badge CSR.
//
// Ce fichier conserve uniquement les helpers transverses utilisés par plusieurs
// sous-modules (deref, conversion outcome, extraction de labels d'assets,
// réduction par fréquence).
package analysis

import (
	"fmt"
	"strings"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/i18n"
)

// derefIntZero retourne *p ou 0 si p est nil. Utilitaire interne.
func derefIntZero(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// canonicalOutcomeToInt convertit canonical.Outcome → int Halo.
func canonicalOutcomeToInt(o canonical.Outcome) int {
	switch o {
	case canonical.OutcomeWin:
		return domain.OutcomeWin
	case canonical.OutcomeLoss:
		return domain.OutcomeLoss
	case canonical.OutcomeTie:
		return domain.OutcomeDraw
	case canonical.OutcomeDNF:
		return domain.OutcomeDNF
	}
	return 0
}

// cleanAssetLabel filtre les valeurs UUID brutes (cas où la metadata
// translation est manquante et le sync stocke l'asset ID comme fallback).
// Bug #6.
func cleanAssetLabel(value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return ""
	}
	if IsRawAssetUUID(v) {
		return ""
	}
	return v
}

// assetText rassemble les libellés d'une AssetReference canonical en un i18n.Text (nil-safe) :
// AssetReference.Text, dont chaque valeur passe par cleanAssetLabel (les UUIDs bruts sont filtrés,
// une valeur filtrée n'est pas inscrite, et le libellé par défaut nettoyé reprend la place de la
// langue de repli quand le sien est filtré). Bug #6.
func assetText(ref *canonical.AssetReference) i18n.Text {
	out := i18n.Text{}
	if ref == nil {
		return out
	}
	for l, v := range ref.Text() {
		if cleaned := cleanAssetLabel(v); cleaned != "" {
			out[l] = cleaned
		}
	}
	if out[i18n.Fallback] == "" {
		if cleaned := cleanAssetLabel(ref.DefaultLabel); cleaned != "" {
			out[i18n.Fallback] = cleaned
		}
	}
	return out
}

// modeText rend les libellés du mode du match : Summary.PairMode, fourni par la source de lecture
// du titre (partie pour Halo Infinite, type de jeu pour Halo 5). Aucun repli sur GameVariant : un
// mode sans libellé reste vide (ratchet archlint/mode_gamevariant_fallback_test.go).
func modeText(r canonical.PlayerMatchRow) i18n.Text {
	return assetText(r.Summary.PairMode)
}

// textValues retourne les valeurs non vides d'un Text dans l'ordre alphabétique des codes de
// langue (i18n.FlatCodes) : la liste des libellés d'une même carte que NormalizeModeLabel retire
// d'un nom de mode, quelle que soit la langue.
func textValues(t i18n.Text) []string {
	var out []string
	for _, l := range i18n.FlatCodes() {
		if v := t[l]; v != "" {
			out = append(out, v)
		}
	}
	return out
}

// textKey retourne la valeur d'un Text qui sert de clé de regroupement : celle de la langue de
// repli, à défaut la première non vide dans l'ordre alphabétique des codes de langue.
func textKey(t i18n.Text) string {
	if v := t[i18n.Fallback]; v != "" {
		return v
	}
	for _, l := range i18n.FlatCodes() {
		if v := t[l]; v != "" {
			return v
		}
	}
	return ""
}

// dominantNameFromRows agrège la fréquence d'un label par row et retourne le
// nom dominant dans la locale demandée. Bug #2.
func dominantNameFromRows(
	rows []canonical.PlayerMatchRow,
	locale string,
	extractor func(canonical.PlayerMatchRow) i18n.Text,
) *string {
	type counts struct {
		name  i18n.Text
		count int
	}
	freq := make(map[string]*counts)
	for _, r := range rows {
		name := extractor(r)
		key := textKey(name)
		if key == "" {
			continue
		}
		c, ok := freq[key]
		if !ok {
			c = &counts{name: name}
			freq[key] = c
		}
		c.count++
	}
	if len(freq) == 0 {
		return nil
	}
	var best *counts
	var bestKey string
	for key, c := range freq {
		switch {
		case best == nil:
			best = c
			bestKey = key
		case c.count > best.count:
			best = c
			bestKey = key
		case c.count == best.count && key < bestKey:
			best = c
			bestKey = key
		}
	}
	picked := best.name.PickOptional(normalizeHomeLocale(locale))
	if picked == "" {
		return nil
	}
	return &picked
}

// homeMapModeDetail compose le détail « carte · mode » d'un fait marquant dans la langue de la
// requête : la carte et le mode se choisissent par Text.PickOptional (la langue, puis la langue de
// repli, jamais le français pour l'anglais), et le nom du mode est débarrassé du libellé de la carte
// dans toutes les langues où elle en porte un.
func homeMapModeDetail(locale string, mapNames, modeNames i18n.Text) string {
	loc := normalizeHomeLocale(locale)
	return fmt.Sprintf("%s · %s", mapNames.PickOptional(loc),
		normalizeHomeModeLabel(modeNames.PickOptional(loc), textValues(mapNames)...))
}
