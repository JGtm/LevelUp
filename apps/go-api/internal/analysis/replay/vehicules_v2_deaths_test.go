package replay

// vehicules_v2_deaths_test.go — INSTRUMENT DE MESURE : DATER la destruction d'un vehicule par la
// MORT de son occupant (volet mort-coincidente de l'item 4 du lot V2). LECTURE SEULE, garde par
// V2D_FILMS.
//
// L'IDEE (insight utilisateur). Le recensement des images-cles borne la fin de vie d'un vehicule a
// +/-20 s, jamais datee. Mais si un vehicule est DETRUIT, son occupant meurt en general aussi — et
// le fil des morts (ScanFilmDeaths) date chaque mort A LA MILLISECONDE sur l'horloge du match. Donc
// une mort proche (dans le temps ET dans l'espace) de la fin de vie d'un vehicule DATE sa
// destruction, bien mieux que la borne du recensement.
//
// CE QUE CET INSTRUMENT REUTILISE, sans rien recopier :
//   - positions JOUEUR : filmdec.ScanFilmBipedPositions (chemin de production, monde en metres) ;
//   - fil des morts : ScanFilmDeaths ; index joueur : ScanFilmPlayerIndices + injectiveOrEmpty ;
//   - PONT slot->xuid + CALAGE d'horloge : buildOwners (own.SlotXUID, own.DeathOffsetMS). Le calage
//     est celui, PROUVE, du pont de production : horlogeFilm_ms = death.TimeMS + DeathOffsetMS ;
//   - vies + trajectoires VEHICULE : filmdec.ScanFilmWorldObjectKeyframes (recensement, bornes de
//     vie) et filmdec.ScanFilmBipedPositionsForBand (grammaire dyn.-prec., monde en metres).
//
// LA MESURE, LES SEUILS ECRITS AVANT LE CODE.
//   - fin de vie SERREE : dernier echantillon de trajectoire du slot vehicule dans [firstSeen,goneBy]
//     (resolution ~0,5 s), et sa POSITION. A defaut, la borne du recensement (lastSeen).
//   - COINCIDENCE TEMPORELLE : une mort a moins de +/-2 s de la fin serree (horloges calees).
//   - COINCIDENCE SPATIALE : la position de la VICTIME a l'instant de sa mort (echantillon joueur le
//     plus proche, ecart < 2 s) est a moins de 5 m de la derniere position du vehicule.
//   - CLASSEMENT : DETRUIT (coincidence temporelle) / dont DETRUIT+conducteur (temporelle ET
//     spatiale) / DESPAWN (aucune mort coincidente : disparition sans destruction datee).
//   - TEMOIN : le meme test au calage DECALE de 37 s (decorrelation). La part DETRUIT reelle doit
//     depasser nettement la part au temoin, sinon la coincidence n'est que le hasard des morts
//     denses. (Un decalage constant, pas un decalage d'un cran : les fins de vie sont autocorrelees.)
//
// RECOUPEMENT (signale, non conclu ici) : l'occupant qui meurt a la fin de vie du vehicule EST
// probablement le CONDUCTEUR — signal complementaire du « debut de trou » (V1a.4).
//
// UN SEUL decodage filmdec par process : le verrou est pris pour tout le test.
//
// USAGE (depuis apps/go-api, cache Go ISOLE) :
//
//	$env:GOCACHE='<scratch>\gocache_v2'
//	CGO_ENABLED=0 V2D_FILM_ROOT=<repo>/data/cache \
//	  V2D_FILMS="0d76e8f1:behemoth,fccc61cd:launch site" \
//	  V2D_BOUNDS=<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  go test ./internal/analysis/replay/ -run '^TestV2VehicleDeathDating$' -v -timeout 180m

import (
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// SEUILS, ecrits AVANT toute mesure.
const (
	v2dTimeWinMS      = 2000  // +/-2 s : coincidence temporelle
	v2dSpaceMaxM      = 5.0   // < 5 m : coincidence spatiale (victime vs derniere pos vehicule)
	v2dVictimGapMS    = 2000  // ecart max echantillon joueur <-> instant de mort pour situer la victime
	v2dWitnessShiftMS = 37000 // decalage du temoin (decorrelation, non resonant)
)

type v2dFilmSpec struct{ short8, mapKey string }

// v2dMapAgg agrege un carte.
type v2dMapAgg struct {
	films                                   int
	lives, withEnd, withPos                 int
	detruitTemporal, detruitSpatio, despawn int
	witnessTemporal                         int
	deaths, bridgeMatched                   int
	vNoSlot, vGapTooBig, vSampled           int
	dists                                   []float64
}

func TestV2VehicleDeathDating(t *testing.T) {
	films := v2dParseFilms(t)
	boundsCat := v2dLoadBounds(t)

	release := filmdec.LockProcessDecode()
	defer release()

	aggs := map[string]*v2dMapAgg{}
	for _, f := range films {
		entry, err := boundsCat.Lookup(f.mapKey)
		if err != nil {
			t.Fatalf("%s : bornes de %q introuvables : %v", f.short8, f.mapKey, err)
		}
		ag := aggs[f.mapKey]
		if ag == nil {
			ag = &v2dMapAgg{}
			aggs[f.mapKey] = ag
		}
		dir := v2dDir(f.short8)
		t.Run(f.short8, func(t *testing.T) { v2dProcessFilm(t, dir, entry, ag) })
	}

	keys := make([]string, 0, len(aggs))
	for k := range aggs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v2dReport(t, k, aggs[k])
	}
}

