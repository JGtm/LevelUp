package main

// cmd_tactical_rasters.go — sous-commande `levelup tactical-rasters --backfill`.
//
// ELLE DEPOSE LE SIDECAR D'OCCUPATION DES ARTEFACTS DEJA CUITS. Le fil de l'eau ne couvre
// que les artefacts construits par les cycles a venir (etape post-sync, cf.
// internal/sync/replayartifacts/raster.go) : le corpus existant passe par cette commande.
//
// # ELLE NE CUIT RIEN, ET C'EST SA PROPRIETE PRINCIPALE
//
// La source est l'ARTEFACT deja range (`data/cache/replays/{slug}/{short}.json`), jamais un
// film. Aucun appel a `replaybuild`, aucun decodage, aucun processus enfant, aucune bombe
// RAM — un garde-rail l'interdit par construction
// (internal/archlint/no_cuisson_depuis_tactique_test.go). Une passe est une LECTURE de
// fichiers suivie d'une ecriture de petits JSON.
//
// # ELLE N'OUVRE AUCUNE BASE, PAS MEME EN LECTURE
//
// Le sidecar est PAR MATCH et ANONYME : ni carte, ni equipe, ni resultat n'y entrent (ils
// se resolvent a la lecture, cf. domain/tactical_raster.go). Il n'y a donc rien a
// demander a DuckDB — ce qui rend cette passe jouable serveur allume comme serveur
// arrete, et incapable de corrompre quoi que ce soit. L'enumeration vient du SEUL lecteur
// d'artefacts du depot (`service.ReplayService.AvailableSet`), pas d'un second listing.
//
// # ELLE EST IDEMPOTENTE, ET LA CLE DE FRAICHEUR EST DOUBLE
//
// Un sidecar est refait s'il MANQUE, si son propre `schema_version` n'est plus le courant,
// ou si son `artifact_schema_version` ne correspond plus a celui de l'artefact (donc apres
// une re-cuisson). Une seconde passe immediate ecrit ZERO fichier.
//
// Usage :
//
//	levelup tactical-rasters --backfill              # tout le corpus du titre par defaut
//	levelup tactical-rasters --backfill --dry-run    # compte ce qui serait ecrit
//	levelup tactical-rasters --backfill --title halo_5
//	levelup tactical-rasters --backfill --limit 50

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"time"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/service"
	"levelup/go-api/internal/sync/replayartifacts"
)

// tacticalRastersOptions : les reglages de la passe.
type tacticalRastersOptions struct {
	titleSlug string
	backfill  bool
	dryRun    bool
	limit     int
}

// bilanTacticalRasters : ce que la passe a fait, et pourquoi elle a saute ce qu'elle a
// saute. `lus` compte les artefacts EXAMINES ; `sautes` ceux dont le sidecar etait deja a
// jour — les distinguer est ce qui permet de lire une sortie sans se demander si quelque
// chose s'est mal passe.
type bilanTacticalRasters struct {
	lus, ecrits, sautes, echecs int
}

