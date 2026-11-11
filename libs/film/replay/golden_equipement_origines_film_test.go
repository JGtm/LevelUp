package replay

// golden_equipement_origines_film_test.go — LES FILMS DE GARDE DU GARDE-RAIL FAMILLE x ORIGINE : un film qu aucun des huit
// builds figes ne remplace, fige pour ce seul garde-rail.
//
// POURQUOI. Les huit builds couvrent le ramasse et la reapparition (`000d5950`, du build actuel, en est : famille B,
// Super Fiesta), mais PAS l equipement de depart de match d un Super Fiesta du build actuel : le detecteur de menaces de
// `bc4bde17` (slot 518, Bazaar), cree 22,3 s avant la premiere frame, active quatre fois, que la jointure par rang ne
// publiait pas. Un build de plus dans la table `goldenBuilds` coute un fixture d entrees de plusieurs Mio et chacun des
// goldens par build : ce fichier garde a la place, pour ces films, le MINIMUM que la regle de publication lit — les
// lectures de l equipement porte et leurs denominateurs (le codec de production), les annonces du porteur, l origine
// et le pas, les slots dont la piste est publiee, la palette du film et les trajectoires des objets que les piles nomment.
//
// LA PUBLICATION SE REJOUE par `buildCarriedEquipment` sur ce fixture, avec le catalogue du titre figé et le catalogue de
// l energie des equipements : le meme constructeur que la production.
//
// REGENERATION (jamais d edition a la main) : depuis le film, UN FILM PAR PROCESSUS,
//
//	REPLAY_FILM_CACHE=<repo>/data/cache/film_chunks go test ./replay/ \
//	  -run 'TestGoldenEquipementOriginesRegenerate/fixtures' -update-golden-equipement-origines
//
// puis la table, depuis les fixtures :
//
//	go test ./replay/ -run 'TestGoldenEquipementOriginesRegenerate/table' -update-golden-equipement-origines

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// filmDeGarde : un film hors de la table des builds, sa carte et son build.
type filmDeGarde struct{ Short8, Map, Build string }

// filmsDeGarde : le Super Fiesta du build actuel (`bc4bde17`, Bazaar), dont le detecteur de menaces de depart de match est
// le temoin nominal.
func filmsDeGarde() []filmDeGarde { return []filmDeGarde{{"bc4bde17", "Bazaar", "HI_1_13_0"}} }

func (f filmDeGarde) path() string {
	return filepath.Join(goldenDir, "equipement_origines_"+f.Short8+".bin.gz")
}

// faitsDeGarde : le minimum que la publication de l equipement porte lit d un film.
type faitsDeGarde struct {
	origine, pas uint64
	pistes       []uint32
	palette      map[int]string
	annonces     []types.EquipmentChange
	lectures     []types.CarriedEquipmentRead
	stats        types.CarriedEquipmentStats
	projectiles  []types.ProjectileTrack
}

// encodeFaitsDeGarde ecrit les faits (les lectures par le codec de production).
func encodeFaitsDeGarde(w *gwriter, f faitsDeGarde) {
	w.u(f.origine)
	w.u(f.pas)
	w.u(uint64(len(f.pistes)))
	for _, s := range f.pistes {
		w.u(uint64(s))
	}
	rangs := make([]int, 0, len(f.palette))
	for r := range f.palette {
		rangs = append(rangs, r)
	}
	slices.Sort(rangs)
	w.u(uint64(len(rangs)))
	for _, r := range rangs {
		w.u(uint64(r)) //nolint:gosec // rangs positifs
		w.str(f.palette[r])
	}
	w.u(uint64(len(f.annonces)))
	for _, a := range f.annonces {
		w.u(a.TimestampUS)
		w.u(uint64(a.Slot))
		w.str(string(a.Kind))
	}
	encodeEquipementPorte(w, f.lectures, f.stats)
	w.u(uint64(len(f.projectiles)))
	for _, p := range f.projectiles {
		w.u(uint64(p.Slot))
		w.u(uint64(p.Gen))
		w.u(uint64(len(p.Pts)))
		for _, s := range p.Pts {
			w.u(s.TimestampUS)
			w.f32(s.X)
			w.f32(s.Y)
			w.f32(s.Z)
			w.bool8(s.AtRest)
		}
	}
}

// decodeFaitsDeGarde relit ce qu [encodeFaitsDeGarde] ecrit.
func decodeFaitsDeGarde(r *greader) faitsDeGarde {
	f := faitsDeGarde{origine: r.u(), pas: r.u(), palette: map[int]string{}}
	for range r.compte(1) {
		f.pistes = append(f.pistes, uint32(r.u())) //nolint:gosec // ecrits depuis des uint32
	}
	for range r.compte(2) {
		rang := int(r.u()) //nolint:gosec // rangs
		f.palette[rang] = r.str()
	}
	for range r.compte(3) {
		a := types.EquipmentChange{TimestampUS: r.u(), Slot: uint32(r.u())} //nolint:gosec // ecrits depuis des uint32
		a.Kind = types.EquipmentChangeKind(r.str())
		f.annonces = append(f.annonces, a)
	}
	f.lectures, f.stats = decodeEquipementPorte(r)
	for range r.compte(3) {
		p := types.ProjectileTrack{Slot: uint32(r.u()), Gen: uint32(r.u())} //nolint:gosec // ecrits depuis des uint32
		for range r.compte(8) {
			s := types.ProjectileSample{TimestampUS: r.u()}
			s.X, s.Y, s.Z = r.f32(), r.f32(), r.f32()
			s.AtRest = r.bool8()
			p.Pts = append(p.Pts, s)
		}
		f.projectiles = append(f.projectiles, p)
	}
	return f
}

