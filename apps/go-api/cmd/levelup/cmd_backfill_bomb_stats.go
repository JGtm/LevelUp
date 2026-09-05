package main

// cmd_backfill_bomb_stats.go — sous-commande `levelup backfill-bomb-stats`.
//
// ELLE REMPLIT `shared.match_bomb_stats` (+ les faits dates de `match_objective_events`) POUR
// LES ARTEFACTS DEJA CUITS. Le fil de l eau ne couvre que les artefacts construits par les
// cycles a venir (etape post-sync, cf. internal/sync/replayartifacts/bombstats.go) : le corpus
// existant passe par cette commande.
//
// # ELLE NE DECODE AUCUN FILM, ET C EST CE QUI LA REND UTILISABLE
//
// La source est l ARTEFACT de rejeu (`data/cache/replays/{slug}/{short8}.json`), dont le champ
// `bombStats` porte DEJA les cinq statistiques : elles sont calculees A LA CUISSON, le seul
// endroit ou leurs quatre sources vivent en pleine fidelite (chronologie de portage en
// millisecondes, recalage film -> match, armements dates, actions d objectif nommees). Cette
// passe ne fait que LIRE et ECRIRE, un artefact a la fois — jamais de map globale vivante
// (lecon du chantier backfill-replay, qui a sature la machine le 2026-08-20).
//
// # L ORDRE DE LA RELEASE, ET IL N EST PAS INTERCHANGEABLE
//
//	1. `levelup backfill-replay`       RE-CUIT le parc — le schema 39 perime tout artefact
//	                                   anterieur, et c est cette passe-la qui fait NAITRE
//	                                   `bombStats` dans les artefacts ;
//	2. `levelup backfill-bomb-stats`   PROJETTE les artefacts ainsi cuits vers la base.
//
// Lancer (2) sans (1) ne trouve aucun `bombStats` sur les artefacts de schema < 39 et n ecrit
// rien : ce n est pas une erreur, c est un no-op qui le DIT (compteur `sans calque`). AUCUNE des
// deux n est lancee par un agent — ce sont des taches de RELEASE.
//
// # ELLE EST REPRENABLE, ET LA CLE EST LA PRESENCE EN BASE
//
// Un match dont la vue `match_bomb_stats_latest` porte deja au moins une ligne est saute, sauf
// `--force`. Interrompre puis relancer reprend donc ou elle en etait. Contrairement au resume
// d usage, il n y a pas de « revision de projection » a comparer : ces statistiques sont le
// CONTENU de l artefact, pas une derivation qui pourrait changer de regle sans changer de
// schema — un artefact re-cuit se re-projette par `--force`, et le journal le dit.
//
// Usage (SERVEUR ARRETE — `OpenReadWrite` echoue si le lock est tenu ; meme precondition que
// backfill-usage-summary et backfill-killsource, y compris pour --dry-run qui joue les
// migrations) :
//
//	levelup backfill-bomb-stats --dry-run          # compteurs par match, aucune ecriture
//	levelup backfill-bomb-stats --match <uuid>     # un seul match
//	levelup backfill-bomb-stats                    # tout le corpus, reprenable
//	levelup backfill-bomb-stats --force            # re-ecrit meme ce qui est deja en base

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/duckdb"
)

// bombStatsOptions : les reglages de la passe.
type bombStatsOptions struct {
	titleSlug string
	limit     int
	force     bool
	dryRun    bool
	match     string
}

// bilanBombStatsBackfill : ce que la passe a fait — et pourquoi elle a saute ce qu elle a saute.
//
// `sansCalque` N EST PAS UN ECHEC : c est un match hors de la famille bomb, ou un artefact
// anterieur au schema 39. Le distinguer de `sansArtefact` et d `echecs` est ce qui permet de
// lire une sortie de passe sans se demander si quelque chose s est mal passe.
type bilanBombStatsBackfill struct {
	ecrits, dejaEnBase, sansArtefact, sansCalque, echecs int
	totalJoueurs, totalFaits                             int
}