func runTacticalRasters(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("tactical-rasters", flag.ExitOnError)
	o := tacticalRastersOptions{}
	fs.StringVar(&o.titleSlug, "title", titlePkg.DefaultSlug, "slug du titre")
	fs.BoolVar(&o.backfill, "backfill", false, "rattraper les sidecars des artefacts deja cuits")
	fs.BoolVar(&o.dryRun, "dry-run", false, "compter ce qui serait ecrit, sans rien ecrire")
	fs.IntVar(&o.limit, "limit", 0, "borne le nombre d'artefacts examines (0 = tous)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !o.backfill {
		// PAS DE MODE PAR DEFAUT. Cette commande n'a qu'un seul travail, et il se demande :
		// une invocation nue qui se mettrait a parcourir tout le corpus serait une
		// surprise, meme sans effet destructeur.
		return fmt.Errorf("rien a faire : preciser --backfill (voir `levelup help`)")
	}

	// LE GATE EST UNE CAPABILITY, JAMAIS UN SLUG (ratchet no_slug_comparison_test.go).
	// Meme cle que la production du fil de l'eau : pas d'artefact, rien a projeter.
	caps, err := capabilitesDuTitre(cfg, o.titleSlug)
	if err != nil {
		return err
	}
	if !caps.Has(games.CapFilmReplayArtifact) {
		fmt.Printf("le titre %s ne declare pas %s : aucun artefact a projeter (passe vide)\n",
			o.titleSlug, string(games.CapFilmReplayArtifact))
		return nil
	}

	ctx := context.Background()
	shorts, err := artefactsDuTitre(ctx, cfg, o.titleSlug)
	if err != nil {
		return err
	}
	debut := time.Now()
	b := projeterCorpusRasters(ctx, cfg, o, shorts)
	// LE JOURNAL EST STRUCTURE (CLAUDE.md n 3) ; la ligne imprimee, elle, est la REPONSE
	// de la commande a son operateur — les deux disent la meme chose a deux publics.
	slog.InfoContext(ctx, "rasters tactiques : passe terminee",
		"titleSlug", o.titleSlug, "lus", b.lus, "ecrits", b.ecrits, "sautes", b.sautes,
		"echecs", b.echecs, "dry_run", o.dryRun, "duration", time.Since(debut))
	fmt.Printf("rasters tactiques : %d lus, %d ecrits, %d deja a jour, %d echecs — %s%s\n",
		b.lus, b.ecrits, b.sautes, b.echecs, time.Since(debut).Round(time.Second),
		suffixeDryRun(o.dryRun))
	return nil
}

// suffixeDryRun dit, sur la ligne de bilan, que rien n'a ete ecrit.
func suffixeDryRun(dryRun bool) string {
	if dryRun {
		return " (--dry-run : aucun fichier ecrit)"
	}
	return ""
}

// artefactsDuTitre enumere les artefacts presents, par LE lecteur d'artefacts du depot.
//
// UN SEUL LECTEUR (doctrine du lot B) : `AvailableSet` fait UN listing de dossier et ne
// compte que les `{short}.json` de premier niveau — le sous-dossier `rasters/` que cette
// passe remplit en est donc exclu par construction. C'est un COMPORTEMENT, garde par un
// test de comportement (`cmd_tactical_rasters_test.go`, « l'enumeration ne se compte pas
// elle-meme ») : le ratchet `no_second_artifact_sink_test.go` ne garde PAS cela — il
// compte les cablages de `SetArtifactStoredSink`, c'est-a-dire le puits d'ECRITURE.
// Le service est monte SANS repo de cartes (`nil`, cas servi) : cette passe n'a besoin
// d'aucune base.
//
// L'ordre est stable (tri des cles) : une passe interrompue reprend au meme endroit, et
// deux passes se comparent ligne a ligne.
func artefactsDuTitre(ctx context.Context, cfg *config.AppConfig, slug string) ([]string, error) {
	set, err := service.NewReplayService(slug, cfg.RepoRoot, nil).AvailableSet(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing des artefacts de %s: %w", slug, err)
	}
	shorts := make([]string, 0, len(set))
	for short := range set {
		shorts = append(shorts, short)
	}
	sort.Strings(shorts)
	return shorts, nil
}

// projeterCorpusRasters examine les artefacts un par un : rien ne survit d'un match a
// l'autre que les compteurs du bilan (lecon du chantier backfill-replay, qui a sature la
// machine le 2026-08-20 avec une map globale vivante).
func projeterCorpusRasters(ctx context.Context, cfg *config.AppConfig,
	o tacticalRastersOptions, shorts []string) bilanTacticalRasters {
	b := bilanTacticalRasters{}
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	for _, short := range shorts {
		if o.limit > 0 && b.lus >= o.limit {
			break
		}
		b.lus++
		cible := pr.TacticalRasterPath(o.titleSlug, short)
		if sidecarAJour(ctx, cible) {
			b.sautes++
			continue
		}
		s, err := replayartifacts.ProjeterRasterTactique(pr.ReplayArtifactPath(o.titleSlug, short))
		if err != nil {
			// Jamais avale : l'echec d'UN artefact n'arrete pas la passe, mais il se voit.
			slog.ErrorContext(ctx, "rasters tactiques : artefact non projete",
				"short_id", short, "err", err)
			b.echecs++
			continue
		}
		if dejaProjete(ctx, cible, s.ArtifactSchemaVersion) {
			b.sautes++
			continue
		}
		if o.dryRun {
			slog.InfoContext(ctx, "rasters tactiques : sidecar a ecrire (--dry-run)",
				"short_id", short, "joueurs", len(s.Joueurs), "schema_artefact", s.ArtifactSchemaVersion)
			b.ecrits++
			continue
		}
		if err := replayartifacts.EcrireSidecarRaster(cible, s); err != nil {
			slog.ErrorContext(ctx, "rasters tactiques : sidecar non ecrit",
				"short_id", short, "err", err)
			b.echecs++
			continue
		}
		b.ecrits++
	}
	return b
}

