// Package duckdb — explorer_repo_resolve.go : résolution d'un gamertag en xuid pour
// l'Explorer (onglet « Joueur »), par les niveaux de la cascade des noms.
//
// Fichier séparé d'explorer_repo.go (au-delà du seuil de 500 lignes, dette gelée par la
// baseline lint : on ne l'accroît pas).
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"levelup/go-api/internal/analysis"
)

// ResolveXUIDByGamertag résout un gamertag en xuid : le xuid dont le nom canonique
// (v_gamertag_lookup) correspond au motif ILIKE, bots exclus.
//
// Le nom canonique d'un xuid est le premier niveau non vide de la cascade de la vue : alias
// (xuid_aliases), puis MAX du gamertag de participant, puis kill-feed, puis libellé masqué.
// Les deux premiers niveaux se lisent sur leurs tables, sans évaluer la vue (ADR 0036 I1,
// 0,5 à 2,3 s mesurés) ; la vue n'est lue qu'en dernier recours, quand aucun alias ni
// participant ne porte ce nom (joueur connu du seul kill-feed, ou libellé masqué).
func (r *ExplorerRepo) ResolveXUIDByGamertag(ctx context.Context, gamertag string) (string, error) {
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return "", fmt.Errorf("ExplorerRepo.ResolveXUIDByGamertag(%q): %w", gamertag, err)
	}
	defer release()

	for _, q := range []string{qResolveParAlias, qResolveParParticipant, qResolveParVueDesNoms} {
		var xuid string
		err := db.QueryRowContext(ctx, q, gamertag).Scan(&xuid)
		if err == nil {
			return xuid, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("ExplorerRepo.ResolveXUIDByGamertag(%q): %w", gamertag, err)
		}
	}
	return "", fmt.Errorf("ExplorerRepo.ResolveXUIDByGamertag(%q): %w", gamertag, sql.ErrNoRows)
}

// qResolveParAlias : niveau 1 de la cascade — un alias non vide gagne toujours.
var qResolveParAlias = `
		SELECT xuid FROM xuid_aliases
		WHERE gamertag IS NOT NULL AND gamertag != '' AND gamertag ILIKE ?
		  AND ` + analysis.SQLIsNotBotCol("xuid") + `
		LIMIT 1
	`

// qResolveParParticipant : niveau 2 — xuid sans alias non vide, dont le MAX des gamertags de
// participant (le nom que la vue retient) correspond. Les candidats sont les xuids dont UNE
// ligne correspond ; le MAX est relu sur toutes leurs lignes.
var qResolveParParticipant = `
		WITH motif AS (SELECT CAST(? AS VARCHAR) AS m),
		candidats AS (
			SELECT DISTINCT mp.xuid FROM match_participants mp, motif
			WHERE mp.gamertag ILIKE motif.m
		),
		noms AS (
			SELECT mp.xuid, MAX(mp.gamertag) AS gamertag
			FROM match_participants mp
			WHERE mp.xuid IN (SELECT xuid FROM candidats)
			GROUP BY mp.xuid
		)
		SELECT n.xuid FROM noms n, motif
		WHERE n.gamertag != '' AND n.gamertag ILIKE motif.m
		  AND ` + analysis.SQLIsNotBotCol("n.xuid") + `
		  AND NOT EXISTS (
			SELECT 1 FROM xuid_aliases xa
			WHERE xa.xuid = n.xuid AND xa.gamertag IS NOT NULL AND xa.gamertag != ''
		  )
		LIMIT 1
	`

// qResolveParVueDesNoms : derniers niveaux (kill-feed, libellé masqué) — la vue elle-même.
var qResolveParVueDesNoms = `
		SELECT xuid FROM v_gamertag_lookup
		WHERE gamertag ILIKE ? AND ` + analysis.SQLIsNotBotCol("xuid") + `
		LIMIT 1
	`
