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
	selectOnly := flag.Bool("select-only", false,
		"dimensionner le corpus et s'arreter (aucun film decode)")
	role := flag.String("role", string(mapvar.RoleStrongholdZone),
		"role de zone mesure (`strongholds_zone`, `totalcontrol_zone`, ...)")
	census := flag.Bool("census", false,
		"recenser le corpus PAR MODE (films en cache, artefacts, schemas) et s'arreter ; "+
			"aucun filtre de mode, aucun film decode")
	dump := flag.Bool("dump", false,
		"detailler chaque action (confrontation a un releve terrain)")
	child := flag.Bool("child", false,
		"ENFANT d'une passe : mesure le seul film de -match sous plafond memoire et rend un "+
			"code de protocole (pose par le parent, pas par l'operateur)")
	flag.Parse()

	code, err := run(*slug, *cacheDir, *matchArg, runTuning{
		witness: *witness, maxGap: *maxGap, selectOnly: *selectOnly, dump: *dump,
		census: *census, role: *role, child: *child, cacheDir: *cacheDir,
	})
	if err != nil {
		slog.Error("mesure d'attribution de zone", "err", err)
		os.Exit(1)
	}
	// LE CODE DE SORTIE EST LE PROTOCOLE quand ce processus est un enfant (cf. passe.go). Sur
	// le chemin du parent il vaut zero, et `os.Exit(0)` est un no-op lisible.
	os.Exit(code)
}

// runTuning regroupe les reglages numeriques : sans ce regroupement, run() depasserait
// les 5 parametres du seuil projet.
type runTuning struct {
	witness float64
	maxGap  int
	// selectOnly arrete apres le dimensionnement du corpus. Le decodage d'un film coute
	// des dizaines de secondes ; savoir COMBIEN de matchs sont mesurables ne doit pas
	// exiger de tous les decoder.
	selectOnly bool
	// dump detaille chaque action au lieu des seuls agregats (cf. dump.go).
	dump bool
	// census recense le corpus PAR MODE et s'arrete (cf. census.go). Il precede selectOnly :
	// il ne filtre aucun mode, la ou selectOnly ne dimensionne que celui de la mesure.
	census bool
	// role est le ROLE de zone mesure. Il etait ecrit en dur (`strongholds_zone`) tant que
	// Bastion etait la seule zone TENUE decodee ; Total Control a ouvert la question, et rien
	// dans la mesure n'etait propre a Bastion — seule la constante l'etait.
	//
	// LE ROLE N'EST PAS VALIDE ICI, ET C'EST VOULU : un role inconnu du catalogue ne rend
	// simplement AUCUNE forme, et le corpus sort a zero eligible avec « sans formes au
	// catalogue » — un diagnostic plus parlant qu'un refus au demarrage. La liste fermee des
	// roles vit ou elle doit vivre : au chargement de la table du titre (loader_objective_roles).
	role string
	// child : ce processus est l'ENFANT d'une passe (cf. passe.go). Il mesure UN film sous la
	// sentinelle memoire et rend un code de protocole.
	child bool
	// cacheDir est recopie ici pour que le parent puisse le RETRANSMETTRE a ses enfants : sans
	// lui, un enfant re-deduirait la racine du cache et pourrait viser une autre arborescence.
	cacheDir string
}

func run(slug, cacheDir, matchArg string, tune runTuning) (int, error) {
	repoRoot, err := title.FindRepoRoot()
	if err != nil {
		return 0, err
	}
	if cacheDir == "" {
		cacheDir = filepath.Join(repoRoot, "data", "cache")
	}
	paths := title.NewPathResolver(repoRoot)

	bounds, err := filmdec.LoadMapQuantCatalog(paths.MapQuantBoundsPath(slug))
	if err != nil {
		return 0, err
	}
	zonesCat, err := replay.LoadMapObjectives(paths.MapObjectivesPath(slug))
	if err != nil {
		return 0, err
	}
	// Lecture SEULE : OpenReadForQuery reste correct meme si le serveur tient la base en
	// ecriture (modele mono-process, ADR 0013/0016).
	db, release, err := duckdb.OpenReadForQuery(paths.SharedDBPath(slug))
	if err != nil {
		return 0, err
	}
	defer release()

	ctx := context.Background()
	// LE RECENSEMENT PASSE AVANT TOUT LE RESTE : il ne filtre aucun mode et n'a besoin ni des
	// bornes de carte ni du catalogue de formes, que la MESURE exige.
	if tune.census {
		return 0, runCensus(ctx, db, slug, cacheDir, repoRoot)
	}
	candidates, err := loadCandidates(ctx, db, matchArg)
	if err != nil {
		return 0, err
	}
	r := &runner{
		slug: slug, cacheDir: cacheDir, db: db,
		bounds: bounds, zones: zonesCat,
		role:    mapvar.Role(tune.role),
		offset:  mapvar.Vec3{X: tune.witness, Y: tune.witness},
		maxGap:  tune.maxGap,
		results: nil,
	}
	eligible := r.selectEligible(candidates)
	// L'ENFANT NE REIMPRIME PAS LA SELECTION : le parent l'a deja publiee, et la repeter par
	// film noierait le journal de la passe.
	if !tune.child {
		printSelection(candidates, eligible, r.rejects)
	}
	if tune.selectOnly {
		return 0, nil
	}
	if tune.child {
		return runChild(ctx, r, eligible, tune), nil
	}
	// LA BOUCLE SUR LES FILMS PASSE PAR L'EXECUTEUR BORNE, TOUJOURS — meme pour un seul film.
	// Un cas particulier « un film, autant le faire ici » rouvrirait exactement la breche du
	// 2026-08-26 : c'est UN film BTB qui a suffi a prendre la machine.
	return 0, runParent(ctx, repoRoot, eligible, tune)
}
