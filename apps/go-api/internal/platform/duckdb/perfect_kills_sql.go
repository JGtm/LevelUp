package duckdb

import (
	"strings"

	"levelup/go-api/internal/analysis"
	title "levelup/go-api/internal/domain/title"
)

// pdbTitleSlug retourne le slug de titre effectif d'un PlayerDB, avec fallback
// sur title.DefaultSlug (halo_infinite) si pdb est nil ou le slug vide. Centralise
// le pattern (déjà présent dans HomeRepo.titleSlug / CareerRepo.titleSlug) pour
// les repos qui n'ont pas de helper local.
func pdbTitleSlug(pdb *PlayerDB) string {
	if pdb == nil {
		return title.DefaultSlug
	}
	if trimmed := strings.TrimSpace(pdb.TitleSlug); trimmed != "" {
		return trimmed
	}
	return title.DefaultSlug
}

// perfectKillMedalInClause résout le fragment SQL « col fait partie des médailles
// frag-parfait du titre `slug` » via la source unique analysis.PerfectKillMedalIDs.
// Évite le magic number `1512363953` codé en dur dans les agrégats perfect_kills
// des repos / requêtes statiques.
//
// `col` doit être un littéral de confiance (nom de colonne, jamais d'entrée
// utilisateur) ; seuls des ids int64 CONSTANTES sont embarqués dans le IN (...).
// Byte-identique pour halo_infinite (`col IN (1512363953)` ≡ `col = 1512363953`).
func perfectKillMedalInClause(col, slug string) string {
	return analysis.PerfectKillMedalInClause(col, slug)
}

// perfectKillToken est le marqueur SQL inséré dans les requêtes STATIQUES (const)
// à la place d'un littéral `medal_name_id = 1512363953`. Il est résolu au moment
// de l'exécution par resolvePerfectKillClause vers le fragment IN (...) du titre.
const perfectKillToken = "/*__PERFECT_KILL_IN__*/"

// resolvePerfectKillClause remplace toutes les occurrences de perfectKillToken
// dans `query` par le fragment `col IN (...)` du titre `slug`. Permet de garder
// des requêtes `const` lisibles tout en restant title-aware au runtime.
func resolvePerfectKillClause(query, col, slug string) string {
	return strings.ReplaceAll(query, perfectKillToken, perfectKillMedalInClause(col, slug))
}
