package main

// cmd_backfill_replay_nongeres.go — LES MATCHS DE MODE NON GERE dans `backfill-replay`.
//
// Le rejeu ne gere pas le Firefight (PvE, decision du 2026-10-15). Le registre porte le drapeau
// (`analysis.SQLModeNonGere`) ; ce fichier en tire les deux gestes de la commande :
//
//   - `--one <id>` sur un tel match REFUSE, avec un message clair (la forme interne appelee par le
//     parent n'y arrive jamais : la selection les a ecartes) ;
//   - `--purge-non-geres [--dry-run]` retire les artefacts DEJA cuits (artefact, marque de
//     derivation, raster), sans jamais toucher au film (`film_chunks/`, archive irremplacable) ni
//     aux faits persistes.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/analysis"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/replaybuild"
)

// matchNonGere dit si le registre classe CE match (identifiant complet) dans un mode non gere.
// `connu` est faux quand le match n'est pas au registre : on ne refuse rien de ce qu'on ne sait
// pas classer.
func matchNonGere(ctx context.Context, db *sql.DB, matchID string) (nonGere, connu bool, err error) {
	row := db.QueryRowContext(ctx,
		`SELECT `+analysis.SQLModeNonGere("")+` FROM match_registry WHERE match_id = ?`, matchID)
	if err := row.Scan(&nonGere); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, false, nil
		}
		return false, false, fmt.Errorf("registre des matchs (mode): %w", err)
	}
	return nonGere, true, nil
}

// idsNonGeres liste les matchs du registre dont le mode n'est pas gere.
func idsNonGeres(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT match_id FROM match_registry WHERE `+analysis.SQLModeNonGere("")+` ORDER BY match_id`)
	if err != nil {
		return nil, fmt.Errorf("registre des matchs (modes non geres): %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("registre des matchs (modes non geres, scan): %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// refuserUnMatchNonGere est la garde de `--one` : erreur [replaybuild.ErrModeNonGere] quand le
// registre classe le match dans un mode non gere. Registre illisible = journalise puis on
// laisse passer (le parent a deja ecarte ces matchs ; la garde ne doit pas inventer une panne).
func refuserUnMatchNonGere(ctx context.Context, pr *titlePkg.PathResolver, titleSlug, matchID string) error {
	db, release, err := duckdb.OpenReadForQuery(pr.SharedDBPath(titleSlug))
	if err != nil {
		slog.WarnContext(ctx, "backfill-replay --one: registre indisponible — mode du match non verifie",
			"err", err, "match_id", matchID)
		return nil
	}
	defer release()
	return garderUnMatchNonGere(ctx, db, matchID)
}

// garderUnMatchNonGere : la decision de [refuserUnMatchNonGere], sur une base deja ouverte.
func garderUnMatchNonGere(ctx context.Context, db *sql.DB, matchID string) error {
	nonGere, _, err := matchNonGere(ctx, db, matchID)
	if err != nil {
		slog.WarnContext(ctx, "backfill-replay --one: mode du match illisible", "err", err, "match_id", matchID)
		return nil
	}
	if nonGere {
		return fmt.Errorf("--one %s : mode non gere par le rejeu (Firefight), rien n'est cuit : %w",
			matchID, replaybuild.ErrModeNonGere)
	}
	return nil
}

// purgerNonGeres retire les artefacts deja cuits des matchs de mode non gere. Rend une erreur
// seulement si le registre est illisible ou qu'une suppression echoue ; le bilan sort sur la
// sortie standard.
func purgerNonGeres(ctx context.Context, pr *titlePkg.PathResolver, o replayBackfillOptions) error {
	db, release, err := duckdb.OpenReadForQuery(pr.SharedDBPath(o.titleSlug))
	if err != nil {
		return fmt.Errorf("open shared RO (%s): %w", pr.SharedDBPath(o.titleSlug), err)
	}
	ids, err := idsNonGeres(ctx, db)
	release() // registre relache AVANT toute suppression de fichier
	if err != nil {
		return err
	}
	return purgerLesArtefacts(pr, o, ids)
}

// purgerLesArtefacts est la boucle de [purgerNonGeres], sans base.
func purgerLesArtefacts(pr *titlePkg.PathResolver, o replayBackfillOptions, ids []string) error {
	verbe, bilan := "supprime", "supprimes"
	if o.dryRun {
		verbe, bilan = "a supprimer (dry-run)", "a supprimer (dry-run)"
	}
	matchs, fichiers := 0, 0
	for _, id := range ids {
		touches, err := replaybuild.PurgeMatchFiles(pr, o.titleSlug, id, o.dryRun)
		for _, p := range touches {
			fmt.Printf("  %s : %s\n", verbe, p)
		}
		if err != nil {
			return err
		}
		if len(touches) > 0 {
			matchs++
			fichiers += len(touches)
		}
	}
	fmt.Printf("matchs de mode non gere au registre : %d ; avec rejeu sur disque : %d ; fichiers %s : %d\n",
		len(ids), matchs, bilan, fichiers)
	return nil
}

// purgerNonGeresSeul refuse `--purge-non-geres` combine a une forme qui cuit, puis purge.
func purgerNonGeresSeul(ctx context.Context, pr *titlePkg.PathResolver, o replayBackfillOptions) error {
	if o.one != "" || o.force || o.onlyExisting || o.repairImpoverished {
		return fmt.Errorf("--purge-non-geres s'emploie seul (avec --dry-run, --title, --cache)")
	}
	return purgerNonGeres(ctx, pr, o)
}
