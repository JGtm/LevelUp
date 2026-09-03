package killsource

// vehicules_v10_deadstate_test.go — INSTRUMENT DE MESURE (lot V10) : le dead-state `i11` des
// VEHICULES (`ti=40`). LECTURE SEULE, garde par V10_FILMS. Il n'ajoute AUCUNE grammaire : il
// reutilise la marche de records de ce paquet, telle qu'elle est, et se contente de ne PAS jeter
// les dead-states dont le slot sort de la plage bipede.
//
// POURQUOI ICI ET PAS DANS `filmdec`. Le balayage ANCRE de `filmdec` (matchBipedHeader) n'accepte
// qu'un record dont le masque commence par i0 ET dont i0 est ABSOLU. Recensement V10 sur deux
// films : sur la bande `ti=40`, 0 record sur 37 677 declare i11 ; mais sur la bande BIPEDE du
// MEME film — ou les morts sont un fait etabli — il en voit 1 sur 207 808 et 0 sur 193 762.
// L'ancre est donc AVEUGLE au dead-state, pour tout le monde : « 0 record ti=40 porte i11 » n'y
// dit rien du vehicule. La MARCHE de `killsource` (DecodeFrameRecords + World) est la seule voie
// du depot qui atteint reellement i11 — c'est elle qui sert le kill-feed.
//
// CE QUE GHIDRA A ETABLI AVANT LA MESURE (2026-09-03, chaine statique du lot V9) :
//
//	`object-dead-state-component` @0x143c99320 est l'UNIQUE chaine du binaire pour ce composant
//	    (aucun `-dynamic-precision-`, donc aucun piege equivalent a celui d'i2/i3) ;
//	xref DATA unique -> thunk `name()` @0x14064c6d0 -> slot de vtable @0x143d0ba48
//	    -> vtable @0x143d0ba40 ; vtable[0x28] = 0x14076ce9c (LE thunk) -> deser = vtable[0x30]
//	    = **0x140c1dce0** ;
//	FUN_140c1dce0 : mort = R(1) ; si ti in {0x23, 0x28} -> FUN_140c1dd44 ; si ti == 0x23 -> R(1).
//
// `ti=40` = 0x28 : la forme est celle du bipede MOINS le bit de queue, et FUN_140c1dd44 ne recoit
// PAS le typeIndex — c'est le MEME corps lourd que celui qui sert le kill-feed. Le depot le
// portait deja (`consumeObjectDeadStateBipedTI`, branche `typeIndex == 0x28`).
//
// USAGE :
//
//	CGO_ENABLED=0 V10_FILM_ROOT=<repo>/data/cache V10_FILMS="0d76e8f1,fccc61cd" \
//	  go test ./internal/games/halo_infinite/film/killsource/ -run '^TestV10DeadStateVehicules$' -v

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// vehicleTypeIndex : l'archetype VEHICULE du registre film.
const v10VehicleTI = 40

// v10Dead : un dead-state atteint par la marche, sans aucun filtre de plage.
//
// `us` est l HORODATAGE ABSOLU du paquet porteur — le MEME champ d en-tete que
// `filmdec.BipedPosition.TimestampUS`. Il est indispensable : `ms` est relatif au premier paquet
// type-0 du film, donc incomparable aux fenetres de vie du calque vehicule, qui sont absolues.
// C est ce qui permet de RECOUPER un dead-state avec la derniere lecture de vitalite d une vie.
type v10Dead struct {
	ms   int
	us   uint64
	slot int
	ti   uint32
	dead filmdec.DeadState
}

// v10Couverture compte, par archetype, les records PROPRES que la marche atteint — le
// denominateur SANS LEQUEL « 0 dead-state ti=40 » ne veut rien dire. Il separe « le vehicule
// n'emet pas i11 » de « la marche ne voit pas les records de vehicule ».
type v10Couverture struct {
	recs    map[uint32]int // records propres par archetype
	withI11 map[uint32]int // dont le masque a fait lire un dead-state (Mort quelconque)
}

func TestV10DeadStateVehicules(t *testing.T) {
	films := v10Films(t)
	root := v10Root()
	release := filmdec.LockProcessDecode()
	defer release()

	for _, short8 := range films {
		dir := filepath.Join(root, "film_chunks", short8)
		t.Run(short8, func(t *testing.T) { v10RunFilm(t, dir, short8) })
	}
}

