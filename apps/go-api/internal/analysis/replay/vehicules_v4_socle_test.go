package replay

// vehicules_v4_socle_test.go — SOCLE COMMUN des instruments du lot V4 (couverture des episodes
// d occupation, et rattachement des tirs en vehicule). LECTURE SEULE, garde par V4_ROOT /
// V4_FILMS : sans env, tous les tests du lot sautent proprement.
//
// POURQUOI UN SOCLE PLUTOT QUE DE REPRENDRE `v3dContexte`. Les instruments V1/V3 mesuraient des
// PRIMITIVES (trou de position, oracle geometrique) sur des nuages reconstruits a la main. Le
// lot V4 mesure ce que la PRODUCTION publie, et l ecart entre deux variantes de la production :
// son contexte doit donc etre celui de `BuildFromFilm`, aux memes largeurs de bloc MPP, au meme
// decoupage d axe, au meme pont slot -> joueur. Un socle qui divergerait de `build.go` rendrait
// des chiffres incomparables a l artefact.
//
//	CGO_ENABLED=0 V4_ROOT=<depot>/data/cache \
//	  V4_FILMS="0d76e8f1:Behemoth,fccc61cd:Launch Site" \
//	  go test ./internal/analysis/replay/ -run TestV4 -v -timeout 120m

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// Gardes d environnement du lot V4.
const (
	v4RootEnv  = "V4_ROOT"
	v4FilmsEnv = "V4_FILMS"
)

// v4Ctx porte, pour UN film, exactement ce que `BuildFromPositions` recoit pour poser le calque
// des vehicules et celui des tirs.
type v4Ctx struct {
	film  v0Film
	dir   string
	bip   []filmdec.BipedPosition
	scan  VehicleScan
	fire  []filmdec.FireEvent
	own   OwnerReport
	clock replayClock
	// lives / vehBySlot / spawns sont les entrees DEJA derivees de `scan`, pour ne pas les
	// recalculer a chaque etage.
	lives     []vehicleLife
	vehBySlot map[uint32][]filmdec.BipedPosition
	spawns    map[filmdec.EquipmentLifeKey]filmdec.EquipmentCreation
	// lifeBySlot indexe les vies par slot, et slots liste TOUS les slots de vehicule (ceux du
	// nuage ET ceux qui n ont qu une naissance), TRIES. Les deux existent pour le cout : la
	// mesure interroge le voisin le plus proche des dizaines de milliers de fois, et un balayage
	// lineaire des vies a chaque interrogation rendait l instrument inutilisable.
	lifeBySlot map[uint32][]vehicleLife
	slots      []uint32
}

// v4Corpus lit le corpus du lot. Forme « short8:Nom de carte, ... » — le nom de carte peut
// contenir des espaces (Launch Site), la coupure se fait donc sur le PREMIER deux-points.
func v4Corpus(t *testing.T) []v0Film {
	t.Helper()
	v := os.Getenv(v4FilmsEnv)
	if v == "" {
		t.Skipf("mesure non demandee : %s vide (« short8:Carte, ... »)", v4FilmsEnv)
	}
	var out []v0Film
	for _, s := range strings.Split(v, ",") {
		id, carte, ok := strings.Cut(strings.TrimSpace(s), ":")
		if !ok {
			t.Fatalf("entree de corpus invalide %q : forme attendue « short8:Carte »", s)
		}
		out = append(out, v0Film{ID: strings.TrimSpace(id), Carte: strings.TrimSpace(carte)})
	}
	return out
}

// v4Root rend la racine du cache film.
func v4Root(t *testing.T) string {
	t.Helper()
	root := os.Getenv(v4RootEnv)
	if root == "" {
		t.Skipf("mesure non demandee : %s vide (racine du cache film)", v4RootEnv)
	}
	return root
}

// v4Carte rend l entree de catalogue d une carte NOMMEE (bornes + decoupage d axe).
func v4Carte(t *testing.T, root, carte string) (filmdec.MapQuantEntry, bool) {
	t.Helper()
	cat, err := filmdec.LoadMapQuantCatalog(filepath.Join(attRefDir(root), "map_quant_bounds.json"))
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	e, err := cat.Lookup(carte)
	if err != nil {
		t.Logf("carte %q absente du catalogue de bornes (%v)", carte, err)
		return filmdec.MapQuantEntry{}, false
	}
	return e, true
}