func runBackfillBombStats(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("backfill-bomb-stats", flag.ExitOnError)
	o := bombStatsOptions{}
	fs.StringVar(&o.titleSlug, "title", titlePkg.DefaultSlug, "slug du titre")
	fs.IntVar(&o.limit, "limit", 0, "borne le nombre de matchs projetes (0 = tous)")
	fs.BoolVar(&o.force, "force", false, "re-ecrire meme les matchs deja presents en base")
	fs.BoolVar(&o.dryRun, "dry-run", false, "projeter et imprimer les compteurs sans rien ecrire")
	fs.StringVar(&o.match, "match", "", "ne traiter que ce match (identifiant complet du registre)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx := context.Background()
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	sharedPath := pr.SharedDBPath(o.titleSlug)
	if _, err := os.Stat(sharedPath); err != nil {
		return fmt.Errorf("shared_matches introuvable (%s): %w", sharedPath, err)
	}
	handle, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		return fmt.Errorf("open shared RW (%s): %w (serveur arrete ?)", sharedPath, err)
	}
	defer handle.Close()
	db := handle.SQLDb()

	// La table et sa vue doivent exister AVANT de lire ou d ecrire — cette commande tourne
	// SERVEUR ARRETE, donc rien n a joue les migrations pour elle.
	if err := migrerSchemaPartage(db, o.titleSlug); err != nil {
		return err
	}

	// LE GATE EST UNE CAPABILITY, JAMAIS UN SLUG (ratchet no_slug_comparison_test.go).
	caps, err := capabilitesDuTitre(cfg, o.titleSlug)
	if err != nil {
		return err
	}
	if !caps.Has(games.CapFilmBombStats) {
		fmt.Printf("le titre %s ne declare pas %s : rien a projeter (passe vide)\n",
			o.titleSlug, string(games.CapFilmBombStats))
		return nil
	}

	candidats, err := candidatsBombStats(ctx, db, o)
	if err != nil {
		return err
	}
	dejaEcrits := map[string]bool{}
	if !o.force {
		if dejaEcrits, err = matchsDejaProjetes(ctx, db); err != nil {
			return err
		}
	}
	debut := time.Now()
	b := projeterCorpusBombe(ctx, db, pr, o, candidats, dejaEcrits)
	fmt.Printf("stats d Assaut : %d ecrits, %d deja en base, %d sans artefact, "+
		"%d sans calque de bombe, %d echecs — %s\n",
		b.ecrits, b.dejaEnBase, b.sansArtefact, b.sansCalque, b.echecs,
		time.Since(debut).Round(time.Second))
	if o.dryRun {
		fmt.Printf("totaux du corpus projete : %d lignes joueur, %d faits dates\n",
			b.totalJoueurs, b.totalFaits)
	}
	return nil
}

// candidatsBombStats : les matchs a examiner, dans un ordre stable — `--match` seul, sinon tout
// le registre (c est l artefact qui filtre ensuite : pas d artefact, pas de projection).
func candidatsBombStats(ctx context.Context, db *sql.DB, o bombStatsOptions) ([]string, error) {
	if o.match != "" {
		return []string{o.match}, nil
	}
	return matchsDuRegistre(ctx, db, 0)
}

