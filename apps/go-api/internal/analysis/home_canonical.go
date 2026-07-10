// Package analysis â€” home_canonical.go : entry-points canonical-aware pour la
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
// **DÃ©coupage thÃ©matique** (split god-file 2026-05) :
//   - home_canonical_converters.go : conversion canonical â†’ legacy.
//   - home_canonical_kpis.go       : KPIs, tendance, hero card.
//   - home_canonical_highlights.go : 8 tuiles highlights.
//   - home_canonical_recent.go     : BuildRecentMatchesWithFavorites + score label.
//   - home_canonical_sessions.go   : BuildSessionSummary[ies].
//   - home_canonical_skill.go      : InferHomeSkillHistory + badge CSR.
//
// Ce fichier conserve uniquement les helpers transverses utilisÃ©s par plusieurs
// sous-modules (deref, conversion outcome, extraction de labels d'assets,
// rÃ©duction par frÃ©quence).
package analysis

import (
	"strings"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// derefIntZero retourne *p ou 0 si p est nil. Utilitaire interne.
func derefIntZero(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// canonicalOutcomeToInt convertit canonical.Outcome â†’ int Halo.
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

// assetLabels extrait (en/fr) d'une AssetReference canonical (nil-safe).
// Les UUIDs bruts sont filtrés via cleanAssetLabel. Bug #6.
func assetLabels(ref *canonical.AssetReference) (en, fr string) {
	if ref == nil {
		return "", ""
	}
	en = cleanAssetLabel(ref.DefaultLabel)
	if v, ok := ref.Labels["en"]; ok {
		if cleaned := cleanAssetLabel(v); cleaned != "" {
			en = cleaned
		}
	}
	if v, ok := ref.Labels["fr"]; ok {
		fr = cleanAssetLabel(v)
	}
	return en, fr
}

// modeLabels privilégie PairMode (FR/EN dispo en DB) puis fallback GameVariant
// (FR jamais peuplé en DB). Bug #7.
func modeLabels(r canonical.PlayerMatchRow) (en, fr string) {
	if r.Summary.PairMode != nil {
		en, fr = assetLabels(r.Summary.PairMode)
		if en != "" || fr != "" {
			return en, fr
		}
	}
	return assetLabels(r.Summary.GameVariant)
}

// dominantNameFromRows agrège la fréquence d'un label par row et retourne le
// nom dominant dans la locale demandée. Bug #2.
func dominantNameFromRows(
	rows []canonical.PlayerMatchRow,
	locale string,
	extractor func(canonical.PlayerMatchRow) (string, string),
) *string {
	type counts struct {
		en, fr string
		count  int
	}
	freq := make(map[string]*counts)
	for _, r := range rows {
		en, fr := extractor(r)
		key := en
		if key == "" {
			key = fr
		}
		if key == "" {
			continue
		}
		c, ok := freq[key]
		if !ok {
			c = &counts{en: en, fr: fr}
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
	name := labelForLocale(locale, best.fr, best.en)
	if name == "" {
		return nil
	}
	return &name
}
