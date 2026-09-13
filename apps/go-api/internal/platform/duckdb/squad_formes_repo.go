// Package duckdb — squad_formes_repo.go : LES DEUX LECTURES PROPRES au bloc
// « formes retenues » de la page Escouade (artefact 2ec1b8eb, 2026-09-13).
//
// Le bloc réutilise les trois lectures du résumé d'usage (films, joueurs,
// participants — session_usage_repo.go) et n'ajoute que ce qu'aucune page ne
// lisait encore :
//
//   - LE GRAIN MATCH DES SOCLES : `pad_named` et `weapon_pads_json` vivaient en
//     table sans lecteur (le commentaire de LoadUsageFilms le disait). Ce sont
//     les deux dénominateurs du bloc « contrôle des armes spéciales » : les
//     occupations d'un socle, et les prises qui portent un nom.
//   - LES COLONNES D'OBJECTIF PAR JOUEUR : l'agrégat de session ne lit que les
//     sommes PAR RÔLE ; l'artefact demande en plus les grandeurs réelles du
//     mode (« Drapeaux saisis », « Temps en zone »), les deux camps.
//
// VUES `_latest` UNIQUEMENT (ADR 0026), scope fermé de match_id (aucun filtre
// temporel), lecture seule.
package duckdb

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/analysis/squadformes"
)

// LoadUsageFilmPads retourne, par match_id, le grain match des socles. Un match
// sans ligne film est absent de la map : il n'est pas mesuré.
func (r *SessionUsageRepo) LoadUsageFilmPads(
	ctx context.Context, matchIDs []string,
) (map[string]squadformes.FilmPads, error) {
	out := make(map[string]squadformes.FilmPads, len(matchIDs))
	if len(matchIDs) == 0 {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, sessionUsageQueryTimeout)
	defer cancel()
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("SessionUsageRepo: shared reader: %w", err)
	}
	defer release()

	q := `SELECT match_id, pad_named, pad_unnamed, weapon_pads_json
	      FROM match_usage_films_latest
	      WHERE match_id IN (` + Placeholders(len(matchIDs)) + `)`
	rows, err := db.QueryContext(ctx, q, ToAnySlice(matchIDs)...)
	if err != nil {
		return nil, fmt.Errorf("SessionUsageRepo: film pads query: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var f squadformes.FilmPads
		var pads string
		if err := rows.Scan(&f.MatchID, &f.PadNamed, &f.PadUnnamed, &pads); err != nil {
			return nil, fmt.Errorf("SessionUsageRepo: film pads scan: %w", err)
		}
		if f.WeaponPads, err = weaponPadsFromJSON(pads); err != nil {
			return nil, fmt.Errorf("SessionUsageRepo: %s weapon_pads_json: %w", f.MatchID, err)
		}
		out[f.MatchID] = f
	}
	return out, rows.Err()
}

// weaponPadsFromJSON décode la liste des socles d'arme d'un match. "" et "[]"
// rendent nil : un match sans socle d'arme n'est pas une erreur.
func weaponPadsFromJSON(raw string) ([]squadformes.WeaponPad, error) {
	if raw == "" || raw == "[]" {
		return nil, nil
	}
	var pads []struct {
		Weapon      string `json:"weapon"`
		Occupations int    `json:"occupations"`
		Named       int    `json:"named"`
	}
	if err := json.Unmarshal([]byte(raw), &pads); err != nil {
		return nil, err
	}
	out := make([]squadformes.WeaponPad, 0, len(pads))
	for _, p := range pads {
		out = append(out, squadformes.WeaponPad{
			Weapon: p.Weapon, Occupations: p.Occupations, Named: p.Named,
		})
	}
	return out, nil
}

// objectiveColumnSelect — les colonnes lues, dans un ordre DÉTERMINISTE (le scan
// en dépend) : l'union des colonnes d'action classées par rôle et des colonnes
// de durée, dérivée des tables uniques de narrative. Aucune liste locale — la
// même doctrine que objectiveIndexSelectColumns.
func objectiveColumnSelect() []string {
	seen := map[string]bool{}
	var out []string
	for _, role := range narrative.AllObjectiveRoles() {
		cols := narrative.ObjectiveRoleColumns(role)
		sort.Strings(cols)
		for _, c := range cols {
			if !seen[c] {
				seen[c] = true
				out = append(out, c)
			}
		}
	}
	return out
}