func v2dProcessFilm(t *testing.T, dir string, entry filmdec.MapQuantEntry, ag *v2dMapAgg) {
	worldRange := entry.Range()
	lay := entry.Layout()

	ppos, err := v2dPlayerPositions(dir, worldRange, lay)
	if err != nil {
		t.Fatalf("positions joueur : %v", err)
	}
	tracks := indexBySlot(ppos)
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Fatalf("fil des morts : %v", err)
	}
	idx, err := ScanFilmPlayerIndices(dir, rosterFromDeaths(deaths))
	if err != nil {
		t.Logf("index joueur illisible (%v) — pont sans identite", err)
	}
	table, _ := injectiveOrEmpty(idx)
	own := buildOwners(tracks, deaths, table, nil)
	xuidSlots := v2dInvertSlotXUID(own.SlotXUID)

	kf := filmdec.ScanFilmWorldObjectKeyframes(dir, filmdec.VehicleTypeIndex)
	vtracks := v2dVehicleTracks(dir, kf.Band, worldRange, lay)

	fr := v2dScoreFilm(kf, vtracks, tracks, deaths, xuidSlots, own.DeathOffsetMS)
	ag.films++
	ag.lives += fr.lives
	ag.withEnd += fr.withEnd
	ag.withPos += fr.withPos
	ag.detruitTemporal += fr.detruitTemporal
	ag.detruitSpatio += fr.detruitSpatio
	ag.despawn += fr.despawn
	ag.witnessTemporal += fr.witnessTemporal
	ag.deaths += len(deaths)
	ag.bridgeMatched += own.DeathOffsetMatches
	ag.vNoSlot += fr.vNoSlot
	ag.vGapTooBig += fr.vGapTooBig
	ag.vSampled += fr.vSampled
	ag.dists = append(ag.dists, fr.dists...)

	t.Logf("V2D %s — offset %d ms (%d/%d vies joueur calees) · %d morts · vies veh %d (fin serree %d, avec pos %d) · DETRUIT temp %d / spatio %d · DESPAWN %d · temoin temp %d",
		shortOf(dir), own.DeathOffsetMS, own.DeathOffsetMatches, own.LivesTotal,
		len(deaths), fr.lives, fr.withEnd, fr.withPos,
		fr.detruitTemporal, fr.detruitSpatio, fr.despawn, fr.witnessTemporal)
}

type v2dFilmResult struct {
	lives, withEnd, withPos                 int
	detruitTemporal, detruitSpatio, despawn int
	witnessTemporal                         int
	// diagnostics du critere spatial, sur les vies DETRUIT (temporelles) avec position vehicule.
	vNoSlot    int       // xuid de la victime absent du pont slot->xuid
	vGapTooBig int       // slot trouve mais aucun echantillon joueur a moins de v2dVictimGapMS
	vSampled   int       // position de la victime obtenue
	dists      []float64 // distances victime<->vehicule obtenues (metres)
}

// v2dScoreFilm classe chaque vie de vehicule : DETRUIT (mort coincidente) vs DESPAWN.
func v2dScoreFilm(kf filmdec.WorldObjectKeyframes, vtracks, ptracks map[uint32]slotTrack,
	deaths []Death, xuidSlots map[uint64][]uint32, off int64) v2dFilmResult {
	var fr v2dFilmResult
	for key, seen := range kf.SeenUS {
		if len(seen) == 0 {
			continue
		}
		fr.lives++
		firstSeen, lastSeen := seen[0], seen[len(seen)-1]
		goneBy := lastSeen
		for _, ts := range kf.TimesUS {
			if ts > lastSeen {
				goneBy = ts
				break
			}
		}
		endUS, endPos, hasPos := v2dTightEnd(vtracks[key.Slot], firstSeen, goneBy)
		if endUS == 0 {
			endUS = lastSeen // borne du recensement a defaut de trajectoire
		} else {
			fr.withEnd++
		}
		if hasPos {
			fr.withPos++
		}
		endMS := int64(endUS / 1000)
		// coincidence temporelle (calage reel) et temoin (calage decale)
		d, ok := v2dNearestDeath(deaths, off, endMS, v2dTimeWinMS)
		_, wok := v2dNearestDeath(deaths, off+v2dWitnessShiftMS, endMS, v2dTimeWinMS)
		if wok {
			fr.witnessTemporal++
		}
		if !ok {
			fr.despawn++
			continue
		}
		fr.detruitTemporal++
		if hasPos {
			dist, hasSample, hasSlot := v2dVictimProbe(d, off, ptracks, xuidSlots, endPos)
			switch {
			case !hasSlot:
				fr.vNoSlot++
			case !hasSample:
				fr.vGapTooBig++
			default:
				fr.vSampled++
				fr.dists = append(fr.dists, dist)
				if dist < v2dSpaceMaxM {
					fr.detruitSpatio++
				}
			}
		}
	}
	return fr
}

