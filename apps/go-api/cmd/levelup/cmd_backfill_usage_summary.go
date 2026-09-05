package main

// cmd_backfill_usage_summary.go — sous-commande `levelup backfill-usage-summary`.
//
// ELLE REMPLIT `shared.match_usage_players` + `shared.match_usage_films` (resume d usage
// equipement/socles, chantier session-usage 2026-09-04) POUR LES ARTEFACTS DEJA CUITS. Le fil
// de l eau ne couvre que les artefacts construits par les cycles a venir (etape post-sync
// replaybuild, cf. internal/sync/replayartifacts/usage.go) : le corpus existant (~114
// artefacts) passe par cette commande.
//
// # ELLE NE DECODE AUCUN FILM
//
// La source est l ARTEFACT de rejeu (`data/cache/replays/{slug}/{short8}.json`), projete par
// `replay.BuildUsageSummary` — la meme projection que le fil de l eau, a l octet pres. Un
// match sans artefact est saute (il releve de `backfill-replay`, qui CUIT ; cette passe-ci
// RESUME). Lecture UN ARTEFACT A LA FOIS, jamais de map globale vivante (lecon du chantier
// backfill-replay, qui a sature la machine le 2026-08-20).
//
// # ELLE EST REPRENABLE, ET LA CLE EST (summary_rev, artifact_schema)
//
// Un match dont la passe COURANTE (vue `match_usage_films_latest`, ADR 0026) porte la
// revision de projection courante (`replay.UsageSummaryRev`) ET le schema de l artefact
// sur disque est saute. Interrompre puis relancer reprend donc ou elle en etait ; un
// artefact RE-CUIT a un schema plus recent est re-resume tout seul. `--force` re-resume
// tout — c est ce qu il faut le jour ou une regle de projection change sans changer la
// revision (ne devrait jamais arriver : changer une regle DOIT incrementer UsageSummaryRev).
//
// # `--dry-run` EST L INSTRUMENT DES CONTROLES CROISES DU PILOTE
//
// Il projette et IMPRIME les compteurs par match — prises de socle nommees, anonymes,
// occupations de socle de bonus par famille — sans rien ecrire. C est ce qui permet de
// verifier les attendus du handoff (696a9d7c = 26 nommees / 8 anonymes, 10 powerup_camo
// en bonus et ZERO en pad_pickups ; b8a44fe8 = 51 / 11 ; session 2026-07-31 = 193 / 102).
//
// Usage (SERVEUR ARRETE — `OpenReadWrite` echoue si le lock est tenu ; meme precondition
// que backfill-killsource, y compris pour --dry-run qui joue les migrations) :
//
//	levelup backfill-usage-summary --dry-run                    # compteurs par match, aucune ecriture
//	levelup backfill-usage-summary --match 696a9d7c-...         # un seul match
//	levelup backfill-usage-summary                              # tout le corpus, reprenable
//	levelup backfill-usage-summary --force                      # re-resume meme ce qui est a jour

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/duckdb"
)

// usageSummaryOptions : les reglages de la passe.
type usageSummaryOptions struct {
	titleSlug string
	limit     int
	force     bool
	dryRun    bool
	match     string
}

// passeCouranteUsage : ce que la vue `_latest` dit d un match deja resume.
type passeCouranteUsage struct {
	rev    string
	schema int
}

// bilanUsageBackfill : ce que la passe a fait — et pourquoi elle a saute ce qu elle a saute.
type bilanUsageBackfill struct {
	ecrits, dejaAJour, sansArtefact, echecs int
	// Totaux du corpus projete, pour les controles croises de session (--dry-run).
	totalNommees, totalAnonymes int
	totalPowerups               map[string]int
}

func runBackfillUsageSummary(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("backfill-usage-summary", flag.ExitOnError)
	o := usageSummaryOptions{}
	fs.StringVar(&o.titleSlug, "title", titlePkg.DefaultSlug, "slug du titre")
	fs.IntVar(&o.limit, "limit", 0, "borne le nombre de matchs projetes (0 = tous)")
	fs.BoolVar(&o.force, "force", false, "re-resumer meme les matchs deja a jour pour (revision, schema) courants")
	fs.BoolVar(&o.dryRun, "dry-run", false, "projeter et imprimer les compteurs par match sans rien ecrire")
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

	// Les tables et la vue de reprise doivent exister AVANT de lire ou d ecrire — cette
	// commande tourne SERVEUR ARRETE, donc rien n a joue les migrations pour elle.
	if err := migrerSchemaPartage(db, o.titleSlug); err != nil {
		return err
	}

	// LE GATE EST UNE CAPABILITY, JAMAIS UN SLUG (ratchet no_slug_comparison_test.go) : un
	// titre qui ne declare pas `film.usage_summary` fait une passe VIDE, proprement.
	caps, err := capabilitesDuTitre(cfg, o.titleSlug)
	if err != nil {
		return err
	}
	if !caps.Has(games.CapFilmUsageSummary) {
		fmt.Printf("le titre %s ne declare pas %s : rien a resumer (passe vide)\n",
			o.titleSlug, string(games.CapFilmUsageSummary))
		return nil
	}

	candidats, err := candidatsUsage(ctx, db, o)
	if err != nil {
		return err
	}
	dejaResumes := map[string]passeCouranteUsage{}
	if !o.force {
		if dejaResumes, err = passesCourantesUsage(ctx, db); err != nil {
			return err
		}
	}
	debut := time.Now()
	b := resumerCorpus(ctx, db, pr, o, candidats, dejaResumes)
	fmt.Printf("resume usage : %d ecrits, %d deja a jour, %d sans artefact, %d echecs — %s\n",
		b.ecrits, b.dejaAJour, b.sansArtefact, b.echecs, time.Since(debut).Round(time.Second))
	if o.dryRun {
		fmt.Printf("totaux du corpus projete : %d prises nommees, %d anonymes, powerups %s\n",
			b.totalNommees, b.totalAnonymes, usagePowerupsTexte(b.totalPowerups))
	}
	return nil
}

