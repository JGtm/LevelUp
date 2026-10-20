package replay

// zone_colline_periodes_mesure_test.go — QUELLE COLLINE EST ACTIVE, PERIODE PAR PERIODE, sur un
// film reel : le calque est assemble par le chemin de production (`BuildFromPositions`), puis chaque
// periode du designateur est relue avec trois decomptes de positions par zone :
//
//	rampes     le repli de production (`hillVotesInRamps`) : positions pendant les montees de la
//	           jauge qui tombent dans la periode ;
//	montees    les memes montees, ouvertes a leur PREMIER PAS montant (`hillRampClimbStart`) ;
//	garde      la regle de production (`hillGardeOf`, ici en positions) : le camp PROPRIETAIRE,
//	           aux instants ou il l est — un temoin independant de la jauge.
//
// Elle publie aussi, par periode, la part de garde de chaque camp : le camp qui marque a la fin
// d une periode doit dominer sa garde.
//
// SOUS GARDE D ENVIRONNEMENT, un film par processus, avant-plan, lecture seule :
//
//	$env:CGO_ENABLED=0
//	$env:HILL_FILM="C:/.../data/cache/film_chunks/0d9a9af9"
//	$env:HILL_MAP_ID="e8d56863-9ad4-4efe-9059-81270884589c"; $env:HILL_CARTE="Forest"
//	go test -count=1 -run TestCollinePeriodesMesure -v ./internal/games/halo_infinite/film/replay/

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/replay/mapvar"
)

// TestCollinePeriodesMesure publie les decomptes par periode d un film KOTH.
func TestCollinePeriodesMesure(t *testing.T) {
	dir := os.Getenv("HILL_FILM")
	if dir == "" {
		t.Skip("mesure non demandee : HILL_FILM vide")
	}
	short := filepath.Base(dir)
	sc, err := grammar.ScanFilmManagedProperties(dir)
	if err != nil {
		t.Fatalf("balayage ti=13 : %v", err)
	}
	zones := p2aZones(t, os.Getenv("HILL_MAP_ID"), mapvar.RoleHill)
	cuire := etatInitialCuisson(t, dir, short, p2aQuant(t, os.Getenv("HILL_CARTE")))
	doc, origin := cuire(ZoneInput{Scanned: true, Reads: sc.Reads, KeyReads: sc.KeyReads, Zones: zones,
		Roles: "hill", Hill: true}, nil)
	hillCampsDuRoster(t, doc.Tracks)
	for _, z := range doc.ZoneStates {
		t.Logf("%s : PUBLIE zone %d de %d a %d (%d intervalles, %d points de jauge)", short, z.ZoneRef,
			z.Spans[0].T0, z.Spans[len(z.Spans)-1].T1, len(z.Spans), len(z.Gauge))
	}
	c := zoneCtx{origin: origin, step: uint64(doc.FrameIntervalMS) * 1000, frames: doc.FrameCount,
		intervalMS: doc.FrameIntervalMS, tracks: doc.Tracks}
	ser := zoneSeriesOf(sc.Reads, c)
	ser.noms = zoneNomsDesSlots(sc.KeyReads)
	d, ok := hillDesignatorOf(ser)
	if !ok {
		t.Fatalf("aucun designateur")
	}
	cat := zoneCatalogOf(zones)
	pts := zonePointsByFrame(doc.Tracks)
	owner := ser.owner[hillOwnerSlotOf(ser, d, &ZonesCoverage{}, nil)]
	jauge := ser.gauge[hillModeObjectSlotsGauge(ser.noms, d.slot)]
	for i, p := range hillDesignatedPeriods(d, d.first, c.frames) {
		hillPeriodeMesure(t, fmt.Sprintf("%s P%d [%d-%d]", short, i+1, p.t0, p.t1),
			hillMesureEntree{cat: cat, pts: pts, tracks: doc.Tracks, owner: owner, jauge: jauge}, p)
	}
}

// hillMesureEntree porte ce que la mesure d une periode lit (regle des 5 parametres).
type hillMesureEntree struct {
	cat    []Zone
	pts    map[int][]Point
	tracks []Track
	owner  []zoneSample
	jauge  []zoneSample
}

