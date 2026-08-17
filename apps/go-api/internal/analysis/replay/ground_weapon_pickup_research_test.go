package replay

// ground_weapon_pickup_research_test.go — LE RAMASSAGE d'une arme au sol : disparition BORNEE
// par le recensement des images-cles, DATEE par le passage d'un joueur (phase 2 du plan
// .ai/V7.5/replay2d/PLAN_ARMES_AU_SOL_2E_LECTURE.md).
//
// L'ENTREE EST CELLE DE LA PHASE 1 : les records de CREATION `ti=42` dont le mot MPP de 32 bits
// se resout dans le catalogue d'armes (`weaponv3.KnownWeaponHigh32`). Le filtre d'identite est
// le garde-fou de la decouverte 2 — le temoin fantome du balayage n'est PAS discriminant seul.
//
// DEUX POSITIONS, DEUX ROLES, et les confondre fausserait les deux mesures :
//   - la position de CREATION grappe les socles (item 1.2 : c'est la que l'arme APPARAIT) ;
//   - la position de REFERENCE date le ramassage (dernier point de la piste delta si l'objet a
//     bouge, position de creation sinon : c'est la qu'il EST au moment ou on le prend).
//
// CE QUE MESURE L'INSTRUMENT, UN FILM PAR PROCESSUS :
//
//	2.1  bornage par le recensement des images-cles, datation par le passage a < 1,5 m,
//	     distribution des distances et des largeurs d'intervalle, DEUX temoins ;
//	2.2  ORACLE INDEPENDANT : le loadout d'image-cle du ramasseur porte-t-il la famille
//	     ramassee a l'image-cle suivante (`ScanFilmKeyframeLoadouts`) ? Temoin : un autre
//	     joueur tire au sort a la meme image-cle ;
//	2.3  lignes `PICKUP` et `PADSTATE` — la proposition de format pour la phase 3 ;
//	2.4  cycle RE-MESURE depuis le ramassage, socle par socle, contre le cycle d'apparition
//	     de l'item 1.3.
//
// LECTURE SEULE, aucune base. UN SEUL decodage filmdec par process (`LockProcessDecode`),
// largeurs d'axe restaurees (`installWorldObjectPrecision`).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 GW_PICKUP=<repo>/data/cache/film_chunks/01e1f945 GW_PICKUP_MAP=Catalyst \
//	  go test ./internal/analysis/replay/ -run '^TestGroundWeaponPickups$' -timeout 30m -v

import (
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	gwPickupEnv       = "GW_PICKUP"        // repertoire des chunks du film
	gwPickupMapEnv    = "GW_PICKUP_MAP"    // nom de carte (bornes de dequantification)
	gwPickupBoundsEnv = "GW_PICKUP_BOUNDS" // catalogue map_quant_bounds.json (defaut : PathResolver)
)

// gwPickupTrackTolUS est la tolerance d'appariement entre un record de CREATION et la piste
// delta de la meme vie : 200 ms, soit `originDropWindowUS`. Le record de creation porte lui-meme
// la position i0, donc la piste commence en principe au MEME paquet ; la tolerance couvre le
// cas ou le premier point replique suit d'une frame ou deux.
const gwPickupTrackTolUS = originDropWindowUS

// gwPickupStatus* : les trois issues d'une apparition. Vocabulaire STABLE des lignes de sortie.
const (
	gwPickupStatusDated   = "dated"   // un joueur est passe a < 1,5 m dans l'intervalle
	gwPickupStatusUnknown = "unknown" // personne n'est passe : date = borne haute
	gwPickupStatusNever   = "never"   // encore recense a la derniere image-cle du film
)

// gwPickupFilm porte tout ce que le film rend, lu UNE fois : cinq balayages complets, pas un
// de plus.
type gwPickupFilm struct {
	film, mapName string
	positions     []filmdec.BipedPosition
	bySlot        map[uint32][]filmdec.BipedPosition
	lives         map[uint32][]equipLife
	kfTimes       []uint64
	seen          map[filmdec.EquipmentLifeKey][]uint64
	loadouts      map[uint64]map[uint32][]string
	tracks        map[filmdec.EquipmentLifeKey][]filmdec.ProjectileTrack
	spans         map[filmdec.EquipmentLifeKey][]filmdec.EquipmentLifeSpan
	filmEndUS     uint64
	rng           *rand.Rand
}