// v4Decode reproduit le contexte de production d un film. L APPELANT DETIENT LockProcessDecode
// et restaure `filmdec.WorldObjectPrecision` — c est la discipline de tous les instruments du
// dossier, et elle n est pas negociable (les largeurs d axe sont un global de paquet).
func v4Decode(t *testing.T, root string, f v0Film) (v4Ctx, bool) {
	t.Helper()
	ctx := v4Ctx{film: f, dir: objChunkDir(root, f.ID)}
	if filmdec.CountFilmChunks(ctx.dir) == 0 {
		t.Logf("V4 %s : film absent du cache — saute", f.ID)
		return ctx, false
	}
	entry, ok := v4Carte(t, root, f.Carte)
	if !ok {
		return ctx, false
	}
	filmdec.SetWorldObjectPrecisionFromLayout(entry.Layout())
	wr := entry.Range()
	bip, ok := v4Bipedes(t, ctx.dir, entry, &wr)
	if !ok {
		return ctx, false
	}
	ctx.bip = bip
	// MEME ORDRE QUE `BuildFromFilm` : les poses calibrent les largeurs du bloc MPP, et le
	// balayage des vehicules les reinstalle. Sans cette calibration, AUCUNE famille ne se
	// resoudrait — en silence.
	_, pst := decodeFilmPlacements(ctx.dir, &wr)
	ctx.scan = decodeFilmVehicleScan(ctx.dir, &wr, pst.Calibration.Widths)
	if !ctx.scan.Scanned {
		t.Logf("V4 %s : balayage ti=40 sans resultat — rien a mesurer", f.ID)
		return ctx, false
	}
	fire, err := filmdec.ScanFilmFireEvents(ctx.dir)
	if err != nil {
		t.Logf("V4 %s : events de tir illisibles (%v)", f.ID, err)
	}
	ctx.fire = fire
	ctx.own = v4Pont(t, ctx.dir, bip, fire)
	ctx.clock = v4Horloge(bip)
	ctx.lives = vehicleLives(ctx.scan.Keyframes)
	ctx.vehBySlot = vehiclePositionsBySlot(ctx.scan.Positions)
	ctx.spawns = vehicleSpawnsByLife(ctx.scan.Creations)
	ctx.lifeBySlot = map[uint32][]vehicleLife{}
	for _, l := range ctx.lives {
		if _, seen := ctx.lifeBySlot[l.key.Slot]; !seen {
			ctx.slots = append(ctx.slots, l.key.Slot)
		}
		ctx.lifeBySlot[l.key.Slot] = append(ctx.lifeBySlot[l.key.Slot], l)
	}
	sort.Slice(ctx.slots, func(i, j int) bool { return ctx.slots[i] < ctx.slots[j] })
	return ctx, true
}

// v4Bipedes lit le nuage bipede aux MEMES reglages que la production (cap capture, decoupage
// d axe du catalogue), TRIE par instant comme `BuildFromPositions` le fait.
func v4Bipedes(
	t *testing.T, dir string, entry filmdec.MapQuantEntry, wr *filmdec.Vec3Range,
) ([]filmdec.BipedPosition, bool) {
	t.Helper()
	opt := filmdec.DefaultScanFilmOptions()
	opt.WorldRange = wr
	opt.CaptureDirs = true
	if lay := entry.Layout(); lay.Valid() {
		opt.Layout = &lay
	}
	pos, err := filmdec.ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Logf("V4 : nuage bipede illisible (%v)", err)
		return nil, false
	}
	sort.SliceStable(pos, func(i, j int) bool { return pos[i].TimestampUS < pos[j].TimestampUS })
	return pos, true
}

// v4Pont construit le pont slot -> joueur EXACTEMENT comme `BuildFromPositions`.
func v4Pont(
	t *testing.T, dir string, bip []filmdec.BipedPosition, fire []filmdec.FireEvent,
) OwnerReport {
	t.Helper()
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Logf("V4 : fil des morts illisible (%v) — pont vide", err)
		return OwnerReport{}
	}
	idx, err := ScanFilmPlayerIndices(dir, rosterFromDeaths(deaths))
	if err != nil {
		t.Logf("V4 : index joueur illisible (%v) — pont sans identite", err)
	}
	table, _ := injectiveOrEmpty(idx)
	return buildOwners(indexBySlot(bip), deaths, table, fireRefs(fire))
}