func v10RunFilm(t *testing.T, dir, short8 string) {
	src, err := DirChunks(dir)
	if err != nil {
		t.Fatalf("%s : chunks illisibles : %v", short8, err)
	}
	f, err := loadFilm(src)
	if err != nil {
		t.Fatalf("%s : film illisible : %v", short8, err)
	}
	tl, err := newTimeline(f)
	if err != nil {
		t.Fatalf("%s : timeline : %v", short8, err)
	}
	kf := filmdec.ScanFilmWorldObjectKeyframes(dir, v10VehicleTI)
	if len(kf.Band) == 0 {
		t.Fatalf("%s : aucun slot ti=%d aux images-cles", short8, v10VehicleTI)
	}
	bipLo, bipHi := tl.bipedRange()

	deads, cov := v10Harvest(f, tl)
	byTI := map[uint32]int{}
	var vehDeads []v10Dead
	nBiped := 0
	for _, d := range deads {
		byTI[d.ti]++
		if d.ti == v10VehicleTI {
			vehDeads = append(vehDeads, d)
		}
		if d.slot >= bipLo && d.slot <= bipHi {
			nBiped++
		}
	}
	tis := make([]int, 0, len(byTI))
	for k := range byTI {
		tis = append(tis, int(k))
	}
	sort.Ints(tis)
	var parts []string
	for _, k := range tis {
		parts = append(parts, fmt.Sprintf("ti%d:%d", k, byTI[uint32(k)]))
	}
	t.Logf("V10 %s — dead-states Mort=1 atteints par la marche : %d au total · plage bipede [%d..%d] : %d · par archetype : %s",
		short8, len(deads), bipLo, bipHi, nBiped, strings.Join(parts, " "))

	// LE POINT DE LA MESURE : combien de dead-states tombent sur la bande `ti=40`, et sur
	// combien de VIES distinctes ? Le recensement des images-cles borne les vies.
	inBand, lives := 0, map[uint32]bool{}
	for _, d := range vehDeads {
		if kf.Band[uint32(d.slot)] {
			inBand++
			lives[uint32(d.slot)] = true
		}
	}
	t.Logf("V10 %s — dead-states d'archetype ti=40 : %d (dont %d sur un slot de la bande, %d slots distincts) · vies recensees ti=40 : %d",
		short8, len(vehDeads), inBand, len(lives), len(kf.SeenUS))
	// LE DENOMINATEUR : la marche voit-elle des records de vehicule, et y lit-elle i11 ?
	covTis := make([]int, 0, len(cov.recs))
	for k := range cov.recs {
		covTis = append(covTis, int(k))
	}
	sort.Ints(covTis)
	var covParts []string
	for _, k := range covTis {
		covParts = append(covParts, fmt.Sprintf("ti%d:%d(i11=%d)", k, cov.recs[uint32(k)],
			cov.withI11[uint32(k)]))
	}
	t.Logf("V10 %s — COUVERTURE de la marche (records PROPRES par archetype, et combien ont fait lire i11) : %s",
		short8, strings.Join(covParts, " "))
	for i, d := range vehDeads {
		if i >= 30 {
			t.Logf("  ... et %d autres", len(vehDeads)-30)
			break
		}
		t.Logf("  ti=40 dead-state @%d ms (ABSOLU %.1f s) slot=%d victime=%d tueur=%d cat=%d tag=%08x bande=%v",
			d.ms, float64(d.us)/1e6, d.slot, d.dead.EnumA, d.dead.EnumB, d.dead.Val0c,
			d.dead.SrcTag0, kf.Band[uint32(d.slot)])
	}
}

// v10Harvest deroule la MEME marche que `runWalk`, sans le filtre de credibilite : tous les
// dead-states `Mort=1` de records PROPRES, quel que soit leur slot et leur archetype. Il rend en
// plus la COUVERTURE de la marche par archetype — le denominateur de la mesure.
func v10Harvest(f *film, tl *timeline) ([]v10Dead, v10Couverture) {
	tl.rewind()
	cfg := filmdec.DefaultFrameConfig()
	views := v10Views()
	cov := v10Couverture{recs: map[uint32]int{}, withI11: map[uint32]int{}}
	var out []v10Dead
	for i := range f.t0 {
		p := &f.t0[i]
		w := tl.advanceTo(p.ts)
		start := 2
		if hasEvents(p) {
			s := locateRecords(p.payload, w, cfg)
			if s < 0 {
				continue
			}
			start = s
		}
		snap := w.Snapshot()
		recs := walkFrom(p.payload, w, cfg, start, views)
		for j := range recs {
			r := &recs[j]
			if r.DesyncAt != -1 {
				continue
			}
			ti, _ := w.ArchetypeForSlot(r.Slot)
			cov.recs[ti]++
			if r.Trace.Dead != nil {
				cov.withI11[ti]++
			}
			if r.Trace.Dead == nil || !r.Trace.Dead.Mort {
				continue
			}
			out = append(out, v10Dead{ms: f.ms(p), us: p.ts, slot: int(r.Slot), ti: ti,
				dead: *r.Trace.Dead})
		}
		w.Restore(snap)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ms < out[j].ms })
	return out, cov
}

// v10Views rend la PROFONDEUR de marche (vues de replication par paquet). Le defaut de
// production est 8 ; `V10_VIEWS` permet de MESURER si la rarete d'i11 sur `ti=40` est un fait du
// film ou un effet de bord de cette profondeur — la question ne se tranche pas au raisonnement.
func v10Views() int {
	if v := os.Getenv("V10_VIEWS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return DefaultOptions().Views
}

func v10Films(t *testing.T) []string {
	raw := os.Getenv("V10_FILMS")
	if raw == "" {
		t.Skipf("V10_FILMS absent : instrument dead-state vehicule saute")
	}
	var out []string
	for _, tok := range strings.Split(raw, ",") {
		tok = strings.TrimSpace(tok)
		if i := strings.Index(tok, ":"); i >= 0 {
			tok = strings.TrimSpace(tok[:i]) // tolere la forme "short8:carte" des autres instruments
		}
		if tok != "" {
			out = append(out, tok)
		}
	}
	if len(out) == 0 {
		t.Skipf("V10_FILMS vide")
	}
	return out
}

func v10Root() string {
	if r := os.Getenv("V10_FILM_ROOT"); r != "" {
		return r
	}
	return `C:\Users\Guillaume\Projects\LevelUp\data\cache`
}