// gwPickupObject est une apparition retenue, bornee et datee.
type gwPickupObject struct {
	Key    filmdec.EquipmentLifeKey
	Appar  gwPadApparition // position de CREATION, classe, vie delta (entree de la phase 1)
	Pos    [3]float32      // position de REFERENCE (dernier point de piste, ou creation)
	Moved  bool
	Bounds gwPickupBounds
	Picker gwPickupHit
	Status string
}

func TestGroundWeaponPickups(t *testing.T) {
	dir := os.Getenv(gwPickupEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", gwPickupEnv)
	}
	entry := mapQuantEntryFromEnv(t, gwPickupMapEnv, gwPickupBoundsEnv)
	release := filmdec.LockProcessDecode()
	defer release()
	defer installWorldObjectPrecision(entry, dir)()

	wr := entry.Range()
	f := gwPickupRead(t, dir, &wr)
	f.film, f.mapName = filepath.Base(dir), os.Getenv(gwPickupMapEnv)
	t.Logf("FILM %s · carte %q (module %s) · largeurs %v · images-cles %d · fin %.1f s",
		f.film, f.mapName, entry.Module, entry.AxisWidths, len(f.kfTimes),
		float64(f.filmEndUS)/1e6)

	objs := gwPickupObjects(t, dir, &wr, f)
	gwPickupReport21(t, f, objs)
	gwPickupReport22(t, f, objs)
	gwPickupReport23(t, f, objs)
	gwPickupReport24(t, f, objs)
}

// gwPickupRead lit le film : nuage des bipedes, recensement et instants des images-cles,
// loadouts d'image-cle, pistes delta `ti=42`.
func gwPickupRead(t *testing.T, dir string, wr *filmdec.Vec3Range) *gwPickupFilm {
	t.Helper()
	pos, err := filmdec.ScanFilmBipedPositions(dir, gwPadsScanOptions(wr))
	if err != nil {
		t.Fatalf("nuage des bipedes indisponible : %v", err)
	}
	sort.Slice(pos, func(i, j int) bool { return pos[i].TimestampUS < pos[j].TimestampUS })
	f := &gwPickupFilm{
		positions: pos,
		bySlot:    map[uint32][]filmdec.BipedPosition{},
		lives:     equipmentLives(pos),
		tracks:    map[filmdec.EquipmentLifeKey][]filmdec.ProjectileTrack{},
		rng:       rand.New(rand.NewSource(gwPickupWitnessSeed)), //nolint:gosec // temoin reproductible
	}
	for _, p := range pos {
		f.bySlot[p.Slot] = append(f.bySlot[p.Slot], p)
		if p.TimestampUS > f.filmEndUS {
			f.filmEndUS = p.TimestampUS
		}
	}
	f.kfTimes, f.seen = gwPickupKeyframes(t, dir)
	if n := len(f.kfTimes); n > 0 && f.kfTimes[n-1] > f.filmEndUS {
		f.filmEndUS = f.kfTimes[n-1]
	}
	f.loadouts = gwPickupLoadouts(t, dir)
	tracks, err := filmdec.ScanFilmWorldObjects(dir, wr, filmdec.GroundWeaponTypeIndex)
	if err != nil {
		t.Fatalf("vies delta ti=42 : %v", err)
	}
	for _, tr := range tracks {
		k := filmdec.EquipmentLifeKey{Slot: tr.Slot, Gen: tr.Gen}
		f.tracks[k] = append(f.tracks[k], tr)
	}
	f.spans = filmdec.EquipmentLifeSpans(tracks)
	t.Logf("LECTURE — %d positions de bipede · %d slots · %d vies · %d cles `ti=42` recensees"+
		" · %d cles a piste delta", len(pos), len(f.bySlot), gwPadsCountLives(f.lives),
		len(f.seen), len(f.spans))
	return f
}

