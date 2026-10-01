package main

// cmd_backfill_vehicle_takes.go — sous-commande `levelup backfill-vehicle-takes`.
//
// ELLE REMPLIT `shared.match_vehicle_takes` POUR LES ARTEFACTS DEJA CUITS (plan Emprise
// vehicules, lot L7.2). Le fil de l eau ne couvre que les artefacts ranges par les cycles a venir
// (etape post-sync, cf. internal/sync/replayartifacts/vehicletakes.go) : le corpus existant, et
// tout artefact recuit ensuite, passent par cette commande.
//
// # ELLE NE DECODE AUCUN FILM, ET ELLE N EN RE-CUIT AUCUN
//
// La source est l ARTEFACT de rejeu deja range (`data/cache/replays/{slug}/{short8}.json`), dont
// le calque vehicules (`vehicles[].rides[]`) est publie depuis le schema 67. Un artefact plus
// ancien donne une passe « non mesure » (D8) — jamais zero : c est l etat de tout le parc tant
// qu il n est pas recuit, et la commande l ecrit pour que l onglet le dise. Cuire des artefacts
// en lot est INTERDIT (bombe RAM, verrou `filmproc.AcquireSolo`).
//
// # UNE LECTURE EN BASE, ET ELLE EST CELLE DU FIL DE L EAU
//
// Les frags de classe engin se lisent dans `match_kill_events_latest` par
// `replayartifacts.LireFragsDEngin`, la projection est `replayartifacts.ProjeterPrisesDeVehicules` :
// aucune seconde copie de la regle (deux projections de la meme regle divergeraient).
//
// # UNE PORTE : LA CAPABILITY
//
// `film.vehicle_usage`, jamais un slug. Sans elle : passe vide qui le dit a l ecran.
//
// # ELLE EST REPRENABLE, ET LA CLE EST LA PRESENCE EN BASE
//
// Un match dont la vue `match_vehicle_takes_latest` porte deja une ligne est saute, sauf
// `--force`. ⚠ UN ARTEFACT RECUIT (schema >= 67) OU DES EVENEMENTS DE MORT ARRIVES APRES LA
// PREMIERE PASSE EXIGENT `--force` : la passe en base dit « non mesure » ou « frags non lus » et
// la reprise ne la reverrait jamais.
//
// Usage (SERVEUR ARRETE — `OpenReadWrite` echoue si le lock est tenu ; meme precondition que
// backfill-pad-tiers, y compris pour --dry-run qui joue les migrations) :
//
//	levelup backfill-vehicle-takes --dry-run        # une ligne par match, aucune ecriture
//	levelup backfill-vehicle-takes --match a1b2c3d4,<uuid>   # ces matchs (liste, prefixes 8+ univoques)
//	levelup backfill-vehicle-takes                  # tout le corpus, reprenable
//	levelup backfill-vehicle-takes --force          # re-ecrit meme ce qui est deja en base

import (
	"context"
	"database/sql"
	"errors"
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
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync/replayartifacts"
)

// vehicleTakesOptions : les reglages de la passe.
type vehicleTakesOptions struct {
	titleSlug string
	limit     int
	force     bool
	dryRun    bool
	match     string
}

// bilanVehicleTakesBackfill : ce que la passe a fait, et pourquoi elle a saute ce qu elle a saute.
// `nonMesures` N EST PAS UN ECHEC : c est un artefact sans occupation lue (D8), ecrit comme tel.
type bilanVehicleTakesBackfill struct {
	ecrits, dejaEnBase, sansArtefact, echecs int
	nonMesures, sansFrags                    int
	lignes, prises, fragsTotal, fragsNonApp  int
}

