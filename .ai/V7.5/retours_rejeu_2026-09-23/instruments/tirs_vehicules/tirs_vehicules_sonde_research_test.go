//go:build research

package replay

// tirs_vehicules_sonde_research_test.go — SONDE DE L ENQUETE `tirs_vehicules` (2026-09-23).
//
// QUESTION : pourquoi les tirs du Ghost (et d autres armes de vehicule) ne sont-ils jamais
// publies ? LECTURE SEULE : au plus TROIS fichiers de faits persistes, aucun octet de film.
// L assemblage est rejoue PASSE PAR PASSE (meme ordre que `BuildFromPositions`) jusqu a la
// seconde porte des tirs, pour lire l etat interne : orphelins, pont slot -> index, episodes.
//
//	TV_ROOT=<depot principal> TV_FILMS=81c02726,8a485699,4f77afc1 \
//	  go test -tags research -count=1 -run TestSondeTirsVehicules -v \
//	  ./internal/games/halo_infinite/film/replay/

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/damagetag"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

const tvLoPerso = uint64(0x42C9679F)

// tvClasse range un WeaponID lu aux offsets FIXES du record 105/36.
func tvClasse(w uint64) string {
	lo := w & 0xFFFFFFFF
	switch {
	case w == 0:
		return "nul"
	case lo == tvLoPerso:
		return "perso"
	case lo == 0:
		return "vehicule"
	}
	for d := 1; d <= 20; d++ {
		if (w>>d)&0xFFFFFFFF == tvLoPerso {
			return fmt.Sprintf("perso_decale-%d", d)
		}
	}
	for s := 1; s <= 20; s++ {
		mask := uint64(1)<<(32-s) - 1
		if w&mask == tvLoPerso>>s {
			return fmt.Sprintf("perso_decale+%d", s)
		}
	}
	return "autre"
}

// tvEtat rend l issue de la PREMIERE porte (bipede) pour un evenement, par la regle de production.
func tvEtat(tracks map[uint32]slotTrack, owner map[uint32]int, e grammar.FireEvent) string {
	slot, reason := slotFor(tracks, owner, e.FilmIndex, e.TimestampUS)
	switch reason {
	case reasonNoSlot:
		return "sansSlot"
	case reasonAmbiguous:
		return "ambigu"
	}
	p, d := tracks[slot].at(e.TimestampUS)
	if d > shotPosToleranceUS || !p.HasWorld {
		return "horsFenetre"
	}
	return fmt.Sprintf("RATTACHE@%d", slot)
}

var tvVerdicts = map[vehicleShotVerdict]string{
	vehicleShotNoRide: "sansEpisode", vehicleShotAmbiguous: "ambigu",
	vehicleShotUnplaced: "sansPosition", vehicleShotPlaced: "POSE",
}