// releveDuFilmDeGarde : un film de garde, publie par le constructeur de production depuis son fixture.
func releveDuFilmDeGarde(t *testing.T, film filmDeGarde) releve {
	t.Helper()
	raw, err := os.ReadFile(film.path()) //nolint:gosec // chemin construit depuis la table des films de garde
	if err != nil {
		t.Fatalf("fixture du film de garde %s absent (%v) — regenerer avec %s=<cache> -run "+
			"TestGoldenEquipementOriginesRegenerate/fixtures -update-golden-equipement-origines", film.Short8, err, miniFilmCacheEnv)
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("fixture %s : gzip illisible : %v", film.Short8, err)
	}
	defer func() { _ = zr.Close() }()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(zr); err != nil {
		t.Fatalf("fixture %s : decompression : %v", film.Short8, err)
	}
	r := &greader{b: buf.Bytes()}
	f := decodeFaitsDeGarde(r)
	if r.err != nil || r.off != len(r.b) {
		t.Fatalf("fixture %s : relecture invalide (err %v, %d/%d octets lus)", film.Short8, r.err, r.off, len(r.b))
	}
	c := releve{lectures: f.lectures, annonces: annoncesDuPorteur(f.annonces), origine: f.origine, pas: f.pas,
		pistes: map[uint32]bool{}}
	tracks := make([]Track, 0, len(f.pistes))
	for _, s := range f.pistes {
		c.pistes[s] = true
		tracks = append(tracks, Track{Slot: s})
	}
	c.publies, _, _ = buildCarriedEquipment(entreeDEquipementPorte{reads: f.lectures, st: f.stats,
		palette: &AbilityPalette{Families: f.palette}, tracks: tracks, projectiles: f.projectiles,
		refs:   nouvelleReferenceDesObjetsPortes(goldenEnergyCatalog(t), goldenCatalog(t).EquipmentFamilies),
		origin: f.origine, step: f.pas})
	return c
}

// faitsDeGardeDuFilm decode un film (le contexte de la cuisson) et en tire les faits de garde.
func faitsDeGardeDuFilm(t *testing.T, cache string, film filmDeGarde) faitsDeGarde {
	t.Helper()
	entry, err := goldenBuild{Map: film.Map}.mapQuant(t)
	if err != nil {
		t.Fatalf("carte %q : %v", film.Map, err)
	}
	g, err := decodeFilmInputsAvecRoster(film.Short8, filepath.Join(cache, film.Short8), entry, nil)
	if err != nil {
		t.Fatalf("decodage de %s : %v", film.Short8, err)
	}
	opt := g.options()
	opt.Labels, opt.EquipmentEnergy, opt.MapQuant = goldenCatalog(t), goldenEnergyCatalog(t), &entry
	doc := BuildFromPositions(t.Context(), film.Short8, "halo_infinite", g.Positions, g.Fire, opt)
	f := faitsDeGarde{origine: g.Positions[0].TimestampUS, pas: uint64(doc.FrameIntervalMS) * 1000,
		palette: map[int]string{}, lectures: g.CarriedEquipment, stats: g.CarriedEquipmentStats}
	for _, p := range g.Positions {
		f.origine = min(f.origine, p.TimestampUS)
	}
	for _, tr := range doc.Tracks {
		if !slices.Contains(f.pistes, tr.Slot) {
			f.pistes = append(f.pistes, tr.Slot)
		}
	}
	slices.Sort(f.pistes)
	for k, lab := range doc.AbilityLabels {
		if rang, err := strconv.Atoi(k); err == nil && lab.Family != "" {
			f.palette[rang] = lab.Family
		}
	}
	for _, a := range g.EquipmentChanges {
		if a.Kind == types.EquipmentTaken || a.Kind == types.EquipmentSpawned {
			f.annonces = append(f.annonces, types.EquipmentChange{TimestampUS: a.TimestampUS, Slot: a.Slot, Kind: a.Kind})
		}
	}
	// Les trajectoires des seuls objets que les piles i28 nomment : ce que la position de repos lit.
	nommes := map[types.LifeKey]bool{}
	for _, l := range g.CarriedEquipment {
		for _, v := range l.Tracked {
			nommes[v] = true
		}
	}
	for _, p := range g.Projectiles {
		if nommes[types.LifeKey{Slot: p.Slot, Gen: p.Gen}] {
			f.projectiles = append(f.projectiles, p)
		}
	}
	return f
}

// ecrireLesFixturesDeGarde : la porte d ecriture des fixtures des films de garde (cf. l en-tete).
func ecrireLesFixturesDeGarde(t *testing.T) {
	t.Helper()
	cache := os.Getenv(miniFilmCacheEnv)
	if cache == "" {
		t.Skip("regeneration des fixtures des films de garde : " + miniFilmCacheEnv + " non defini")
	}
	for _, film := range filmsDeGarde() {
		f := faitsDeGardeDuFilm(t, cache, film)
		w := &gwriter{}
		encodeFaitsDeGarde(w, f)
		var out bytes.Buffer
		zw, err := gzip.NewWriterLevel(&out, gzip.BestCompression)
		if err != nil {
			t.Fatalf("gzip : %v", err)
		}
		if _, err := zw.Write(w.b); err != nil {
			t.Fatalf("gzip : %v", err)
		}
		if err := zw.Close(); err != nil {
			t.Fatalf("gzip : %v", err)
		}
		if err := os.WriteFile(film.path(), out.Bytes(), 0o600); err != nil {
			t.Fatalf("ecriture %s : %v", film.path(), err)
		}
		t.Errorf("REECRIT %s (%d lectures, %d pistes, %d trajectoires, %d octets) — relancer sans -update pour verifier",
			film.path(), len(f.lectures), len(f.pistes), len(f.projectiles), out.Len())
	}
}