// gwPickupKeyframes rend les instants de TOUTES les images-cles du film et, par vie
// (slot, gen), ceux qui RECENSENT un record `ti=42`. C'est le walker durci (249/250 entites)
// qui recense — pas le balayage de familles, qui n'attrape que les records porteurs d'une
// famille connue et manquerait donc les objets muets.
func gwPickupKeyframes(t *testing.T, dir string) ([]uint64, map[filmdec.EquipmentLifeKey][]uint64) {
	t.Helper()
	n := filmdec.CountFilmChunks(dir)
	if n == 0 {
		t.Fatalf("aucun chunk film dans %s", dir)
	}
	var times []uint64
	seen := map[filmdec.EquipmentLifeKey][]uint64{}
	for c := 1; c <= n; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(data) {
			if p.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			times = append(times, p.TimestampUS)
			for _, r := range filmdec.WalkKeyframeWorld(p.Payload(data)) {
				if r.TI != filmdec.GroundWeaponTypeIndex {
					continue
				}
				k := filmdec.EquipmentLifeKey{Slot: uint32(r.Slot), Gen: uint32(r.Gen)}
				if v := seen[k]; len(v) > 0 && v[len(v)-1] == p.TimestampUS {
					continue // un record par vie et par image-cle
				}
				seen[k] = append(seen[k], p.TimestampUS)
			}
		}
	}
	sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
	for k := range seen {
		sort.Slice(seen[k], func(i, j int) bool { return seen[k][i] < seen[k][j] })
	}
	return times, seen
}

// gwPickupLoadouts indexe les armes PORTEES par image-cle et par slot de bipede, repliees sur
// leur nom canonique. LE REPLI DES ALIAS EST OBLIGATOIRE : un meme canon apparait deux fois
// dans le record sous deux identifiants distincts, et comparer les identifiants bruts ferait
// echouer l'oracle sur la moitie des armes (piege documente par `buildLoadouts`).
func gwPickupLoadouts(t *testing.T, dir string) map[uint64]map[uint32][]string {
	t.Helper()
	raw, err := filmdec.ScanFilmKeyframeLoadouts(dir, loadoutFamilies())
	if err != nil {
		t.Fatalf("loadouts d'image-cle illisibles : %v", err)
	}
	out := map[uint64]map[uint32][]string{}
	for _, l := range raw {
		if out[l.TimestampUS] == nil {
			out[l.TimestampUS] = map[uint32][]string{}
		}
		for _, w := range l.Families {
			nom := gwPadsWeaponFamily(w)
			if !gwPickupHasFamily(out[l.TimestampUS][l.Slot], nom) {
				out[l.TimestampUS][l.Slot] = append(out[l.TimestampUS][l.Slot], nom)
			}
		}
	}
	return out
}

func gwPickupHasFamily(in []string, want string) bool {
	for _, f := range in {
		if f == want {
			return true
		}
	}
	return false
}

// gwPickupObjects rend les apparitions retenues, bornees et datees. Les creations sont d'abord
// groupees par cle pour que la REPRISE de cle borne la vie de la precedente.
func gwPickupObjects(
	t *testing.T, dir string, wr *filmdec.Vec3Range, f *gwPickupFilm,
) []gwPickupObject {
	t.Helper()
	band := filmdec.GroundWeaponSlotBand(dir)
	cre, st, err := filmdec.ScanFilmGroundWeaponCreationsForBand(dir, wr, band)
	if err != nil {
		t.Fatalf("creations ti=42 : %v", err)
	}
	known := loadoutFamilies()
	byKey := map[filmdec.EquipmentLifeKey][]filmdec.EquipmentCreation{}
	kept := 0
	for _, c := range cre {
		w, ok := gwPadsIdentity(c)
		if !ok || !known[w] {
			continue
		}
		kept++
		k := filmdec.EquipmentLifeKey{Slot: c.Slot, Gen: c.Gen}
		byKey[k] = append(byKey[k], c)
		if c.TimestampUS > f.filmEndUS {
			f.filmEndUS = c.TimestampUS
		}
	}
	out := make([]gwPickupObject, 0, kept)
	for k, list := range byKey {
		sort.Slice(list, func(i, j int) bool { return list[i].TimestampUS < list[j].TimestampUS })
		for i, c := range list {
			lifeEnd := f.filmEndUS
			if i+1 < len(list) {
				lifeEnd = list[i+1].TimestampUS
			}
			out = append(out, gwPickupObject{Key: k, Appar: gwPickupApparition(c, f)})
			gwPickupResolve(&out[len(out)-1], c, lifeEnd, f)
		}
	}
	sort.Slice(out, func(i, j int) bool { return gwPadsLess(out[i].Appar, out[j].Appar) })
	t.Logf("2.1 ENTREE — ancres %d · acceptees %d · RETENUES (identite croisee) %d · ecartees %d",
		st.Anchors, st.Accepted, kept, st.Accepted-kept)
	return out
}