// v4Horloge rend l horloge du document (origine = premier paquet, pas = FrameIntervalMS defaut).
func v4Horloge(bip []filmdec.BipedPosition) replayClock {
	if len(bip) == 0 {
		return replayClock{}
	}
	origin := bip[0].TimestampUS
	step := uint64(DefaultFrameIntervalMS) * 1000
	return replayClock{origin: origin, step: step, frames: frameSpan(bip, origin, step)}
}

// v4HeldPos rend la position TENUE d un slot de vehicule a un instant : le dernier echantillon a
// cet instant ou avant, DANS la fenetre de la vie active ; a defaut la NAISSANCE de cette vie.
//
// POURQUOI « TENUE » ET PAS « ECHANTILLONNEE ». Un vehicule qui ne bouge pas ne replique pas sa
// position — c est la mesure du present lot. Exiger un echantillon RECENT (la regle de
// production, 1 s) revient a exiger que le vehicule ait bouge dans la seconde qui precede
// l embarquement, ce qui est faux pour tout vehicule GARE : precisement le cas nominal d un
// embarquement. La position tenue est le seul etat que le film prouve a cet instant.
//
// `ageUS` vaut `v4AgeSpawn` quand la position vient de la naissance (aucun echantillon anterieur).
func v4HeldPos(ctx v4Ctx, slot uint32, atUS uint64) (x, y float32, ageUS uint64, ok bool) {
	l, alive := v4LifeAt(ctx.lifeBySlot[slot], atUS)
	if !alive {
		return 0, 0, 0, false
	}
	pts := ctx.vehBySlot[slot]
	i := sort.Search(len(pts), func(k int) bool { return pts[k].TimestampUS > atUS })
	if i > 0 && pts[i-1].TimestampUS >= l.loUS {
		p := pts[i-1]
		return p.X, p.Y, atUS - p.TimestampUS, true
	}
	if sp, has := ctx.spawns[l.key]; has {
		return sp.X, sp.Y, v4AgeSpawn, true
	}
	return 0, 0, 0, false
}

// v4AgeSpawn marque une position tenue qui vient de la NAISSANCE et non d un echantillon.
const v4AgeSpawn = ^uint64(0)

// v4LifeAt rend la VIE d un instant parmi celles d UN SEUL slot (meme predicat que
// `vehicleLifeAt`, mais il rend la vie entiere : la mesure a besoin de sa fenetre, pas seulement
// de sa cle). Les fenetres d un meme slot ne se recouvrent pas, la reponse est unique.
func v4LifeAt(lives []vehicleLife, atUS uint64) (vehicleLife, bool) {
	for _, l := range lives {
		if atUS >= l.loUS && atUS <= l.hiUS {
			return l, true
		}
	}
	return vehicleLife{}, false
}

// v4NearestHeld rend le vehicule VIVANT le plus proche EN PLAN d une position, a la position
// TENUE, et la distance. Deterministe (slots tries).
func v4NearestHeld(
	ctx v4Ctx, e filmdec.BipedPosition,
) (slot uint32, dist float64, age uint64, ok bool) {
	best, bestD, bestAge, found := uint32(0), 0.0, uint64(0), false
	for _, s := range ctx.slots {
		x, y, a, has := v4HeldPos(ctx, s, e.TimestampUS)
		if !has {
			continue
		}
		d := planDist(e.X, e.Y, x, y)
		if !found || d < bestD {
			best, bestD, bestAge, found = s, d, a, true
		}
	}
	return best, bestD, bestAge, found
}

// v4Percentiles rend les percentiles demandes d un echantillon (copie triee).
func v4Percentiles(vals []float64, ps ...float64) []float64 {
	if len(vals) == 0 {
		return make([]float64, len(ps))
	}
	c := append([]float64(nil), vals...)
	sort.Float64s(c)
	out := make([]float64, 0, len(ps))
	for _, p := range ps {
		i := int(p * float64(len(c)-1))
		out = append(out, c[i])
	}
	return out
}