// LoadObjectiveColumnRows retourne, sur un scope fermé de matchs, les lignes
// (match, joueur, famille) avec la VALEUR de chaque colonne d'objectif classée —
// les deux camps (aucun filtre xuid : les parts de camp se calculent en aval).
//
// Best-effort : une erreur de requête (vue absente sur une base non migrée)
// dégrade en nil + warn, comme LoadObjectiveRoleRows. Le bloc perd alors ses
// cartes d'objectif, jamais la page entière.
func (r *ObjectiveStatsRepo) LoadObjectiveColumnRows(
	ctx context.Context, matchIDs []string,
) ([]squadformes.ObjectiveColumnRow, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("ObjectiveStatsRepo: shared reader: %w", err)
	}
	defer release()

	cols := objectiveColumnSelect()
	sel := make([]string, 0, len(cols))
	for _, c := range cols {
		sel = append(sel, "COALESCE(o."+c+", 0)::DOUBLE")
	}
	q := "SELECT o.match_id, o.xuid, " + objectiveFamilyCaseSQL() + " AS family, " +
		strings.Join(sel, ", ") + `
		FROM match_objective_stats_latest o
		WHERE o.match_id IN (` + Placeholders(len(matchIDs)) + `)`
	rows, err := db.QueryContext(ctx, q, ToAnySlice(matchIDs)...)
	if err != nil {
		return nil, fmt.Errorf("ObjectiveStatsRepo: objective columns query: %w", err)
	}
	defer rows.Close()

	var out []squadformes.ObjectiveColumnRow
	for rows.Next() {
		var matchID, xuid string
		var family *string
		values := make([]float64, len(cols))
		dst := make([]any, 0, 3+len(cols))
		dst = append(dst, &matchID, &xuid, &family)
		for i := range values {
			dst = append(dst, &values[i])
		}
		if err := rows.Scan(dst...); err != nil {
			return nil, fmt.Errorf("ObjectiveStatsRepo: objective columns scan: %w", err)
		}
		// Famille NULL : la ligne n'appartient à aucune famille connue (mode sans
		// objectif, ou colonnes toutes nulles). Elle est ÉCARTÉE plutôt que rangée
		// dans une famille par défaut — une grille du mauvais mode serait pire
		// qu'une carte absente.
		if family == nil || *family == "" {
			continue
		}
		row := squadformes.ObjectiveColumnRow{
			MatchID: matchID, XUID: xuid,
			Family: narrative.ObjectiveFamily(*family),
			Values: make(map[string]float64, len(cols)),
		}
		for i, c := range cols {
			row.Values[c] = values[i]
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// LoadFlagGrabsNet retourne, sur un scope fermé de matchs, LES PRISES NETTES DE
// DRAPEAU par match et par joueur — les deux camps, comme les colonnes ci-dessus.
//
// LECTURE SÉPARÉE, PAS UN JOIN, ET C'EST DÉLIBÉRÉ. La grandeur vit dans une
// autre table (`match_flag_grabs_net`, alimentée par le film) que
// `match_objective_stats` (alimentée par l'API). Un LEFT JOIN aurait fait
// tomber TOUTES les colonnes d'objectif le jour où la vue `_latest` du film
// manque — sur une base non migrée, ou sur un titre qui ne produit pas la
// grandeur. Deux lectures indépendantes dégradent indépendamment.
//
// UN MATCH ABSENT DE LA MAP N'EST PAS UN MATCH À ZÉRO : son film n'a pas été lu,
// et la grandeur s'affiche « non mesurée ». C'est le sens de l'absence de clé.
//
// Best-effort : une erreur de requête (vue absente) dégrade en nil + warn.
func (r *ObjectiveStatsRepo) LoadFlagGrabsNet(
	ctx context.Context, matchIDs []string,
) ([]sessionusage.FlagGrabsNetRow, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("ObjectiveStatsRepo: shared reader: %w", err)
	}
	defer release()

	q := `SELECT match_id, xuid, flag_grabs_raw, flag_grabs_net, juggle_window_ms
		FROM match_flag_grabs_net_latest
		WHERE match_id IN (` + Placeholders(len(matchIDs)) + `)`
	rows, err := db.QueryContext(ctx, q, ToAnySlice(matchIDs)...)
	if err != nil {
		slog.WarnContext(ctx, "ObjectiveStatsRepo: prises nettes illisibles (best-effort)",
			"match_count", len(matchIDs), "err", err)
		return nil, nil
	}
	defer rows.Close()

	var out []sessionusage.FlagGrabsNetRow
	for rows.Next() {
		var row sessionusage.FlagGrabsNetRow
		if err := rows.Scan(&row.MatchID, &row.XUID, &row.Raw, &row.Net, &row.WindowMS); err != nil {
			return nil, fmt.Errorf("ObjectiveStatsRepo: prises nettes (scan): %w", err)
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