// gwPickupApparition rend l'apparition au sens de la phase 1 : position de CREATION, classe
// `dropped`/`spawned` par la regle de production, et presence d'une vie delta sur la cle. Les
// trois definitions sont celles de la phase 1, sans une virgule de changement — le jeu
// `at_rest` de l'item 2.4 doit etre le MEME que celui de l'item 1.2.
func gwPickupApparition(c filmdec.EquipmentCreation, f *gwPickupFilm) gwPadApparition {
	w, _ := gwPadsIdentity(c)
	a := gwPadApparition{
		Kind: gwPadKindWeapon, Family: gwPadsWeaponFamily(w),
		X: c.X, Y: c.Y, Z: c.Z, TUS: c.TimestampUS,
		HasDelta: len(f.spans[filmdec.EquipmentLifeKey{Slot: c.Slot, Gen: c.Gen}]) > 0,
	}
	a.Class = gwPadsClass(f.lives, a)
	return a
}

// gwPickupResolve borne la disparition de l'objet puis la date.
func gwPickupResolve(o *gwPickupObject, c filmdec.EquipmentCreation, lifeEnd uint64, f *gwPickupFilm) {
	o.Pos, o.Moved = gwPickupRefPos(c, lifeEnd, f.tracks[o.Key])
	o.Bounds = gwPickupBoundsFrom(c.TimestampUS, lifeEnd, f.filmEndUS, f.kfTimes,
		gwPickupSeenWithin(f.seen[o.Key], c.TimestampUS, lifeEnd))
	if o.Bounds.NeverPicked {
		o.Status = gwPickupStatusNever
		return
	}
	o.Picker = gwPickupNearestPass(o.Pos, o.Bounds.LowUS, o.Bounds.HighUS, f.positions,
		originDropMaxDist)
	o.Status = gwPickupStatusUnknown
	if o.Picker.Found {
		o.Status = gwPickupStatusDated
	}
}

// gwPickupSeenWithin restreint le recensement d'une cle a la vie [t0, lifeEnd).
func gwPickupSeenWithin(seen []uint64, t0, lifeEnd uint64) []uint64 {
	lo := sort.Search(len(seen), func(i int) bool { return seen[i] >= t0 })
	hi := sort.Search(len(seen), func(i int) bool { return seen[i] >= lifeEnd })
	if hi < lo {
		hi = lo
	}
	return seen[lo:hi]
}

// gwPickupRefPos rend la position ou l'objet SE TROUVE quand on le ramasse : le dernier point
// de sa piste delta s'il a bouge dans sa vie, sa position de creation sinon.
func gwPickupRefPos(
	c filmdec.EquipmentCreation, lifeEnd uint64, tracks []filmdec.ProjectileTrack,
) ([3]float32, bool) {
	best, bestGap := -1, uint64(math.MaxUint64)
	for i, tr := range tracks {
		if len(tr.Pts) == 0 {
			continue
		}
		t0 := tr.Pts[0].TimestampUS
		if t0 >= lifeEnd || t0+gwPickupTrackTolUS < c.TimestampUS {
			continue
		}
		if g := gwPickupTimeGap(t0, c.TimestampUS); g < bestGap {
			best, bestGap = i, g
		}
	}
	if best < 0 {
		return [3]float32{c.X, c.Y, c.Z}, false
	}
	p := tracks[best].Pts[len(tracks[best].Pts)-1]
	return [3]float32{p.X, p.Y, p.Z}, true
}

// gwPickupDateUS rend l'instant retenu du ramassage : celui du passage quand il existe, la
// borne haute sinon (regle du plan : « aucun : `unknown`, date = borne haute »).
func (o gwPickupObject) gwPickupDateUS() uint64 {
	if o.Picker.Found {
		return o.Picker.TUS
	}
	return o.Bounds.HighUS
}