// hillModeObjectSlotsGauge rend le slot de la jauge du bloc dont le designateur est la cle.
func hillModeObjectSlotsGauge(noms zoneNoms, designateur uint32) uint32 {
	b, ok := zoneBlocDuSlot(designateur, noms, func(b zoneBlocNomme) uint32 { return b.cle })
	if !ok {
		return 0
	}
	return noms.parNom[b.jauge]
}

// hillPeriodeMesure publie les trois decomptes et la garde par camp d une periode.
func hillPeriodeMesure(t *testing.T, nom string, e hillMesureEntree, p hillPeriod) {
	t.Helper()
	ramps := findZoneRamps(0, e.jauge)
	q := p
	prod := hillVotesInRamps(e.cat, e.pts, ramps, &q)
	climb := map[int]int{}
	for _, r := range ramps {
		t0 := hillRampClimbStartForTest(r, e.jauge)
		if r.tPeak < p.t0 || t0 > p.t1 {
			continue
		}
		for ref, n := range hillVotes(e.cat, e.pts, max(t0, p.t0), min(r.tPeak, p.t1)) {
			climb[ref] += n
		}
	}
	garde, parCamp := hillGardeVotes(e, p)
	t.Logf("%s : rampes %s | montees %s | garde %s | frames tenues par camp %v", nom,
		fmtVotes(prod), fmtVotes(climb), fmtVotes(garde), parCamp)
}

// hillRampClimbStartForTest rend l instant du premier pas MONTANT d une rampe.
func hillRampClimbStartForTest(r zoneRamp, ss []zoneSample) int {
	for _, s := range ss {
		if s.t >= r.t0 && s.t <= r.tPeak && s.v > r.start {
			return s.t
		}
	}
	return r.t0
}

// hillGardeVotes compte les positions dans une zone des joueurs du camp proprietaire, aux frames
// ou le canal de propriete le donne proprietaire, et les frames tenues par camp.
func hillGardeVotes(e hillMesureEntree, p hillPeriod) (map[int]int, map[uint64]int) {
	type tp struct {
		team uint64
		pt   Point
	}
	parFrame := map[int][]tp{}
	for _, tr := range e.tracks {
		for _, pt := range tr.Points {
			if pt.T >= p.t0 && pt.T <= p.t1 {
				parFrame[pt.T] = append(parFrame[pt.T], tp{team: uint64(tr.Team), pt: pt}) //nolint:gosec // camp positif
			}
		}
	}
	votes, parCamp := map[int]int{}, map[uint64]int{}
	cur, k := uint64(zoneNeutralOwner), 0
	for f := p.t0; f <= p.t1; f++ {
		for k < len(e.owner) && e.owner[k].t <= f {
			cur = e.owner[k].v
			k++
		}
		if cur == zoneNeutralOwner {
			continue
		}
		parCamp[cur]++
		vus := map[int]bool{}
		for _, x := range parFrame[f] {
			if x.team != cur {
				continue
			}
			if best, hits := nearestZones(e.cat, x.pt); len(hits) == 1 && best <= zoneCaptureDistanceM {
				vus[hits[0].SpatialRank] = true
			}
		}
		for ref := range vus {
			votes[ref]++
		}
	}
	return votes, parCamp
}

// fmtVotes rend un decompte trie par zone.
func fmtVotes(v map[int]int) string {
	refs := make([]int, 0, len(v))
	for r := range v {
		refs = append(refs, r)
	}
	sort.Ints(refs)
	var sb strings.Builder
	for _, r := range refs {
		fmt.Fprintf(&sb, "z%d=%d ", r, v[r])
	}
	return strings.TrimSpace(sb.String())
}

// hillCampsDuRoster pose sur les pistes le camp que l artefact publie (`HILL_ARTEFACT`), par xuid :
// le document assemble ici n a pas de roster, et le temoin de garde a besoin du camp de chaque vie.
func hillCampsDuRoster(t *testing.T, tracks []Track) {
	t.Helper()
	path := os.Getenv("HILL_ARTEFACT")
	if path == "" {
		return
	}
	blob, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("artefact illisible : %v", err)
	}
	var pub ReplayDocument
	if err := json.Unmarshal(blob, &pub); err != nil {
		t.Fatalf("artefact non decodable : %v", err)
	}
	camp := map[string]int{}
	for _, tr := range pub.Tracks {
		if tr.XUID != "" {
			camp[tr.XUID] = tr.Team
		}
	}
	for i := range tracks {
		if c, ok := camp[tracks[i].XUID]; ok {
			tracks[i].Team = c
		}
	}
}
