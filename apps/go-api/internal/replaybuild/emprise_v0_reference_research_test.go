package replaybuild

// emprise_v0_reference_research_test.go — LOT V0 DU PLAN `.ai/PLAN_EMPRISE_VIES_2026-09-28.md`,
// VOLET CUISSON : la REFERENCE des mesures V0.1 et V0.2.
//
// # CE QU'IL REND
//
// Pour chaque film, le document de rejeu CONSTRUIT EN MEMOIRE par le code actuel — les memes
// etapes que `BuildBytes`, dans le meme ordre (bascule, statborg, entrees de catalogue, options,
// `replay.BuildFromFilmAvecFaits`) —, puis les calques que le lot compare :
//
//	V0.1  le calque vehicules : episodes d'occupation (`rides[]`, `src`, occupant, frames) ;
//	V0.2  les porteurs d'objectif : drapeau (`FlagCarries`, etat `carried`), crane
//	      (`SkullCarries`), bombe (`BombCarries`), couronne VIP (`VipCrown`) ;
//
// et l'HORLOGE de l'axe de frames : pas, origine exacte (premier paquet de position de la
// cuisson, minimum des positions qu'elle a balayees), `originMs`, horloge du film et calage du
// pont de la cuisson (`coverage.bridge.deathOffsetMs`). S'y ajoutent les deux entrees de
// catalogue que le volet porteurs rejoue (socles de drapeau, catalogue de libelles).
//
// # CE QU'IL NE FAIT PAS
//
// Il n'ECRIT RIEN hors du repertoire du lot : les faits de film ne sont ni relus ni ranges
// (`sansFaitsPersistes`, et `ecrireLesFaits` n'est jamais appele — c'est la seule etape de
// `documentDeLaCuisson` omise), aucun artefact n'est range, aucune base n'est ouverte (les faits
// du match et les noms de carte viennent du volet collecteur, `k1/`). Les catalogues sont ceux,
// versionnes, du depot ; les films sont lus dans le cache.
//
// SANS LES VARIABLES, IL SE SAUTE :
//
//	EMPRISE_V0_FILMS=<match_id,...> EMPRISE_V0_DIR=<scratch> EMPRISE_V0_CACHE=<racine data/cache> \
//	  go test ./internal/replaybuild/ -run '^TestEmpriseV0Reference$' -v -count=1 -timeout 60m

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/testutil"
)

// v0Identite : ce que le volet collecteur a lu en base (noms de carte, faits du match).
type v0Identite struct {
	MatchID string          `json:"matchId"`
	Noms    []string        `json:"noms"`
	Faits   port.MatchFacts `json:"faits"`
}

// v0Periode : un portage, en frames du document.
type v0Periode struct {
	XUID string `json:"xuid"`
	T0   int    `json:"t0"`
	T1   int    `json:"t1"`
}

// v0Reference : ce que ce volet pose pour les deux autres.
type v0Reference struct {
	MatchID         string                  `json:"matchId"`
	Variante        string                  `json:"variante"`
	Module          string                  `json:"module"`
	FrameIntervalMS int                     `json:"frameIntervalMs"`
	FrameCount      int                     `json:"frameCount"`
	OriginMs        *int64                  `json:"originMs"`
	FilmClockUS     uint64                  `json:"filmClockUs"`
	OrigineUS       uint64                  `json:"origineUs"`
	CalageRefMs     *int64                  `json:"calageRefMs"`
	SchemaVersion   int                     `json:"schemaVersion"`
	Rides           []json.RawMessage       `json:"rides"`
	Drapeau         []v0Periode             `json:"drapeau"`
	Crane           []v0Periode             `json:"crane"`
	Bombe           []v0Periode             `json:"bombe"`
	VIP             []v0Periode             `json:"vip"`
	Couverture      map[string]any          `json:"couverture"`
	Socles          json.RawMessage         `json:"socles"`
	Libelles        json.RawMessage         `json:"libelles"`
	CuissonMS       float64                 `json:"cuissonMs"`
	VehiculesCouv   *replay.VehicleCoverage `json:"couvertureVehicules,omitempty"`
}