// sidecarAJour rend vrai quand le sidecar existant est certainement courant, SANS ouvrir
// l'artefact.
//
// POURQUOI CE RACCOURCI EXISTE : un artefact pese quelques megaoctets, un sidecar quelques
// kilooctets. Sans lui, une passe de controle sur un corpus de milliers de matchs relirait
// des gigaoctets pour n'ecrire rien. Il est SUR parce qu'un artefact ne peut pas porter un
// schema superieur a celui que le binaire courant sait produire : un sidecar qui declare
// deja `replay.SchemaVersion` ne peut pas etre en retard sur son artefact.
//
// Tous les autres cas — sidecar absent, illisible, d'un autre format, ou projete d'un
// artefact plus ancien — retombent sur la comparaison complete (dejaProjete).
func sidecarAJour(ctx context.Context, path string) bool {
	s, ok := lireSidecarRaster(ctx, path)
	return ok && s.ArtifactSchemaVersion == replay.SchemaVersion
}

// dejaProjete compare le sidecar en place au schema de l'artefact qui vient d'etre lu.
func dejaProjete(ctx context.Context, path string, schemaArtefact int) bool {
	s, ok := lireSidecarRaster(ctx, path)
	return ok && s.ArtifactSchemaVersion == schemaArtefact
}

// lireSidecarRaster lit le sidecar en place et dit s'il est exploitable EN L'ETAT.
//
// LE PREDICAT DE FRAICHEUR EST CELUI DU SERVICE (`domain.SidecarRasterCourant`) : format,
// grille et unite de temps. Cette passe y AJOUTE la comparaison a l'artefact
// (`artifact_schema_version`), qu'elle seule peut faire — elle a l'artefact en main. En
// deux definitions, le remede que le service prescrit dans son WARN restait un no-op sur
// un sidecar au bon schema mais a un autre pas (constat C3 de la revue).
//
// Absent : silence, c'est le cas NOMINAL d'un artefact jamais projete. Illisible ou d'un
// autre format : (_, faux) ET une ligne de journal — un fichier corrompu n'est pas une
// absence, et le taire ferait disparaitre un incident derriere une reecriture muette
// (CLAUDE.md n 3, anti-patron n 10).
func lireSidecarRaster(ctx context.Context, path string) (domain.TacticalRasterSidecar, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.WarnContext(ctx, "rasters tactiques : sidecar illisible, il sera reecrit",
				"path", path, "raison", "lecture", "err", err)
		}
		return domain.TacticalRasterSidecar{}, false
	}
	var s domain.TacticalRasterSidecar
	if err := json.Unmarshal(raw, &s); err != nil {
		slog.WarnContext(ctx, "rasters tactiques : sidecar illisible, il sera reecrit",
			"path", path, "raison", "parse", "err", err)
		return domain.TacticalRasterSidecar{}, false
	}
	if !domain.SidecarRasterCourant(&s) {
		slog.DebugContext(ctx, "rasters tactiques : sidecar d'un autre temps, il sera reecrit",
			"path", path, "schema_version", s.SchemaVersion, "pas_m", s.PasM,
			"pas_echantillon_ms", s.PasEchantillonMs)
		return domain.TacticalRasterSidecar{}, false
	}
	return s, true
}
