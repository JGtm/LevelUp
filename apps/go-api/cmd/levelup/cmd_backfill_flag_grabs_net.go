package main

// cmd_backfill_flag_grabs_net.go — sous-commande `levelup backfill-flag-grabs-net`.
//
// ELLE REMPLIT `shared.match_flag_grabs_net` POUR LES ARTEFACTS DEJA CUITS. Le fil de l eau ne
// couvre que les artefacts construits par les cycles a venir (etape post-sync, cf.
// internal/sync/replayartifacts/flaggrabsnet.go) : le corpus existant passe par cette commande.
//
// # ELLE NE DECODE AUCUN FILM, ET ELLE N EN RE-CUIT AUCUN
//
// La source est l ARTEFACT de rejeu deja range (`data/cache/replays/{slug}/{short8}.json`),
// dont le calque `flagCarries` porte la chronologie de portage depuis le SCHEMA 14. Aucune
// recuisson n est necessaire — et aucune n est declenchee : cuire des artefacts en lot est
// INTERDIT (bombe RAM, verrou `filmproc.AcquireSolo`). Cette passe ne fait que LIRE et ECRIRE,
// un artefact a la fois ; rien ne survit d un match a l autre que les compteurs du bilan.
//
// C EST LA DIFFERENCE AVEC `backfill-bomb-stats`, qui exige d avoir re-cuit le parc avant
// (les statistiques d Assaut naissent A LA CUISSON, au schema 39). Ici la mesure se fait A LA
// LECTURE : tout artefact de schema >= 14 est exploitable tel quel.
//
// # DEUX PORTES, ET L UNE N EST PAS UNE CAPABILITY
//
//	`film.flag_grabs_net`                    le titre sait lire ce calque ;
//	`[flag_grabs_net].flag_juggle_window_s`  le titre declare sa regle de jonglage.
//
// L absence de l une ou l autre fait une passe VIDE qui le DIT — jamais des zeros.
//
// # ELLE EST REPRENABLE, ET LA CLE EST LA PRESENCE EN BASE
//
// Un match dont la vue `match_flag_grabs_net_latest` porte deja au moins une ligne est saute,
// sauf `--force`. ⚠ UN CHANGEMENT DE FENETRE EXIGE `--force` : les lignes deja ecrites portent
// l ANCIENNE fenetre dans `juggle_window_ms`, et la reprise ne les reverrait jamais. Le bilan
// imprime la fenetre appliquee pour que ce point se verifie au lieu de se supposer.
//
// Usage (SERVEUR ARRETE — `OpenReadWrite` echoue si le lock est tenu ; meme precondition que
// backfill-bomb-stats, y compris pour --dry-run qui joue les migrations) :
//
//	levelup backfill-flag-grabs-net --dry-run        # compteurs par match, aucune ecriture
//	levelup backfill-flag-grabs-net --match <uuid>   # un seul match
//	levelup backfill-flag-grabs-net                  # tout le corpus, reprenable
//	levelup backfill-flag-grabs-net --force          # re-ecrit meme ce qui est deja en base

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/sync/replayartifacts"
)

// flagGrabsNetOptions : les reglages de la passe.
type flagGrabsNetOptions struct {
	titleSlug string
	limit     int
	force     bool
	dryRun    bool
	match     string
}

// bilanFlagGrabsNetBackfill : ce que la passe a fait — et pourquoi elle a saute ce qu elle a
// saute.
//
// `sansCalque` N EST PAS UN ECHEC : c est un film qui n est pas du CTF (le cas MAJORITAIRE —
// 63 artefacts sur 76 au releve du 2026-09-13), ou un artefact dont le verdict `flagFilm` est
// faux. Le distinguer de `sansArtefact` et d `echecs` est ce qui permet de lire une sortie de
// passe sans se demander si quelque chose s est mal passe.
type bilanFlagGrabsNetBackfill struct {
	ecrits, dejaEnBase, sansArtefact, sansCalque, echecs int
	totalJoueurs, totalBrutes, totalNettes               int
}

