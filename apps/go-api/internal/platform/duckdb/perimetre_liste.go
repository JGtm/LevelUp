package duckdb

// perimetre_liste.go — LA LISTE DES MATCHS D'UNE LECTURE, LIÉE EN UNE SEULE CONSTANTE (lot B du
// plan perf « compaction et périmètre joueur », 2026-09-27, ADR 0036 I2).
//
// Les lectures d'historique complet (Carrière Q26 / Q27, Relations Q28, Tactique MortsParCarte)
// bornent chaque vue `_latest` qu'elles lisent aux matchs de la lecture : DuckDB ne pousse sous
// la fenêtre `QUALIFY … OVER (PARTITION BY match_id …)` qu'un filtre CONSTANT sur `match_id`.
//
// POURQUOI UN PARAMÈTRE LISTE, ET PAS `IN (?, ?, …)`. Mesuré sur la copie compactée (2 threads,
// 512 Mo, kill-feed d'un joueur) : un `IN` de 7 190 paramètres (Nuzzles) coûte plus que la
// fenêtre entière (304-341 ms contre 195-214 sans liste) ; la MÊME liste en un seul paramètre
// `VARCHAR[]` descend aussi sous la fenêtre (272 132 lignes vues au lieu de 412 216) et coûte
// 192-249 ms ; à 1 160 matchs (JGtm), 64-87 ms contre 98-107 en `IN`. Le coût d'un `IN` long
// est la liaison de ses milliers de paramètres, pas le filtre.
//
// UN APPEL PAR VUE : un filtre posé sur une vue ne traverse pas la jointure vers une autre vue
// `_latest` (ADR 0036 I2). Chaque vue lue reçoit son propre prédicat, avec la même liste.

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/analysis"
)

// LE TEXTE DU PRÉDICAT VIT DANS `analysis` (sql_liste.go, source unique partagée avec les gabarits
// de l'annuaire des noms, item B.7) ; ce fichier n'y ajoute que l'argument. Deux formes : la
// constante pour `match_id` sous une fenêtre (clauseListeMatchs), la semi-jointure partout
// ailleurs (clauseListeParJointure) — cf. l'en-tête de sql_liste.go.

// clauseListeMatchs rend le prédicat CONSTANT qui borne la colonne `col` (un `match_id` sous une
// fenêtre `_latest`) à la liste `ids`, et son unique argument : la liste, liée comme `VARCHAR[]`.
// Une liste vide ne retient aucun match (le prédicat vaut faux, jamais « tous ») ; les lecteurs
// rendent la main avant de toute façon.
func clauseListeMatchs(col string, ids []string) (string, any) {
	return analysis.SQLDansListe(col), argListe(ids)
}

// clauseListeParJointure : la même liste en SEMI-JOINTURE (listes de xuids, filtres sur des
// tables ; jamais sous une fenêtre `_latest`).
func clauseListeParJointure(col string, ids []string) (string, any) {
	return analysis.SQLDansListeParJointure(col), argListe(ids)
}

// argListe rend l'argument d'une liste liée : la liste elle-même, jamais nil (un `nil` se lierait
// comme NULL, et le prédicat vaudrait NULL au lieu de faux).
func argListe(ids []string) any {
	if ids == nil {
		return []string{}
	}
	return ids
}

// QMatchsOuJoue : TOUS les matchs où le joueur figure parmi les participants, Campagne
// comprise. C'est une liste de BORNAGE, pas un agrégat d'affichage : elle borne sous la
// fenêtre `_latest` une lecture qui, sans elle, lisait la table entière. Elle garde donc
// exactement la population de cette lecture, Campagne comprise ; la règle Campagne reste
// celle de la lecture bornée.
const QMatchsOuJoue = `SELECT DISTINCT match_id FROM match_participants WHERE xuid = ?`

// matchsOuJoue rend les matchs de [QMatchsOuJoue] pour `xuid` (liste vide si aucun).
func matchsOuJoue(ctx context.Context, db *sql.DB, xuid string) ([]string, error) {
	rows, err := db.QueryContext(ctx, QMatchsOuJoue, xuid)
	if err != nil {
		return nil, fmt.Errorf("matchs où le joueur a joué: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("matchs où le joueur a joué, scan: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