func runBackfillVehicleTakes(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("backfill-vehicle-takes", flag.ExitOnError)
	o := vehicleTakesOptions{}
	fs.StringVar(&o.titleSlug, "title", titlePkg.DefaultSlug, "slug du titre")
	fs.IntVar(&o.limit, "limit", 0, "borne le nombre de matchs projetes (0 = tous)")
	fs.BoolVar(&o.force, "force", false,
		"re-ecrire meme les matchs deja presents en base (OBLIGATOIRE apres une recuisson ou l arrivee des frags)")
	fs.BoolVar(&o.dryRun, "dry-run", false, "projeter et imprimer une ligne par match sans rien ecrire")
	fs.StringVar(&o.match, "match", "", "ne traiter que ces matchs : liste separee par des virgules, identifiants complets ou prefixes "+
		"univoques d au moins 8 caracteres, resolus contre le registre (refus sinon, rien d ecrit)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	caps, err := capabilitesDuTitre(cfg, o.titleSlug)
	if err != nil {
		return err
	}
	if !caps.Has(games.CapFilmVehicleUsage) {
		fmt.Printf("le titre %s ne declare pas %s : rien a projeter (passe vide)\n",
			o.titleSlug, string(games.CapFilmVehicleUsage))
		return nil
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

	// La table et sa vue doivent exister AVANT de lire ou d ecrire : cette commande tourne
	// SERVEUR ARRETE, donc rien n a joue les migrations pour elle.
	if err := migrerSchemaPartage(db, o.titleSlug); err != nil {
		return err
	}
	// `--match` se RESOUT contre le registre (liste, prefixes de 8+ caracteres univoques) : un refus
	// part ICI, avant toute ecriture. Jamais l identifiant pris tel quel (RV8).
	candidats, err := registreBorne(ctx, db, o.match)
	if err != nil {
		return err
	}
	dejaEcrits := map[string]bool{}
	if !o.force {
		if dejaEcrits, err = matchsDejaProjetesVehicules(ctx, db); err != nil {
			return err
		}
	}
	debut := time.Now()
	classifier := replayartifacts.ClassifierDEngins(cfg.RepoRoot, o.titleSlug)
	b := projeterCorpusVehicules(ctx, db, pr, o, classifier, candidats, dejaEcrits)
	fmt.Printf("vehicules : %d ecrits (dont %d non mesures, %d sans frags lus), %d deja en base, "+
		"%d sans artefact, %d echecs — %s\n",
		b.ecrits, b.nonMesures, b.sansFrags, b.dejaEnBase, b.sansArtefact, b.echecs,
		time.Since(debut).Round(time.Second))
	if b.lignes > 0 {
		fmt.Printf("totaux du corpus projete : %d lignes, %d prises, %d frags de classe engin dont %d non apparies\n",
			b.lignes, b.prises, b.fragsTotal, b.fragsNonApp)
	}
	return nil
}

// matchsDejaProjetesVehicules : la cle de reprise, lue sur la VUE `_latest` (ADR 0026 — jamais la
// table brute).
func matchsDejaProjetesVehicules(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT DISTINCT match_id FROM match_vehicle_takes_latest`)
	if err != nil {
		return nil, fmt.Errorf("matchs deja projetes (vehicules): %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("matchs deja projetes (vehicules, scan): %w", err)
		}
		out[id] = true
	}
	return out, rows.Err()
}

// projeterCorpusVehicules lit (et ecrit, hors --dry-run) les matchs du lot, UN ARTEFACT A LA FOIS.
func projeterCorpusVehicules(
	ctx context.Context, db *sql.DB, pr *titlePkg.PathResolver, o vehicleTakesOptions,
	classifier port.KillSourceClassifier, candidats []string, dejaEcrits map[string]bool,
) bilanVehicleTakesBackfill {
	b := bilanVehicleTakesBackfill{}
	p := persist.NewVehicleTakesPersister(db)
	projetes := 0
	for _, id := range candidats {
		if o.limit > 0 && projetes >= o.limit {
			break
		}
		if !o.force && dejaEcrits[id] {
			b.dejaEnBase++
			continue
		}
		doc, err := replayartifacts.LireArtefactRange(pr.ReplayArtifactPath(o.titleSlug, id))
		if errors.Is(err, os.ErrNotExist) {
			b.sansArtefact++
			continue
		}
		if err != nil {
			fmt.Printf("  ECHEC %s : %v\n", id, err)
			b.echecs++
			continue
		}
		lecture := replayartifacts.VehicleFragsLecture{Reason: replayartifacts.VehicleFragsNotMeasured}
		if replay.ProjectVehicleTakes(doc).Measured {
			lectures, err := replayartifacts.LireFragsDEngin(ctx, db, classifier, []string{id})
			if err != nil {
				fmt.Printf("  ECHEC %s : %v\n", id, err)
				b.echecs++
				continue
			}
			lecture = lectures[id]
		}
		batch := replayartifacts.ProjeterPrisesDeVehicules(id, doc, lecture)
		projetes++
		comptabiliserPasseVehicules(&b, batch)
		if o.dryRun {
			// UNE LIGNE PAR MATCH : l operateur doit pouvoir CONTROLER ce qui sera ecrit.
			fmt.Println(ligneDryRunVehicules(id, batch))
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

// comptabiliserPasseVehicules additionne les totaux du bilan.
func comptabiliserPasseVehicules(b *bilanVehicleTakesBackfill, batch persist.VehicleTakesBatch) {
	if !batch.Measured {
		b.nonMesures++
		return
	}
	if !batch.FragsRead {
		b.sansFrags++
	}
	b.lignes += len(batch.Rows)
	for _, r := range batch.Rows {
		b.prises += r.Takes
	}
	b.fragsTotal += batch.FragsTotal
	b.fragsNonApp += batch.FragsUnmatched
}

// ligneDryRunVehicules rend la ligne `--dry-run` d UN match. Les DEUX raisons (celle de la
// passe, celle des frags) portent chacune son etiquette et sont separees : collees, elles se
// lisaient `schema_before_67takes_not_measured` (revue L7.5, RV1, 2026-10-01). Une raison
// vide est rendue `-`, pour que la colonne reste lisible.
func ligneDryRunVehicules(id string, batch persist.VehicleTakesBatch) string {
	return fmt.Sprintf("  %-40s mesure=%-5v schema=%3d lignes=%3d frags=%d/%d raison=%s frags_raison=%s",
		id, batch.Measured, batch.DocSchema, len(batch.Rows), batch.FragsTotal-batch.FragsUnmatched,
		batch.FragsTotal, raisonOuTiret(batch.Reason), raisonOuTiret(batch.FragsReason))
}

func raisonOuTiret(r string) string {
	if r == "" {
		return "-"
	}
	return r
}