// v2dTightEnd rend le dernier echantillon de trajectoire du slot dans [firstSeen, goneBy] : instant
// (us) et position (metres). endUS==0 : aucun echantillon dans la fenetre.
func v2dTightEnd(vt slotTrack, firstSeen, goneBy uint64) (uint64, [3]float64, bool) {
	var endUS uint64
	var pos [3]float64
	var has bool
	for _, p := range vt.pts {
		if p.TimestampUS < firstSeen || p.TimestampUS > goneBy {
			continue
		}
		if p.TimestampUS >= endUS {
			endUS = p.TimestampUS
			if p.HasWorld {
				pos, has = [3]float64{float64(p.X), float64(p.Y), float64(p.Z)}, true
			}
		}
	}
	return endUS, pos, has
}

// v2dNearestDeath rend la mort la plus proche (horloge film) de endMS dans la fenetre.
func v2dNearestDeath(deaths []Death, off, endMS, winMS int64) (Death, bool) {
	best, bd, ok := Death{}, winMS+1, false
	for _, d := range deaths {
		if delta := absI64(d.TimeMS + off - endMS); delta <= winMS && delta < bd {
			bd, best, ok = delta, d, true
		}
	}
	return best, ok
}

// v2dVictimProbe situe la victime a l'instant de sa mort (echantillon joueur le plus proche) et
// rend la distance a la derniere position du vehicule. hasSlot : la victime est nommee dans le pont ;
// hasSample : un echantillon joueur existe a moins de v2dVictimGapMS de la mort. Un occupant EMBARQUE
// cesse de repliquer sa position (V1a.4) : hasSample faux est alors le signal, pas un echec.
func v2dVictimProbe(d Death, off int64, ptracks map[uint32]slotTrack,
	xuidSlots map[uint64][]uint32, vehPos [3]float64) (dist float64, hasSample, hasSlot bool) {
	slots := xuidSlots[d.XUID]
	if len(slots) == 0 {
		return 0, false, false
	}
	deathUS := uint64(int64(d.TimeMS+off) * 1000)
	bestGap := uint64(v2dVictimGapMS) * 1000
	var vp [3]float64
	for _, s := range slots {
		p, gap := ptracks[s].at(deathUS)
		if gap <= bestGap && p.HasWorld {
			bestGap, vp, hasSample = gap, [3]float64{float64(p.X), float64(p.Y), float64(p.Z)}, true
		}
	}
	if !hasSample {
		return 0, false, true
	}
	return v2dDist(vp, vehPos), true, true
}

func v2dReport(t *testing.T, mapKey string, ag *v2dMapAgg) {
	t.Logf("\n############## MORT-COINCIDENTE — CARTE %q (%d films) ##############", mapKey, ag.films)
	t.Logf("  morts totales %d · calage pont %d vies joueur · vies vehicule %d (fin serree %d, avec pos %d)",
		ag.deaths, ag.bridgeMatched, ag.lives, ag.withEnd, ag.withPos)
	if ag.lives == 0 {
		return
	}
	ft := v2dFrac(ag.detruitTemporal, ag.lives)
	fw := v2dFrac(ag.witnessTemporal, ag.lives)
	t.Logf("  DETRUIT (coincidence temporelle +/-%d ms) : %d/%d = %.0f %% · TEMOIN (decale %d ms) : %d/%d = %.0f %%",
		v2dTimeWinMS, ag.detruitTemporal, ag.lives, 100*ft, v2dWitnessShiftMS, ag.witnessTemporal, ag.lives, 100*fw)
	t.Logf("  au-dessus du hasard : %s (reel %.0f %% vs temoin %.0f %%)", v2dVerdict(ft > fw+0.10), 100*ft, 100*fw)
	t.Logf("  DESPAWN (aucune mort coincidente) : %d/%d = %.0f %%", ag.despawn, ag.lives, 100*v2dFrac(ag.despawn, ag.lives))
	// diagnostic du critere spatial sur les vies DETRUIT temporelles avec position vehicule.
	t.Logf("  CRITERE SPATIAL (sur DETRUIT temporel) : victime non mappee %d · victime SANS echantillon a +/-%d ms %d · victime situee %d",
		ag.vNoSlot, v2dVictimGapMS, ag.vGapTooBig, ag.vSampled)
	if ag.vSampled > 0 {
		sort.Float64s(ag.dists)
		t.Logf("     distances victime<->vehicule (situees) : mediane %.1f m · min %.1f m · < %.0f m : %d/%d",
			ag.dists[len(ag.dists)/2], ag.dists[0], v2dSpaceMaxM, ag.detruitSpatio, ag.vSampled)
	}
	t.Logf("  LECTURE : un occupant EMBARQUE cesse de repliquer sa position (V1a.4) ; une victime coincidente")
	t.Logf("  SANS echantillon proche est donc le signal ATTENDU d'un conducteur (mort dans le trou de position),")
	t.Logf("  pas un echec de mesure. Death ne porte pas de position ; le critere spatial passe par la victime.")
}