func TestSondeTirsVehicules(t *testing.T) {
	root, films := os.Getenv("TV_ROOT"), os.Getenv("TV_FILMS")
	if root == "" || films == "" {
		t.Skip("TV_ROOT / TV_FILMS non fournis")
	}
	ids := strings.Split(films, ",")
	if len(ids) > 3 {
		t.Fatalf("au plus 3 fichiers de faits (%d demandes)", len(ids))
	}
	cat, err := profile.LoadMapQuantCatalog(filepath.Join(root, "data", "titles", "halo_infinite",
		"reference", "map_quant_bounds.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		t.Run(id, func(t *testing.T) { tvSonder(t, root, cat, id) })
	}
}

func tvSonder(t *testing.T, root string, cat *profile.MapQuantCatalog, id string) {
	blob, err := os.ReadFile(filepath.Join(root, "data", "cache", "film_facts", "halo_infinite",
		id+".filmfacts.bin"))
	if err != nil {
		t.Fatal(err)
	}
	head, err := DecodeFilmFactsEntete(blob)
	if err != nil {
		t.Fatal(err)
	}
	var entry profile.MapQuantEntry
	trouve := false
	for _, e := range cat.Maps {
		if e.Module == head.MapModule {
			entry, trouve = e, true
			break
		}
	}
	if !trouve {
		t.Fatalf("module %q absent du catalogue", head.MapModule)
	}
	f, err := DecodeFilmFactsFile(blob, entry)
	if err != nil {
		t.Fatal(err)
	}
	opt := Options{MapQuant: &entry, Labels: goldenCatalog(t)}
	opt.Fallbacks = fallback.NouveauCompteur()
	opt.Fallbacks.Cumuler(f.Fallbacks)
	opt.FilmIdentity = f.Identity
	in := f.Facts.FilmInputs
	in.applyTo(&opt)

	a := &assemblage{matchID: id, opt: opt, pos: in.Positions, fire: in.Fire}
	if !a.ouvrir("halo_infinite") {
		t.Fatal("aucune position")
	}
	a.poserLesPistes()
	a.poserLesEquipesEtLeRoster()
	a.poserTirsProjectilesEtGrenades()
	a.poserScoreEtObjectifs()
	a.poserEpisodesDEquipement()
	a.composerLaCouverture()
	a.poserGrappinEtPoses()
	a.poserPrisesEtSocles()
	a.poserArmesAuSolEtVehicules()

	tvRapport(t, id, head.MapModule, a, f, in)
}

func tvRapport(t *testing.T, id, module string, a *assemblage, f *FilmFactsFile, in FilmInputs) {
	clock := replayClock{origin: a.origin, step: a.step, frames: a.doc.FrameCount}
	tracks := indexBySlot(a.sorted)
	owner := a.reg.IndexParSlot()
	slotsOf := vehicleSlotsByPlayer(owner)
	cs, cv := a.doc.Coverage.Shots, a.doc.Coverage.Vehicles
	t.Logf("=== %s module=%s roster=%d tirs=%d frames=%d origineFilmUS=%d origineUS=%d",
		id, module, len(a.doc.Roster), len(a.fire), a.doc.FrameCount, in.FilmClockOriginUS, a.origin)
	t.Logf("COUV tirs %+v", cs)
	if cv != nil {
		t.Logf("COUV veh poses=%d sansEpisode=%d ambigus=%d sansPosition=%d armeVeh=%d episodes=%d",
			cv.Shots, cv.ShotsNoRide, cv.ShotsAmbiguous, cv.ShotsUnplaced, cv.ShotsVehicleWeapon, cv.Rides)
	}
	nomDe := map[int]string{}
	for _, r := range a.doc.Roster {
		nomDe[r.FilmIndex] = r.Name
		t.Logf("ROSTER idx=%2d (4b=%2d) %-20s slots=%v", r.FilmIndex, r.FilmIndex&0xF, r.Name, slotsOf[r.FilmIndex])
	}
	tvParIndex(t, a, tracks, owner)
	tvClasses(t, a)
	tvEpisodes(t, a, tracks, owner, nomDe, clock)
	tvOrphelins(t, a, slotsOf, clock)
	tvKills(t, a, f, in, tracks, owner, clock)
}

// tvParIndex : evenements par FilmIndex lu (0..15) et par classe, avec l issue de la porte 1.
func tvParIndex(t *testing.T, a *assemblage, tracks map[uint32]slotTrack, owner map[uint32]int) {
	type agg struct {
		n       int
		classes map[string]int
		etats   map[string]int
	}
	par := map[int]*agg{}
	for _, e := range a.fire {
		g := par[e.FilmIndex]
		if g == nil {
			g = &agg{classes: map[string]int{}, etats: map[string]int{}}
			par[e.FilmIndex] = g
		}
		g.n++
		g.classes[tvClasse(e.WeaponID)]++
		etat := tvEtat(tracks, owner, e)
		if strings.HasPrefix(etat, "RATTACHE") {
			etat = "rattache"
		}
		g.etats[etat]++
	}
	idx := make([]int, 0, len(par))
	for k := range par {
		idx = append(idx, k)
	}
	sort.Ints(idx)
	for _, k := range idx {
		t.Logf("INDEX %2d : %5d evts classes=%v porte1=%v", k, par[k].n, par[k].classes, par[k].etats)
	}
}

// tvClasses : pour chaque classe hors perso/vehicule, les WeaponID les plus frequents.
func tvClasses(t *testing.T, a *assemblage) {
	parW := map[uint64]map[int]int{}
	for _, e := range a.fire {
		c := tvClasse(e.WeaponID)
		if c == "perso" {
			continue
		}
		if parW[e.WeaponID] == nil {
			parW[e.WeaponID] = map[int]int{}
		}
		parW[e.WeaponID][e.FilmIndex]++
	}
	ws := make([]uint64, 0, len(parW))
	for w := range parW {
		ws = append(ws, w)
	}
	total := func(w uint64) int {
		n := 0
		for _, v := range parW[w] {
			n += v
		}
		return n
	}
	sort.Slice(ws, func(i, j int) bool { return total(ws[i]) > total(ws[j]) })
	for i, w := range ws {
		if i >= 40 {
			t.Logf("CLASSES ... %d autres identifiants", len(ws)-40)
			break
		}
		t.Logf("ARME %s classe=%-18s n=%4d parIndex=%v", formatWeaponID(w), tvClasse(w), total(w), parW[w])
	}
}

// tvEpisodes : chaque episode publie, et les evenements de son occupant pendant l episode.
func tvEpisodes(t *testing.T, a *assemblage, tracks map[uint32]slotTrack, owner map[uint32]int,
	nomDe map[int]string, clock replayClock,
) {
	for _, v := range a.doc.Vehicles {
		for _, r := range v.Rides {
			pi, ok := owner[r.Slot]
			if !ok {
				pi = -1
			}
			seat := "nul"
			if r.Seat != nil {
				seat = fmt.Sprint(*r.Seat)
			}
			t.Logf("EPISODE veh=%d/%s/%s occ=%d idx=%d(%s) t=%d-%d (%.1fs) src=%s siege=%s",
				v.Slot, v.Family, v.Chassis, r.Slot, pi, nomDe[pi], r.T0, r.T1,
				float64(r.T1-r.T0)/10, r.Src, seat)
			classes, etats := map[string]int{}, map[string]int{}
			var lignes, autres []string
			for _, e := range a.fire {
				fr := clock.frame(e.TimestampUS)
				if fr < r.T0-10 || fr > r.T1+10 {
					continue
				}
				etat := tvEtat(tracks, owner, e)
				if e.FilmIndex == pi&0xF {
					c := tvClasse(e.WeaponID)
					classes[c]++
					k := etat
					if strings.HasPrefix(k, "RATTACHE") {
						k = "rattache"
					}
					etats[k]++
					if len(lignes) < 25 {
						lignes = append(lignes, fmt.Sprintf("f=%d %s %s %s", fr, formatWeaponID(e.WeaponID), c, etat))
					}
					continue
				}
				if c := tvClasse(e.WeaponID); c != "perso" && len(autres) < 25 {
					autres = append(autres, fmt.Sprintf("f=%d idx=%d %s %s %s", fr, e.FilmIndex,
						formatWeaponID(e.WeaponID), c, etat))
				}
			}
			t.Logf("   occupant (idx&0xF=%d) : classes=%v porte1=%v", pi&0xF, classes, etats)
			for _, l := range lignes {
				t.Logf("     %s", l)
			}
			for _, l := range autres {
				t.Logf("     AUTRE-TIREUR %s", l)
			}
		}
	}
}

// tvOrphelins : chaque orphelin de la porte 1, son verdict a la porte 2, et les episodes de son
// tireur a +/-5 s.
func tvOrphelins(t *testing.T, a *assemblage, slotsOf map[int][]uint32, clock replayClock) {
	rides := vehicleRidesByOccupant(a.doc.Vehicles)
	verdicts, classes := map[string]int{}, map[string]int{}
	for i, o := range a.shotOrphans {
		_, v := vehicleShotOf(o, rides, slotsOf[o.ev.FilmIndex], a.doc.Vehicles, clock)
		verdicts[tvVerdicts[v]]++
		c := tvClasse(o.ev.WeaponID)
		classes[c]++
		if i >= 400 {
			continue
		}
		fr := clock.frame(o.ev.TimestampUS)
		var proches []string
		for _, s := range slotsOf[o.ev.FilmIndex] {
			for _, rr := range rides[s] {
				if fr >= rr.ride.T0-50 && fr <= rr.ride.T1+50 {
					tr := a.doc.Vehicles[rr.track]
					proches = append(proches, fmt.Sprintf("%s[%d-%d]", tr.Family, rr.ride.T0, rr.ride.T1))
				}
			}
		}
		t.Logf("ORPHELIN f=%d idx=%d %s %s raison=%d porte2=%s episodes+-5s=%v", fr, o.ev.FilmIndex,
			formatWeaponID(o.ev.WeaponID), c, o.reason, tvVerdicts[v], proches)
	}
	t.Logf("ORPHELINS %d : porte2=%v classes=%v", len(a.shotOrphans), verdicts, classes)
}

// tvKills : les morts de classe VEHICULE lues par la killsource, et les tirs a +/-2 s.
func tvKills(t *testing.T, a *assemblage, f *FilmFactsFile, in FilmInputs,
	tracks map[uint32]slotTrack, owner map[uint32]int, clock replayClock,
) {
	if f.Kills == nil {
		t.Logf("KILLS : section absente")
		return
	}
	idxDe := map[string]int{}
	for _, r := range a.doc.Roster {
		idxDe[r.Name] = r.FilmIndex
	}
	n := 0
	for _, k := range f.Kills.Kills {
		if k.Source.Class != damagetag.ClassVehicule {
			continue
		}
		n++
		tUS := in.FilmClockOriginUS + uint64(k.TimeMS)*1000
		fr := clock.frame(tUS)
		ki, ok := idxDe[k.Feed.Killer]
		if !ok {
			ki = -1
		}
		t.Logf("KILL-VEH t=%dms f=%d tueur=%s(idx=%d) victime=%s tag=%08x detail=%s", k.TimeMS, fr,
			k.Feed.Killer, ki, k.Victim, k.Source.Tag, k.Source.Detail)
		for _, e := range a.fire {
			fe := clock.frame(e.TimestampUS)
			if fe < fr-20 || fe > fr+5 {
				continue
			}
			marque := ""
			if e.FilmIndex == ki&0xF {
				marque = " <-TUEUR"
			}
			t.Logf("     f=%d idx=%d %s %s %s%s", fe, e.FilmIndex, formatWeaponID(e.WeaponID),
				tvClasse(e.WeaponID), tvEtat(tracks, owner, e), marque)
		}
	}
	t.Logf("KILLS de classe VEHICULE : %d / %d", n, len(f.Kills.Kills))
}
