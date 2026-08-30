// Package skill — objective_participation.go : loader de la métrique ospm
// (`objective_participation`) du score de performance relatif.
//
// **Quoi** : par match, la somme des `award_score` de catégorie `objective`
// (personal_score_awards, DB joueur). Divisée par la durée du match, elle donne les
// points d'objectif par minute — la contribution du joueur à l'objectif, là où les
// métriques de combat ne voient qu'un porteur de drapeau qui meurt beaucoup.
//
// **Pourquoi ici** : jumeau d'exclusion_filter.go — un loader playerDB consommé par
// le batch de notes (sync/performance.go) et par lui seul. Le package skill est la
// feuille des helpers de scoring ; il porte déjà isSchemaMissingErr, que ce loader
// réutilise au lieu d'en poser une 3e copie.
//
// Décision D-C/D-J du plan .ai/PLAN_PERF_NOTE_OBJECTIFS.md (2026-08-27).
package skill

import (
	"context"
	"database/sql"
	"log/slog"
)

// objectiveAwardCategory — valeur de `award_category` portant les actions
// d'objectif (capture, retour de drapeau, temps de possession...) par opposition
// aux catégories de combat. Littéral de la source Personal Scores API.
const objectiveAwardCategory = "objective"

// objectiveParticipationSQL — somme des points d'objectif par match, pour un joueur.
//
// Source : la VUE `personal_score_awards_latest` (ADR 0026), JAMAIS la table brute.
// La vue applique la sémantique append-only : génération MAX par (match_id, xuid)
// via DENSE_RANK, tombstones exclus. Lire la table rendrait les générations périmées
// et gonflerait les sommes.
//
// Filtre xuid STRICT, aligné sur le lecteur canonique de production
// (platform/duckdb/personal_score_awards_repo.go, `psa.xuid IN (...)`) : les lignes
// à xuid vide constatées sur d'anciennes DB (15 chez un joueur du corpus) sont donc
// ignorées ici comme elles le sont partout ailleurs.
//
// Cast ::DOUBLE explicite : SUM d'une colonne INTEGER rend un HUGEINT en DuckDB,
// que le driver ne sait pas scanner dans un float64.
const objectiveParticipationSQL = `
	SELECT match_id,
	       COALESCE(SUM(CASE WHEN award_category = ? THEN COALESCE(award_score, 0) ELSE 0 END), 0)::DOUBLE
	  FROM personal_score_awards_latest
	 WHERE xuid = ?
	 GROUP BY match_id`

// LoadObjectiveParticipation rend, par match_id, les points d'objectif du joueur.
//
// SÉMANTIQUE DE PRÉSENCE (D-J) — les deux cas ne se confondent pas :
//   - match ABSENT de la map = aucune ligne PSA retenue pour lui : la couverture
//     manque (match jamais enrichi, extraction vide/tombstonée). La métrique ospm
//     est alors ABSENTE du calcul et son poids est redistribué sur les autres ;
//   - match PRÉSENT avec la valeur 0 = match couvert où le joueur n'a marqué aucun
//     point d'objectif. C'est une VALEUR, classée par percentile comme une autre.
//
// Traiter le second cas comme le premier reviendrait à offrir la note d'un match
// « non mesuré » à qui n'a rien fait à l'objectif ; l'inverse écraserait la note de
// tous les matchs non enrichis. La valeur pointée n'est jamais nil.
//
// Dégradation gracieuse et title-agnostic : sur un titre ou une DB legacy sans
// personal_score_awards (donc sans la vue `_latest`), la requête échoue sur un
// schéma manquant → map vide + log Debug, aucune erreur remontée. Le batch calcule
// alors les notes sans ospm, exactement comme avant ce lot. Aucun test de slug :
// c'est la présence de la donnée qui décide.
//
// Best-effort assumé : une panne SQL autre que « schéma absent » est LOGUÉE en
// Warn puis dégradée de la même façon — une métrique optionnelle ne doit pas faire
// échouer le calcul de toutes les notes du joueur.
func LoadObjectiveParticipation(ctx context.Context, playerDB *sql.DB, xuid string) map[string]*float64 {
	out := make(map[string]*float64)
	rows, err := playerDB.QueryContext(ctx, objectiveParticipationSQL, objectiveAwardCategory, xuid)
	if err != nil {
		if isSchemaMissingErr(err) {
			slog.DebugContext(ctx, "LoadObjectiveParticipation: personal_score_awards_latest absente — métrique objective_participation désactivée",
				"xuid", xuid, "err", err)
			return out
		}
		slog.WarnContext(ctx, "LoadObjectiveParticipation: lecture des awards échouée — notes calculées sans objective_participation",
			"xuid", xuid, "err", err)
		return out
	}
	defer rows.Close()

	scanErrors := 0
	for rows.Next() {
		var matchID string
		var objectiveScore float64
		if err := rows.Scan(&matchID, &objectiveScore); err != nil {
			scanErrors++
			continue
		}
		v := objectiveScore
		out[matchID] = &v
	}
	if err := rows.Err(); err != nil {
		slog.WarnContext(ctx, "LoadObjectiveParticipation: itération des awards interrompue — couverture partielle",
			"xuid", xuid, "loaded", len(out), "err", err)
	}
	if scanErrors > 0 {
		slog.WarnContext(ctx, "LoadObjectiveParticipation: scan errors sur les awards",
			"xuid", xuid, "scan_errors", scanErrors, "loaded", len(out))
	}
	return out
}