// ------------------------------------------------------------------ helpers

func v2dPlayerPositions(dir string, wr filmdec.Vec3Range, lay filmdec.I0Layout) ([]filmdec.BipedPosition, error) {
	scan := filmdec.DefaultScanFilmOptions()
	scan.WorldRange = &wr
	if lay.Valid() {
		scan.Layout = &lay
	}
	scan.CaptureDirs = true
	return filmdec.ScanFilmBipedPositions(dir, scan)
}

func v2dVehicleTracks(dir string, band map[uint32]bool, wr filmdec.Vec3Range, lay filmdec.I0Layout) map[uint32]slotTrack {
	opt := filmdec.ScanFilmOptions{WorldRange: &wr, RequireTag1: false, DropSaturated: true}
	if lay.Valid() {
		opt.Layout = &lay
	}
	pos, err := filmdec.ScanFilmBipedPositionsForBand(dir, filmdec.NewSlotBand(band), opt)
	if err != nil {
		return map[uint32]slotTrack{}
	}
	return indexBySlot(pos)
}

func v2dInvertSlotXUID(slotXUID map[uint32]uint64) map[uint64][]uint32 {
	out := map[uint64][]uint32{}
	for s, x := range slotXUID {
		out[x] = append(out[x], s)
	}
	return out
}

// v2dDist : adaptateur d'une ligne vers dist3 (l'unique formule du paquet, geometry.go). Les
// positions viennent de float32 (monde film) ; la reconversion est sans perte a l'echelle du match.
func v2dDist(a, b [3]float64) float64 {
	return dist3([3]float32{float32(a[0]), float32(a[1]), float32(a[2])}, [3]float32{float32(b[0]), float32(b[1]), float32(b[2])})
}

func v2dFrac(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

func v2dVerdict(pass bool) string {
	if pass {
		return "VALIDE"
	}
	return "NON VALIDE"
}

func v2dParseFilms(t *testing.T) []v2dFilmSpec {
	raw := os.Getenv("V2D_FILMS")
	if raw == "" {
		t.Skipf("V2D_FILMS absent : instrument mort-coincidente saute")
	}
	var out []v2dFilmSpec
	for _, tok := range strings.Split(raw, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		i := strings.Index(tok, ":")
		if i < 0 {
			t.Fatalf("V2D_FILMS : entree %q sans ':'", tok)
		}
		out = append(out, v2dFilmSpec{strings.TrimSpace(tok[:i]), strings.TrimSpace(tok[i+1:])})
	}
	if len(out) == 0 {
		t.Skipf("V2D_FILMS vide")
	}
	return out
}

func v2dDir(short8 string) string {
	root := os.Getenv("V2D_FILM_ROOT")
	if root == "" {
		root = `C:\Users\Guillaume\Projects\LevelUp\data\cache`
	}
	return root + `\film_chunks\` + short8
}

func v2dLoadBounds(t *testing.T) *filmdec.MapQuantCatalog {
	path := os.Getenv("V2D_BOUNDS")
	if path == "" {
		path = `C:\Users\Guillaume\Projects\LevelUp\data\titles\halo_infinite\reference\map_quant_bounds.json`
	}
	cat, err := filmdec.LoadMapQuantCatalog(path)
	if err != nil {
		t.Fatalf("catalogue de bornes illisible : %v", err)
	}
	return cat
}

// shortOf rend les 8 derniers caracteres d'un chemin (le short8), pour le journal.
func shortOf(dir string) string {
	if i := strings.LastIndexAny(dir, `\/`); i >= 0 {
		return dir[i+1:]
	}
	return dir
}