func runBackfillFlagGrabsNet(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("backfill-flag-grabs-net", flag.ExitOnError)
	o := flagGrabsNetOptions{}
	fs.StringVar(&o.titleSlug, "title", titlePkg.DefaultSlug, "slug du titre")
	fs.IntVar(&o.limit, "limit", 0, "borne le nombre de matchs projetes (0 = tous)")
	fs.BoolVar(&o.force, "force", false, "re-ecrire meme les matchs deja presents en base (OBLIGATOIRE apres un changement de fenetre)")
	fs.BoolVar(&o.dryRun, "dry-run", false, "projeter et imprimer les compteurs sans rien ecrire")
	fs.StringVar(&o.match, "match", "", "ne traiter que ce match (identifiant complet du registre)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	window, err := porteFlagGrabsNet(cfg, o.titleSlug)
	if err != nil || window <= 0 {
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

	candidats, err := candidatsFlagGrabsNet(ctx, db, o)
	if err != nil {
		return err
	}
	dejaEcrits := map[string]bool{}
	if !o.force {
		if dejaEcrits, err = matchsDejaProjetesFlagGrabsNet(ctx, db); err != nil {
			return err
		}
	}
	debut := time.Now()
	b := projeterCorpusFlagGrabsNet(ctx, db, pr, o, window, candidats, dejaEcrits)
	fmt.Printf("prises de drapeau (fenetre %v) : %d ecrits, %d deja en base, %d sans artefact, "+
		"%d sans calque de drapeau, %d echecs — %s\n",
		window, b.ecrits, b.dejaEnBase, b.sansArtefact, b.sansCalque, b.echecs,
		time.Since(debut).Round(time.Second))
	if b.totalBrutes > 0 {
		fmt.Printf("totaux du corpus projete : %d lignes joueur, %d prises brutes, %d nettes (%.1f %% repliees)\n",
			b.totalJoueurs, b.totalBrutes, b.totalNettes,
			100*float64(b.totalBrutes-b.totalNettes)/float64(b.totalBrutes))
	}
	return nil
}

// porteFlagGrabsNet franchit les DEUX portes du titre. Rend (0, nil) quand le titre ne declare
// pas la grandeur : ce n est pas une erreur, c est une passe vide qui le dit a l ecran.
func porteFlagGrabsNet(cfg *config.AppConfig, titleSlug string) (time.Duration, error) {
	// LE GATE EST UNE CAPABILITY, JAMAIS UN SLUG (ratchet no_slug_comparison_test.go).
	caps, err := capabilitesDuTitre(cfg, titleSlug)
	if err != nil {
		return 0, err
	}
	if !caps.Has(games.CapFilmFlagGrabsNet) {
		fmt.Printf("le titre %s ne declare pas %s : rien a projeter (passe vide)\n",
			titleSlug, string(games.CapFilmFlagGrabsNet))
		return 0, nil
	}
	window, ok, err := replayartifacts.FenetreJonglage(cfg.RepoRoot, titleSlug)
	if err != nil {
		return 0, fmt.Errorf("regulation.toml du titre %s: %w", titleSlug, err)
	}
	if !ok {
		fmt.Printf("le titre %s ne declare pas [flag_grabs_net].flag_juggle_window_s : "+
			"aucune regle de jonglage, rien a projeter (passe vide)\n", titleSlug)
		return 0, nil
	}
	return window, nil
}

// candidatsFlagGrabsNet : les matchs a examiner, dans un ordre stable — `--match` seul, sinon
// tout le registre (c est l artefact qui filtre ensuite : pas d artefact, pas de projection).
func candidatsFlagGrabsNet(ctx context.Context, db *sql.DB, o flagGrabsNetOptions) ([]string, error) {
	if o.match != "" {
		return []string{o.match}, nil
	}
	return matchsDuRegistre(ctx, db, 0)
}

// matchsDejaProjetesFlagGrabsNet : la cle de reprise, lue sur la VUE `_latest` (ADR 0026 —
// jamais la table brute, qui servirait les lignes d une passe deja supplantee).
func matchsDejaProjetesFlagGrabsNet(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT DISTINCT match_id FROM match_flag_grabs_net_latest`)
	if err != nil {
		return nil, fmt.Errorf("matchs deja projetes (prises nettes): %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("matchs deja projetes (prises nettes, scan): %w", err)
		}
		out[id] = true
	}
	return out, rows.Err()
}

// Etats d un match examine par la passe.
type etatFlagGrabsNetMatch int

const (
	prisesAProjeter etatFlagGrabsNetMatch = iota
	prisesSansArtefact
	prisesSansCalque
	prisesEchec
)

// projeterCorpusFlagGrabsNet lit (et ecrit, hors --dry-run) les matchs du lot, UN ARTEFACT A LA
// FOIS.
func projeterCorpusFlagGrabsNet(
	ctx context.Context, db *sql.DB, pr *titlePkg.PathResolver, o flagGrabsNetOptions,
	window time.Duration, candidats []string, dejaEcrits map[string]bool,
) bilanFlagGrabsNetBackfill {
	b := bilanFlagGrabsNetBackfill{}
	p := persist.NewFlagGrabsNetPersister(db)
	projetes := 0
	for _, id := range candidats {
		if o.limit > 0 && projetes >= o.limit {
			break
		}
		if !o.force && dejaEcrits[id] {
			b.dejaEnBase++
			continue
		}
		batch, etat := lireUnArtefactPrisesNettes(pr.ReplayArtifactPath(o.titleSlug, id), id, window)
		switch etat {
		case prisesSansArtefact:
			b.sansArtefact++
			continue
		case prisesSansCalque:
			b.sansCalque++
			continue
		case prisesEchec:
			b.echecs++
			continue
		}
		projetes++
		comptabiliserPasse(&b, batch)
		if o.dryRun {
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

// comptabiliserPasse additionne les totaux du bilan, et imprime la ligne du --dry-run.
func comptabiliserPasse(b *bilanFlagGrabsNetBackfill, batch persist.FlagGrabsNetBatch) {
	brut, net := 0, 0
	for _, pl := range batch.Players {
		brut += pl.Raw
		net += pl.Net
	}
	b.totalJoueurs += len(batch.Players)
	b.totalBrutes += brut
	b.totalNettes += net
}

// lireUnArtefactPrisesNettes lit UN artefact et en tire la passe a ecrire, ou dit pourquoi il
// ne le fait pas. L artefact entier est relache a la sortie — seule la passe (quelques
// dizaines d octets) survit.
//
// LA PROJECTION EST CELLE DU FIL DE L EAU, PAS UNE SECONDE : `replayartifacts.ProjeterPrisesNettes`
// est exportee exactement pour ca. Deux projections de la meme regle divergeraient au premier
// changement — c est la dette que la regle du depot (« <= 2 copies d un meme motif ») interdit.
func lireUnArtefactPrisesNettes(path, matchID string, window time.Duration) (persist.FlagGrabsNetBatch, etatFlagGrabsNetMatch) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return persist.FlagGrabsNetBatch{}, prisesSansArtefact
		}
		fmt.Printf("  ECHEC %s : lecture artefact: %v\n", matchID, err)
		return persist.FlagGrabsNetBatch{}, prisesEchec
	}
	var doc replay.ReplayDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		fmt.Printf("  ECHEC %s : parse artefact: %v\n", matchID, err)
		return persist.FlagGrabsNetBatch{}, prisesEchec
	}
	batch := replayartifacts.ProjeterPrisesNettes(matchID, &doc, window)
	if batch.MatchID == "" {
		return persist.FlagGrabsNetBatch{}, prisesSansCalque
	}
	return batch, prisesAProjeter
}