// candidatsUsage : les matchs a examiner, dans un ordre stable — `--match` seul, sinon tout
// le registre (c est l artefact qui filtre ensuite : pas d artefact, pas de resume).
func candidatsUsage(ctx context.Context, db *sql.DB, o usageSummaryOptions) ([]string, error) {
	if o.match != "" {
		return []string{o.match}, nil
	}
	return matchsDuRegistre(ctx, db, 0)
}

// passesCourantesUsage : la cle de reprise, lue sur la VUE `_latest` (ADR 0026 — une passe
// ancienne, deja supplantee, ne doit pas faire sauter un match).
func passesCourantesUsage(ctx context.Context, db *sql.DB) (map[string]passeCouranteUsage, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT DISTINCT match_id, summary_rev, artifact_schema FROM match_usage_films_latest`)
	if err != nil {
		return nil, fmt.Errorf("matchs deja resumes: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]passeCouranteUsage{}
	for rows.Next() {
		var id string
		var p passeCouranteUsage
		if err := rows.Scan(&id, &p.rev, &p.schema); err != nil {
			return nil, fmt.Errorf("matchs deja resumes (scan): %w", err)
		}
		out[id] = p
	}
	return out, rows.Err()
}

// resumerCorpus projette (et ecrit, hors --dry-run) les matchs du lot, UN ARTEFACT A LA
// FOIS : rien ne survit d un match a l autre que les compteurs du bilan.
func resumerCorpus(
	ctx context.Context, db *sql.DB, pr *titlePkg.PathResolver, o usageSummaryOptions,
	candidats []string, dejaResumes map[string]passeCouranteUsage,
) bilanUsageBackfill {
	b := bilanUsageBackfill{totalPowerups: map[string]int{}}
	p := persist.NewUsageSummaryPersister(db)
	projetes := 0
	for _, id := range candidats {
		if o.limit > 0 && projetes >= o.limit {
			break
		}
		s, etat := projeterUnArtefact(pr.ReplayArtifactPath(o.titleSlug, id), id, o, dejaResumes)
		switch etat {
		case usageSansArtefact:
			b.sansArtefact++
			continue
		case usageDejaAJour:
			b.dejaAJour++
			continue
		case usageEchec:
			b.echecs++
			continue
		}
		projetes++
		b.totalNommees += s.Match.PadNamed
		b.totalAnonymes += s.Match.PadUnnamed
		for fam, n := range s.Match.PowerupPadPickups {
			b.totalPowerups[fam] += n
		}
		if o.dryRun {
			fmt.Printf("  %-40s nommees=%3d anonymes=%3d joueurs=%2d powerups=%s\n",
				id, s.Match.PadNamed, s.Match.PadUnnamed, len(s.Players),
				usagePowerupsTexte(s.Match.PowerupPadPickups))
			continue
		}
		if err := p.PersistPass(ctx, id, s); err != nil {
			// Jamais avale : l echec d UN match n arrete pas la passe, mais il se voit.
			fmt.Printf("  ECHEC %s : %v\n", id, err)
			b.echecs++
			continue
		}
		b.ecrits++
	}
	return b
}

// Etats d un match examine par la passe.
type etatUsageMatch int

const (
	usageAProjeter etatUsageMatch = iota
	usageSansArtefact
	usageDejaAJour
	usageEchec
)

// projeterUnArtefact lit UN artefact et le projette, ou dit pourquoi il ne le fait pas.
// L artefact entier est relache a la sortie — seule la projection (quelques centaines
// d octets par joueur) survit.
func projeterUnArtefact(
	path, matchID string, o usageSummaryOptions, dejaResumes map[string]passeCouranteUsage,
) (*replay.UsageSummary, etatUsageMatch) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, usageSansArtefact
		}
		fmt.Printf("  ECHEC %s : lecture artefact: %v\n", matchID, err)
		return nil, usageEchec
	}
	var doc replay.ReplayDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		fmt.Printf("  ECHEC %s : parse artefact: %v\n", matchID, err)
		return nil, usageEchec
	}
	// La reprise se decide sur le schema de l artefact SUR DISQUE : un artefact re-cuit a
	// un schema plus recent que la passe courante doit etre re-resume.
	if !o.force {
		if rec, ok := dejaResumes[matchID]; ok &&
			rec.rev == replay.UsageSummaryRev && rec.schema == doc.SchemaVersion {
			return nil, usageDejaAJour
		}
	}
	s := replay.BuildUsageSummary(&doc)
	return &s, usageAProjeter
}

// usagePowerupsTexte : la ventilation des bonus, cles triees — une sortie de passe doit
// etre reproductible, y compris dans son ordre.
func usagePowerupsTexte(m map[string]int) string {
	if len(m) == 0 {
		return "{}"
	}
	fams := make([]string, 0, len(m))
	for fam := range m {
		fams = append(fams, fam)
	}
	sort.Strings(fams)
	parts := make([]string, 0, len(fams))
	for _, fam := range fams {
		parts = append(parts, fmt.Sprintf("%s:%d", fam, m[fam]))
	}
	return "{" + strings.Join(parts, " ") + "}"
}
