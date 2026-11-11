package main

// cmd_backfill_lead_share.go — sous-commande `levelup backfill-lead-share`.
//
// ELLE REMPLIT `shared.match_lead_share` POUR LES ARTEFACTS DEJA CUITS. Le fil de l'eau ne couvre
// que les artefacts ranges par les cycles a venir (etape post-sync, cf.
// internal/sync/replayartifacts/leadshare.go) : le corpus existant, et tout artefact recuit
// ensuite, passent par cette commande.
//
// # ELLE NE DECODE AUCUN FILM, ET ELLE N'EN RE-CUIT AUCUN
//
// La source est l'artefact de rejeu deja range ; sa lecture passe par
// `replayartifacts.LireArtefactRange`, la projection par `replayartifacts.ProjeterMainmise` :
// aucune seconde copie de la regle. Le seul acces a la base est la lecture du mode et de la duree
// du match dans le registre. Sequentielle : un artefact a la fois.
//
// # UNE PORTE : LA CAPABILITY
//
// `film.lead_share`, jamais un slug. Sans elle : passe vide qui le dit.
//
// # --dry-run N'ECRIT RIEN, PAS MEME UNE MIGRATION
//
// La base partagee est ouverte en LECTURE SEULE et aucune migration n'est jouee : la commande
// imprime une ligne par match et le bilan par raison de refus. La table n'a pas besoin d'exister.
//
// # ELLE EST REPRENABLE, ET LA CLE EST LA PRESENCE EN BASE
//
// Un match dont la vue `match_lead_share_latest` porte deja une ligne est saute, sauf `--force`.
// ⚠ UN ARTEFACT RECUIT OU UN CHANGEMENT DE REGLE EXIGENT `--force` : la passe en base dit « non
// mesure » ou une ancienne mesure et la reprise ne la reverrait jamais.
//
// Usage (SERVEUR ARRETE) :
//
//	levelup backfill-lead-share --dry-run --limit 20      # une ligne par match, aucune ecriture
//	levelup backfill-lead-share --match a1b2c3d4,<uuid>   # ces matchs (liste, prefixes 8+ univoques)
//	levelup backfill-lead-share                           # tout le corpus, reprenable
//	levelup backfill-lead-share --force                   # re-ecrit meme ce qui est deja en base

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/sync/replayartifacts"
)

// leadShareOptions : les reglages de la passe.
type leadShareOptions struct {
	titleSlug string
	limit     int
	force     bool
	dryRun    bool
	match     string
}

// bilanLeadShareBackfill : ce que la passe a fait. `nonMesures` N'EST PAS UN ECHEC : c'est un match
// dont la mesure est refusee, ecrit comme tel avec sa raison.
type bilanLeadShareBackfill struct {
	ecrits, mesures, dejaEnBase, sansArtefact, echecs, sansRegistre int
	raisons                                                         map[string]int
}

func (b *bilanLeadShareBackfill) nonMesures() int {
	n := 0
	for _, c := range b.raisons {
		n += c
	}
	return n
}

