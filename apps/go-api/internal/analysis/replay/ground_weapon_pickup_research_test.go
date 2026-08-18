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
//	     de l'item 1.3 ;
//	2.5  LE MEME ORACLE, MAIS PAR JOUEUR : le ramasseur passe par le pont slot -> xuid du
//	     CONSTRUCTEUR (`buildOwners`, `SlotXUID`) et l'on lit le loadout de sa VIE COURANTE a
//	     l'image-cle suivante — regles, seuil et temoin dans `ground_weapon_pickup_owner_test.go`.
//	     Il ajoute au film TROIS lectures (fil des morts, index de joueur, events de tir), qui
//	     sont exactement celles que `BuildFromFilm` fait pour construire ce pont.
//
// LECTURE SEULE, aucune base. UN SEUL decodage filmdec par process (`LockProcessDecode`),
// largeurs d'axe restaurees (`installWorldObjectPrecision`).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 GW_PICKUP=<repo>/data/cache/film_chunks/01e1f945 GW_PICKUP_MAP=Catalyst \
//	  go test ./internal/analysis/replay/ -run '^TestGroundWeaponPickups$' -timeout 30m -v

import (
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

// LE BORNAGE, LA DATATION ET L'ASSEMBLAGE SONT EN PRODUCTION depuis la phase 3
// (`ground_weapon_rules.go` et `ground_weapon_objects.go` : `gwPickupTrackTolUS`,
// `gwPickupStatus*`, `gwPickupObject`, `padObjects`, `gwPickupPadGaps`). Cet instrument
// les APPELLE : ce qu'il mesure est exactement ce que l'artefact publie, et le controle
// d'ancrage du plan n'aurait plus de sens contre une seconde chaine.

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
	keyframes     filmdec.WorldObjectKeyframes
	tracks        []filmdec.ProjectileTrack
	spans         map[filmdec.EquipmentLifeKey][]filmdec.EquipmentLifeSpan
	filmEndUS     uint64
	rng           *rand.Rand
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
	// L'ITEM 2.5 VIENT EN DERNIER, ET CE N'EST PAS UN DETAIL D'ORDRE : les items 2.1 et 2.2
	// consomment `f.rng`, et un rapport insere AVANT eux deplacerait leurs tirages, donc leurs
	// temoins. Le controle d'ancrage de ce lot (177 ramassages de socle, 62,7 % d'accord par
	// SLOT) doit rester reproductible au chiffre pres apres l'ajout de 2.5.
	gwPickupReport25(t, dir, f, objs)
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
		rng:       rand.New(rand.NewSource(gwPickupWitnessSeed)), //nolint:gosec // temoin reproductible
	}
	for _, p := range pos {
		f.bySlot[p.Slot] = append(f.bySlot[p.Slot], p)
		if p.TimestampUS > f.filmEndUS {
			f.filmEndUS = p.TimestampUS
		}
	}
	// Le recensement des images-cles vient de la PRODUCTION (`ScanFilmWorldObjectKeyframes`,
	// qui rend la bande de slots dans la MEME marche) : la mesure et l'artefact bornent avec le
	// meme recensement, ou l'ancrage ne prouverait rien.
	f.keyframes = filmdec.ScanFilmWorldObjectKeyframes(dir, filmdec.GroundWeaponTypeIndex)
	f.kfTimes, f.seen = f.keyframes.TimesUS, f.keyframes.SeenUS
	if n := len(f.kfTimes); n > 0 && f.kfTimes[n-1] > f.filmEndUS {
		f.filmEndUS = f.kfTimes[n-1]
	}
	f.loadouts = gwPickupLoadouts(t, dir)
	tracks, err := filmdec.ScanFilmWorldObjectsForBand(dir, wr, f.keyframes.Band)
	if err != nil {
		t.Fatalf("vies delta ti=42 : %v", err)
	}
	f.tracks = tracks
	f.spans = filmdec.EquipmentLifeSpans(tracks)
	t.Logf("LECTURE — %d positions de bipede · %d slots · %d vies · %d cles `ti=42` recensees"+
		" · %d cles a piste delta", len(pos), len(f.bySlot), gwPadsCountLives(f.lives),
		len(f.seen), len(f.spans))
	return f
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

// gwPickupObjects rend les apparitions retenues, bornees et datees — PAR LA CHAINE DE
// PRODUCTION (`padObjects`), celle-la meme que l'artefact de rejeu publie. Cet
// instrument n'ajoute que les denominateurs au journal.
func gwPickupObjects(
	t *testing.T, dir string, wr *filmdec.Vec3Range, f *gwPickupFilm,
) []gwPickupObject {
	t.Helper()
	cre, st, err := filmdec.ScanFilmGroundWeaponCreationsForBand(dir, wr, f.keyframes.Band)
	if err != nil {
		t.Fatalf("creations ti=42 : %v", err)
	}
	scan := WorldObjectScan{
		Scanned: true, Creations: cre, Stats: st, Keyframes: f.keyframes, Tracks: f.tracks,
	}
	out, _ := padObjects(scan, weaponPadRule(nil), f.lives, f.positions)
	if end := gwFilmEndUS(scan, f.positions); end > f.filmEndUS {
		f.filmEndUS = end
	}
	t.Logf("2.1 ENTREE — ancres %d · acceptees %d · RETENUES (identite croisee) %d · ecartees %d",
		st.Anchors, st.Accepted, len(out), st.Accepted-len(out))
	return out
}
