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

// parseTierTargets convertit une chaîne CSV de paliers en slice d'entiers croissants.
// Ex: "10,20,30,50,100" → [10, 20, 30, 50, 100].
// Les valeurs non numériques sont ignorées silencieusement.
func parseTierTargets(s string) []int {
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
// Retourne (100.0, true) si total >= dernier palier (masterisé).
// Retourne (0.0, false) si tiers est vide ou total <= 0.
func computeTierProgress(total int, tiers []int) (pct float64, fullyMastered bool) {
	if len(tiers) == 0 {
		return 0, false
	}
	lastTier := tiers[len(tiers)-1]
	if total >= lastTier {
		return 100.0, true
	}
	prevTier := 0
	for _, t := range tiers {
		if total < t {
			width := t - prevTier
			if width <= 0 {
				return 0, false
			}
			return float64(total-prevTier) / float64(width) * 100.0, false
		}
		prevTier = t
	}
	return 100.0, true
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

		tiers := parseTierTargets(row.TierTargets)

		before := row.Cumulative - row.Delta
		if len(tiers) > 0 {
			lastTier := tiers[len(tiers)-1]
			if before >= lastTier {
				// Déjà masterisé avant ce match → on n'affiche pas.
				continue
			}
		}

		pct, _ := computeTierProgress(row.Cumulative, tiers)

		isNewlyMastered := false
		if len(tiers) > 0 {
			lastTier := tiers[len(tiers)-1]
			isNewlyMastered = row.Cumulative >= lastTier && before < lastTier
		}

		var imgURL *string
		if row.ImagePath != "" {
			// image_path en DB : "static/commendations/h5g/FILENAME" (sans / initial).
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
			ProgressPct:     pct,
			IsNewlyMastered: isNewlyMastered,
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
