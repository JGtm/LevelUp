// Package analysis — citation_snippets.go : calcul des snippets de citations pour MatchCard.
// Fonctions pures, sans accès DB ni Streamlit.
package analysis

import (
	"net/url"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/domain"
)

// ParseTierTargets convertit une chaîne CSV de paliers en slice d'entiers croissants.
// Ex: "10,20,30,50,100" → [10, 20, 30, 50, 100].
// Les valeurs non numériques (ou <= 0) sont ignorées silencieusement.
func ParseTierTargets(s string) []int {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	tiers := make([]int, 0, len(parts))
	for _, p := range parts {
		v, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil || v <= 0 {
			continue
		}
		tiers = append(tiers, v)
	}
	sort.Ints(tiers)
	return tiers
}

// computeTierProgress calcule le pourcentage de progression vers le prochain palier.
// Formule : (total - prevTier) / (currTier - prevTier) × 100.
// Retourne 100.0 si total >= dernier palier (masterisé).
// Retourne 0.0 si tiers est vide ou total <= 0.
func computeTierProgress(total int, tiers []int) float64 {
	if len(tiers) == 0 {
		return 0
	}
	lastTier := tiers[len(tiers)-1]
	if total >= lastTier {
		return 100.0
	}
	prevTier := 0
	for _, t := range tiers {
		if total < t {
			width := t - prevTier
			if width <= 0 {
				return 0
			}
			return float64(total-prevTier) / float64(width) * 100.0
		}
		prevTier = t
	}
	return 100.0
}

// TierProgression regroupe les indicateurs de progression d'un palier, calculés à
// partir d'un cumul absolu, d'un delta de match et d'une liste de paliers croissants.
// Réutilisé par les citations dérivées (Halo Infinite) ET les commendations natives
// (Halo 5) — même mécanique d'anneau de progression côté frontend (CitationProgressRing).
type TierProgression struct {
	ProgressPct     float64
	IsNewlyMastered bool
	TierIndex       int // nombre de paliers atteints (0..len(tiers))
	TierCount       int // nombre total de paliers (longueur de tier_targets)
	NextTierTarget  int // seuil absolu du prochain palier (0 si maîtrisé)
	// AlreadyMastered : le palier final était déjà franchi AVANT ce match
	// (cumulative - delta >= dernier palier) → snippet à filtrer (rien de neuf).
	AlreadyMastered bool
}

// ComputeTierProgression calcule la progression de palier d'une citation/commendation.
// `cumulative` = total absolu APRÈS ce match ; `delta` = ce qui a été gagné CE match ;
// `tiers` = paliers croissants (cf. ParseTierTargets). Fonction pure, sans accès DB.
func ComputeTierProgression(cumulative, delta int, tiers []int) TierProgression {
	tp := TierProgression{TierCount: len(tiers)}
	before := cumulative - delta
	if len(tiers) > 0 {
		lastTier := tiers[len(tiers)-1]
		tp.AlreadyMastered = before >= lastTier
		tp.IsNewlyMastered = cumulative >= lastTier && before < lastTier
	}
	tp.ProgressPct = computeTierProgress(cumulative, tiers)
	for _, t := range tiers {
		if cumulative >= t {
			tp.TierIndex++
		} else {
			tp.NextTierTarget = t
			break
		}
	}
	return tp
}

// BuildCitationSnippets construit au plus `limit` snippets de citations depuis les lignes brutes.
// Filtre les citations déjà masterisées AVANT ce match.
// Trie par delta décroissant.
func BuildCitationSnippets(rows []domain.HomeMatchCitationRaw, limit int) []domain.MatchCitationSnippet {
	if len(rows) == 0 {
		return nil
	}

	snippets := make([]domain.MatchCitationSnippet, 0, len(rows))
	for _, row := range rows {
		// Ignorer les norms sans métadonnées (ex: _processed, flags internes).
		if row.Display == "" {
			continue
		}

		tiers := ParseTierTargets(row.TierTargets)
		tp := ComputeTierProgression(row.Cumulative, row.Delta, tiers)
		if tp.AlreadyMastered {
			// Déjà masterisé avant ce match → on n'affiche pas.
			continue
		}

		var imgURL *string
		if row.ImagePath != "" {
			// image_path en DB : "static/commendations/halo_5_guardians/FILENAME" (sans / initial).
			// Les fichiers sur disk ont des noms URL-encodés littéraux (ex: %C3%89).
			// url.PathEscape encode % → %25, ce qui permet à Go FileServer de retrouver
			// le fichier par son nom littéral après décodage d'un seul niveau. ✓
			parts := strings.Split(row.ImagePath, "/")
			encoded := make([]string, len(parts))
			for i, seg := range parts {
				encoded[i] = url.PathEscape(seg)
			}
			p := "/" + strings.Join(encoded, "/")
			imgURL = &p
		}

		var desc *string
		if row.Description != "" {
			d := row.Description
			desc = &d
		}

		snippets = append(snippets, domain.MatchCitationSnippet{
			Key:             row.Norm,
			Name:            row.Display,
			Description:     desc,
			ImageURL:        imgURL,
			Delta:           row.Delta,
			ProgressPct:     tp.ProgressPct,
			IsNewlyMastered: tp.IsNewlyMastered,
			Cumulative:      row.Cumulative,
			TierIndex:       tp.TierIndex,
			TierCount:       tp.TierCount,
			NextTierTarget:  tp.NextTierTarget,
		})
	}

	// Trier par delta décroissant.
	sort.Slice(snippets, func(i, j int) bool {
		return snippets[i].Delta > snippets[j].Delta
	})

	if len(snippets) > limit {
		snippets = snippets[:limit]
	}
	return snippets
}