func TestEmpriseV0Reference(t *testing.T) {
	dir, cache := os.Getenv("EMPRISE_V0_DIR"), os.Getenv("EMPRISE_V0_CACHE")
	var films []string
	for _, f := range strings.Split(os.Getenv("EMPRISE_V0_FILMS"), ",") {
		if f = strings.TrimSpace(f); f != "" {
			films = append(films, f)
		}
	}
	if len(films) == 0 || dir == "" || cache == "" {
		t.Skip("EMPRISE_V0_FILMS, EMPRISE_V0_DIR et EMPRISE_V0_CACHE requis : instrument saute")
	}
	repoRoot, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	// LA SENTINELLE DU DEPOT : une cuisson BTB monte a plusieurs Gio ; au-dela du plafond dur, le
	// processus s'arrete plutot que de faire suffoquer la machine de travail.
	garde := filmproc.Arm("emprise-v0-reference", 8, func(pic uint64) {
		fmt.Fprintf(os.Stderr, "EMPRISE V0 : empreinte %.2f Gio > plafond — arret\n", float64(pic)/(1<<30))
		os.Exit(3)
	})
	defer garde.Disarm()
	for _, id := range films {
		b, err := NewBuilder(context.Background(), repoRoot, title.DefaultSlug)
		if err != nil {
			t.Fatalf("builder : %v", err)
		}
		b.sansFaitsPersistes = true
		ref := v0Cuire(t, b, id, dir, cache)
		v0Poser(t, dir, title.FilmShortMatchID(id), ref)
	}
	t.Logf("pic memoire de la passe de cuisson : %.2f Gio", float64(garde.Peak())/(1<<30))
}

// v0Cuire construit le document EN MEMOIRE et en extrait la reference.
func v0Cuire(t *testing.T, b *Builder, id, dir, cache string) v0Reference {
	t.Helper()
	court := title.FilmShortMatchID(id)
	var ident v0Identite
	blob, err := os.ReadFile(filepath.Join(dir, "k1", court+".json"))
	if err != nil || json.Unmarshal(blob, &ident) != nil {
		t.Fatalf("%s : k1 illisible (%v) — jouer TestEmpriseV0Identites d'abord", court, err)
	}
	debut := time.Now()
	doc, faits, film, opts := v0Document(t, b, id, ident, filmcache.ChunkDir(cache, court))
	ref := v0Reference{
		MatchID: id, Variante: ident.Faits.GameVariantName, FrameIntervalMS: doc.FrameIntervalMS,
		FrameCount: doc.FrameCount, OriginMs: doc.OriginMs, SchemaVersion: doc.SchemaVersion,
		CuissonMS: float64(time.Since(debut).Microseconds()) / 1000,
	}
	if doc.Coverage != nil {
		ref.CalageRefMs = doc.Coverage.Bridge.DeathOffsetMs
		ref.VehiculesCouv = doc.Coverage.Vehicles
		ref.Couverture = map[string]any{"flagCarries": doc.Coverage.FlagCarries,
			"skullCarries": doc.Coverage.SkullCarries, "bombCarries": doc.Coverage.BombCarries,
			"vipCrown": doc.Coverage.VipCrown}
	}
	if clk, err := decfilm.ScanClockOrigin(film); err == nil {
		ref.FilmClockUS = clk
	}
	if faits != nil {
		for i := range faits.Facts.Positions {
			if ts := faits.Facts.Positions[i].TimestampUS; ref.OrigineUS == 0 || ts < ref.OrigineUS {
				ref.OrigineUS = ts
			}
		}
	}
	v0Calques(t, &ref, doc, opts)
	return ref
}