// matchsDejaProjetes : la cle de reprise, lue sur la VUE `_latest` (ADR 0026 — jamais la table
// brute, qui servirait les lignes d une passe deja supplantee).
func matchsDejaProjetes(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT DISTINCT match_id FROM match_bomb_stats_latest`)
	if err != nil {
		return nil, fmt.Errorf("matchs deja projetes: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("matchs deja projetes (scan): %w", err)
		}
		out[id] = true
	}
	return out, rows.Err()
}

// Etats d un match examine par la passe.
type etatBombeMatch int

const (
	bombeAProjeter etatBombeMatch = iota
	bombeSansArtefact
	bombeSansCalque
	bombeDejaEnBase
	bombeEchec
)

// projeterCorpusBombe lit (et ecrit, hors --dry-run) les matchs du lot, UN ARTEFACT A LA FOIS :
// rien ne survit d un match a l autre que les compteurs du bilan.
func projeterCorpusBombe(
	ctx context.Context, db *sql.DB, pr *titlePkg.PathResolver, o bombStatsOptions,
	candidats []string, dejaEcrits map[string]bool,
) bilanBombStatsBackfill {
	b := bilanBombStatsBackfill{}
	p := persist.NewBombStatsPersister(db)
	projetes := 0
	for _, id := range candidats {
		if o.limit > 0 && projetes >= o.limit {
			break
		}
		if !o.force && dejaEcrits[id] {
			b.dejaEnBase++
			continue
		}
		batch, etat := lireUnArtefactBombe(pr.ReplayArtifactPath(o.titleSlug, id), id)
		switch etat {
		case bombeSansArtefact:
			b.sansArtefact++
			continue
		case bombeSansCalque:
			b.sansCalque++
			continue
		case bombeEchec:
			b.echecs++
			continue
		}
		projetes++
		b.totalJoueurs += len(batch.Players)
		b.totalFaits += len(batch.Events)
		if o.dryRun {
			fmt.Printf("  %-40s joueurs=%2d faits=%2d\n", id, len(batch.Players), len(batch.Events))
			continue
		}
		if err := p.PersistPass(ctx, batch); err != nil {
			// Jamais avale : l echec d UN match n arrete pas la passe, mais il se voit.
			fmt.Printf("  ECHEC %s : %v\n", id, err)
			b.echecs++
			continue
		}
		b.ecrits++
	}
	return b
}

// lireUnArtefactBombe lit UN artefact et en tire la passe a ecrire, ou dit pourquoi il ne le
// fait pas. L artefact entier est relache a la sortie — seule la passe (quelques centaines
// d octets) survit.
//
// LA CONVERSION EST CELLE DU FIL DE L EAU, PAS UNE SECONDE : `replayartifacts` porte la meme,
// et c est voulu qu elles restent DEUX — le paquet `cmd` ne peut pas importer un identifiant
// non exporte de `sync/replayartifacts`, et exporter le transport pour ce seul appelant
// coderait une dependance de la production sur un outil hors ligne. Les deux sont TENUES par la
// meme validation (`persist.validateBombStatsBatch`) et par le meme vocabulaire
// (`replay.BombEventProvenance`), qui vit en UN seul endroit.
func lireUnArtefactBombe(path, matchID string) (persist.BombStatsBatch, etatBombeMatch) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return persist.BombStatsBatch{}, bombeSansArtefact
		}
		fmt.Printf("  ECHEC %s : lecture artefact: %v\n", matchID, err)
		return persist.BombStatsBatch{}, bombeEchec
	}
	var doc replay.ReplayDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		fmt.Printf("  ECHEC %s : parse artefact: %v\n", matchID, err)
		return persist.BombStatsBatch{}, bombeEchec
	}
	if doc.BombStats == nil {
		return persist.BombStatsBatch{}, bombeSansCalque
	}
	batch := persist.BombStatsBatch{MatchID: matchID}
	for _, pl := range doc.BombStats.Players {
		batch.Players = append(batch.Players, persist.BombPlayerStatsRow{
			XUID: pl.XUID, Detonations: pl.Detonations, Arms: pl.Arms, Grabs: pl.Grabs,
			TimeAsCarrierSeconds: pl.TimeAsCarrierSeconds, CarriersKilled: pl.CarriersKilled,
		})
	}
	for _, ev := range doc.BombEvents {
		src, conf := replay.BombEventProvenance(ev.Type)
		batch.Events = append(batch.Events, persist.BombEventRow{
			EventType: ev.Type, TimeMS: ev.TimeMS, XUID: ev.XUID,
			Source: src, Confidence: conf,
		})
	}
	if len(batch.Players) == 0 && len(batch.Events) == 0 {
		return persist.BombStatsBatch{}, bombeSansCalque
	}
	return batch, bombeAProjeter
}