func runBackfillLeadShare(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("backfill-lead-share", flag.ExitOnError)
	o := leadShareOptions{}
	fs.StringVar(&o.titleSlug, "title", titlePkg.DefaultSlug, "slug du titre")
	fs.IntVar(&o.limit, "limit", 0, "borne le nombre de matchs projetes (0 = tous)")
	fs.BoolVar(&o.force, "force", false,
		"re-ecrire meme les matchs deja presents en base (OBLIGATOIRE apres une recuisson ou un changement de regle)")
	fs.BoolVar(&o.dryRun, "dry-run", false,
		"projeter et imprimer une ligne par match sans rien ecrire (base ouverte en lecture seule, aucune migration)")
	fs.StringVar(&o.match, "match", "", "ne traiter que ces matchs : liste separee par des virgules, identifiants complets ou "+
		"prefixes univoques d'au moins 8 caracteres, resolus contre le registre (refus sinon, rien d'ecrit)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	caps, err := capabilitesDuTitre(cfg, o.titleSlug)
	if err != nil {
		return err
	}
	if !caps.Has(games.CapFilmLeadShare) {
		fmt.Printf("le titre %s ne declare pas %s : rien a projeter (passe vide)\n",
			o.titleSlug, string(games.CapFilmLeadShare))
		return nil
	}
	pr := cfg.PathResolver()
	reg, err := replayartifacts.ReglementDuTitre(pr, o.titleSlug)
	if err != nil {
		return fmt.Errorf("regulation.toml (%s): %w", o.titleSlug, err)
	}
	sharedPath := pr.SharedDBPath(o.titleSlug)
	if _, err := os.Stat(sharedPath); err != nil {
		return fmt.Errorf("shared_matches introuvable (%s): %w", sharedPath, err)
	}
	db, fermer, err := ouvrirSharedMainmise(sharedPath, o.titleSlug, o.dryRun)
	if err != nil {
		return err
	}
	defer fermer()

	ctx := context.Background()
	// `--match` se RESOUT contre le registre : un refus part ICI, avant toute ecriture.
	candidats, err := registreBorne(ctx, db, o.match)
	if err != nil {
		return err
	}
	dejaEcrits := map[string]bool{}
	if !o.force && !o.dryRun {
		if dejaEcrits, err = matchsDejaProjetesMainmise(ctx, db); err != nil {
			return err
		}
	}
	debut := time.Now()
	b := passeMainmise{db: db, pr: pr, o: o, reg: reg}.projeterCorpus(ctx, candidats, dejaEcrits)
	fmt.Print(syntheseMainmise(b, time.Since(debut).Round(time.Millisecond)))
	return nil
}

// ouvrirSharedMainmise ouvre la base partagee : en LECTURE SEULE pour --dry-run (aucune migration,
// rien d'ecrit, la table peut ne pas exister), en ecriture sinon, apres les migrations (la commande
// tourne SERVEUR ARRETE, rien ne les a jouees pour elle).
func ouvrirSharedMainmise(path, slug string, dryRun bool) (*sql.DB, func(), error) {
	if dryRun {
		h, err := duckdb.OpenReadOnly(path)
		if err != nil {
			return nil, nil, fmt.Errorf("open shared RO (%s): %w (serveur arrete ?)", path, err)
		}
		return h.SQLDb(), func() { h.Close() }, nil
	}
	h, err := duckdb.OpenReadWrite(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open shared RW (%s): %w (serveur arrete ?)", path, err)
	}
	if err := migrerSchemaPartage(h.SQLDb(), slug); err != nil {
		h.Close()
		return nil, nil, err
	}
	return h.SQLDb(), func() { h.Close() }, nil
}

// matchsDejaProjetesMainmise : la cle de reprise, lue sur la VUE `_latest` (ADR 0026).
func matchsDejaProjetesMainmise(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT DISTINCT match_id FROM match_lead_share_latest`)
	if err != nil {
		return nil, fmt.Errorf("matchs deja projetes (mainmise): %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("matchs deja projetes (mainmise, scan): %w", err)
		}
		out[id] = true
	}
	return out, rows.Err()
}

// passeMainmise : ce que la projection d'un lot a besoin de tenir.
type passeMainmise struct {
	db  *sql.DB
	pr  *titlePkg.PathResolver
	o   leadShareOptions
	reg *mappings.RegulationSet
}

// projeterCorpus lit (et ecrit, hors --dry-run) les matchs du lot, UN ARTEFACT A LA FOIS.
func (p passeMainmise) projeterCorpus(ctx context.Context, candidats []string, dejaEcrits map[string]bool) bilanLeadShareBackfill {
	b := bilanLeadShareBackfill{raisons: map[string]int{}}
	persister := persist.NewLeadSharePersister(p.db)
	projetes := 0
	for _, id := range candidats {
		if p.o.limit > 0 && projetes >= p.o.limit {
			break
		}
		if !p.o.force && dejaEcrits[id] {
			b.dejaEnBase++
			continue
		}
		doc, err := replayartifacts.LireArtefactRange(p.pr.ReplayArtifactPath(p.o.titleSlug, id))
		if errors.Is(err, os.ErrNotExist) {
			b.sansArtefact++
			continue
		}
		if err != nil {
			fmt.Println("  ECHEC", id, ":", err)
			b.echecs++
			continue
		}
		idents, err := replayartifacts.IdentitesMainmise(ctx, p.db, []string{id})
		if err != nil {
			fmt.Println("  ECHEC", id, ":", err)
			b.echecs++
			continue
		}
		ident, ok := idents[id]
		if !ok {
			b.sansRegistre++
			continue
		}
		batch := replayartifacts.ProjeterMainmise(id, doc, ident, p.reg)
		projetes++
		comptabiliserMainmise(&b, batch)
		if p.o.dryRun {
			// UNE LIGNE PAR MATCH : l'operateur doit pouvoir CONTROLER ce qui sera ecrit.
			fmt.Println(ligneDryRunMainmise(id, batch))
			continue
		}
		if err := persister.PersistPass(ctx, batch); err != nil {
			// Jamais avale : l'echec d'UN match n'arrete pas la passe, mais il se voit.
			fmt.Println("  ECHEC", id, ":", err)
			b.echecs++
			continue
		}
		b.ecrits++
	}
	return b
}

// comptabiliserMainmise additionne les totaux du bilan : mesure, ou raison du refus.
func comptabiliserMainmise(b *bilanLeadShareBackfill, batch persist.LeadShareBatch) {
	if batch.Measured {
		b.mesures++
		return
	}
	b.raisons[batch.Reason]++
}

// ligneDryRunMainmise rend la ligne --dry-run d'UN match : mesure ou raison, duree, retournements
// et la part de chaque camp en pourcentage entier. Une raison vide se lit « - ».
func ligneDryRunMainmise(id string, batch persist.LeadShareBatch) string {
	var camps []string
	for _, r := range batch.Rows {
		camps = append(camps, fmt.Sprintf("%d:%d%%", r.Camp, int(100*float64(r.LeadMS)/float64(batch.MeasuredMS)+0.5)))
	}
	return fmt.Sprintf("  %-40s mesure=%-5v schema=%3d duree_s=%4d retournements=%d camps=%s raison=%s",
		id, batch.Measured, batch.DocSchema, batch.MeasuredMS/1000, batch.LeadChanges,
		strings.Join(camps, " "), raisonOuTiret(batch.Reason))
}

// syntheseMainmise rend le bilan imprime : les comptes, le taux de matchs mesures et les raisons
// de refus ventilees, triees par effectif decroissant.
func syntheseMainmise(b bilanLeadShareBackfill, duree time.Duration) string {
	var s strings.Builder
	projetes := b.mesures + b.nonMesures()
	fmt.Fprintf(&s, "mainmise : %d ecrits, %d deja en base, %d sans artefact, %d sans registre, %d echecs — %s\n",
		b.ecrits, b.dejaEnBase, b.sansArtefact, b.sansRegistre, b.echecs, duree)
	if projetes > 0 {
		fmt.Fprintf(&s, "projetes : %d, dont %d mesures (%.0f %%) et %d non mesures\n",
			projetes, b.mesures, 100*float64(b.mesures)/float64(projetes), b.nonMesures())
	}
	type kv struct {
		raison string
		n      int
	}
	var tri []kv
	for r, n := range b.raisons {
		tri = append(tri, kv{r, n})
	}
	sort.Slice(tri, func(i, j int) bool {
		if tri[i].n != tri[j].n {
			return tri[i].n > tri[j].n
		}
		return tri[i].raison < tri[j].raison
	})
	for _, e := range tri {
		fmt.Fprintf(&s, "  non mesure : %-22s %d\n", e.raison, e.n)
	}
	return s.String()
}
