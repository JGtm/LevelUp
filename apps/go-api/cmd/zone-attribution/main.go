// cmd/zone-attribution — MESURE le croisement « quel joueur est DANS quelle zone a
// l'instant d'une prise », sur les films reellement en cache.
//
// Il n'invente rien : il assemble trois pieces qui existaient deja separement, et il
// PUBLIE le taux obtenu avec son temoin negatif.
//
//	les formes    data/titles/{slug}/reference/map_objectives.json (replay.LoadMapObjectives)
//	les instants  les evenements nommes du statborg, identifies par xuid (objectiveevents)
//	les positions les trajectoires decodees du film (replay.BuildFromFilm)
//
// # Pourquoi un binaire et pas un test
//
// Le croisement lui-meme est pur et teste (internal/analysis/replay). La MESURE, elle,
// exige la base (le pont slot -> xuid lit `match_participants`) et le cache film local,
// qui n'est pas versionne : ce sont des dependances d'orchestration, donc un cmd.
//
// # Le temoin negatif n'est pas optionnel
//
// « Le joueur est dans la zone » n'est pas une mesure tant qu'on n'a pas montre que des
// zones DEPLACEES attrapent nettement moins de monde. Le chantier a deja du retirer un
// critere en or tautologique pour cette raison ; le rapport signal/temoin est donc
// imprime a cote du taux, toujours.
//
// Usage (depuis apps/go-api) :
//
//	go run ./cmd/zone-attribution
//	go run ./cmd/zone-attribution -cache <repo>/data/cache -match 415f2c6c
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"path/filepath"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
)

// defaultWitnessOffsetM : decalage du temoin negatif, en metres, en x ET en y. Douze
// metres est la valeur du releve terrain d'origine (etat de l'art, Q2) : assez pour
// sortir de toute zone mesuree (rayon max 5,0 m, demi-cote max ~5,1 m), assez peu pour
// rester sur la carte et donc sur du sol foule.
const defaultWitnessOffsetM = 12.0

func main() {
	slug := flag.String("title", title.DefaultSlug, "slug du titre")
	cacheDir := flag.String("cache", "", "racine du cache film (defaut : <repo>/data/cache)")
	matchArg := flag.String("match", "", "restreindre a un match (forme courte ou complete)")
	witness := flag.Float64("witness", defaultWitnessOffsetM, "decalage du temoin negatif, en metres")
	maxGap := flag.Int("max-gap", replay.DefaultMaxGapFrames, "tolerance position/action, en frames")
	flag.Parse()

	if err := run(*slug, *cacheDir, *matchArg, runTuning{witness: *witness, maxGap: *maxGap}); err != nil {
		slog.Error("mesure d'attribution de zone", "err", err)
		os.Exit(1)
	}
}

// runTuning regroupe les reglages numeriques : sans ce regroupement, run() depasserait
// les 5 parametres du seuil projet.
type runTuning struct {
	witness float64
	maxGap  int
}

func run(slug, cacheDir, matchArg string, tune runTuning) error {
	repoRoot, err := title.FindRepoRoot()
	if err != nil {
		return err
	}
	if cacheDir == "" {
		cacheDir = filepath.Join(repoRoot, "data", "cache")
	}
	paths := title.NewPathResolver(repoRoot)

	bounds, err := filmdec.LoadMapQuantCatalog(paths.MapQuantBoundsPath(slug))
	if err != nil {
		return err
	}
	zonesCat, err := replay.LoadMapObjectives(paths.MapObjectivesPath(slug))
	if err != nil {
		return err
	}
	// Lecture SEULE : OpenReadForQuery reste correct meme si le serveur tient la base en
	// ecriture (modele mono-process, ADR 0013/0016).
	db, release, err := duckdb.OpenReadForQuery(paths.SharedDBPath(slug))
	if err != nil {
		return err
	}
	defer release()

	ctx := context.Background()
	candidates, err := loadCandidates(ctx, db, matchArg)
	if err != nil {
		return err
	}
	r := &runner{
		slug: slug, cacheDir: cacheDir, db: db,
		bounds: bounds, zones: zonesCat,
		role:    mapvar.RoleStrongholdZone,
		offset:  mapvar.Vec3{X: tune.witness, Y: tune.witness},
		maxGap:  tune.maxGap,
		results: nil,
	}
	eligible := r.selectEligible(candidates)
	printSelection(candidates, eligible, r.rejects)
	for _, m := range eligible {
		r.results = append(r.results, r.measure(ctx, m))
	}
	printResults(r.results, tune)
	return nil
}