// v0Document rejoue `BuildBytes` jusqu'au document, SANS ranger les faits ni serialiser.
func v0Document(t *testing.T, b *Builder, id string, ident v0Identite, filmDir string) (
	replay.ReplayDocument, *replay.FilmFactsFile, *decfilm.Film, replay.Options,
) {
	t.Helper()
	ctx := context.Background()
	entry, err := b.ResolveMapEntry(ident.Noms)
	if err != nil {
		t.Fatalf("%s : carte : %v", id, err)
	}
	src, err := b.entreesDeLaCuisson(ctx, id, filmDir, entry)
	if err != nil {
		t.Fatalf("%s : entrees de cuisson : %v", id, err)
	}
	if src.faits != nil || src.film == nil {
		t.Fatalf("%s : la bascule n'a pas rendu la branche DECODE", id)
	}
	facts := ident.Faits
	stats := assemblerFilmStats(ctx, id, src.statborg, facts, src.deaths)
	if stats.score != nil {
		stats.score.TargetScore, _ = b.regulation.ScoreTarget(facts.GameVariantName)
		stats.score.HoldTicksPerPoint, _ = b.regulation.HoldTicksPerPoint(facts.GameVariantName)
	}
	cat := b.collecterEntreesCatalogue(ctx, id, ident.Noms, facts, &stats, src)
	opts := b.buildReplayOptions(ctx, entry, facts, cat, &stats)
	doc, faits, err := replay.BuildFromFilmAvecFaits(ctx, id, b.titleSlug, src.film, opts)
	if err != nil {
		t.Fatalf("%s : decodage : %v", id, err)
	}
	return doc, faits, src.film, opts
}

// v0Calques extrait les episodes d'occupation et les quatre calques de porteurs.
func v0Calques(t *testing.T, ref *v0Reference, doc replay.ReplayDocument, opts replay.Options) {
	t.Helper()
	for _, v := range doc.Vehicles {
		for _, r := range v.Rides {
			blob, err := json.Marshal(r)
			if err != nil {
				t.Fatalf("episode : %v", err)
			}
			ref.Rides = append(ref.Rides, blob)
		}
	}
	for _, fc := range doc.FlagCarries {
		for _, s := range fc.Spans {
			if s.State == replay.FlagStateCarried && s.XUID != nil {
				ref.Drapeau = append(ref.Drapeau, v0Periode{XUID: *s.XUID, T0: s.T0, T1: s.T1})
			}
		}
	}
	for _, c := range doc.SkullCarries {
		ref.Crane = append(ref.Crane, v0Periode{XUID: c.XUID, T0: c.T0, T1: c.T1})
	}
	for _, c := range doc.BombCarries {
		ref.Bombe = append(ref.Bombe, v0Periode{XUID: c.XUID, T0: c.T0, T1: c.T1})
	}
	for _, c := range doc.VipCrown {
		ref.VIP = append(ref.VIP, v0Periode{XUID: c.XUID, T0: c.T0, T1: c.T1})
	}
	var err error
	if ref.Socles, err = json.Marshal(opts.Flag.Spawns); err != nil {
		t.Fatalf("socles : %v", err)
	}
	if ref.Libelles, err = json.Marshal(opts.Labels); err != nil {
		t.Fatalf("libelles : %v", err)
	}
}

// v0Poser ecrit la reference et sa ligne de journal.
func v0Poser(t *testing.T, dir, court string, ref v0Reference) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "ref"), 0o750); err != nil {
		t.Fatalf("repertoire ref : %v", err)
	}
	blob, err := json.Marshal(ref)
	if err != nil {
		t.Fatalf("%s : serialisation : %v", court, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ref", court+".json"), blob, 0o600); err != nil {
		t.Fatalf("%s : ecriture : %v", court, err)
	}
	calage := "inconnu"
	if ref.CalageRefMs != nil {
		calage = fmt.Sprintf("%d ms", *ref.CalageRefMs)
	}
	t.Logf("%s  %-26s schema=%d cuisson %6.0f ms  episodes=%d drapeau=%d crane=%d bombe=%d vip=%d "+
		"origine=%d µs calage=%s", court, ref.Variante, ref.SchemaVersion, ref.CuissonMS,
		len(ref.Rides), len(ref.Drapeau), len(ref.Crane), len(ref.Bombe), len(ref.VIP),
		ref.OrigineUS, calage)
}
