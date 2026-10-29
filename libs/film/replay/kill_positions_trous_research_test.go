//go:build research

package replay

// kill_positions_trous_research_test.go — POURQUOI UNE MORT N'A-T-ELLE PAS DE POSITION ?
//
// Enquete du 2026-10-17 (question du user : « le joueur est mort, on a sa position, pareil pour le
// tueur, non ? »). Instrument de LECTURE : il relit les faits deja cuits (fixtures versionnes et,
// avec L1_DEPOT, le cache de faits du poste), AUCUN octet de film n'est decode, rien n'est ecrit.
//
// Il place chaque mort exactement comme `kill_positions` (`siegesDe` + `positionOf`, instant
// `TempsMS*1000 + FilmClockOriginUS`, tolerance ± 120 ms) et, quand un cote n'est pas place, dit
// ce que le film contient autour : dernier et prochain echantillon du siege, embarquement non
// referme (evenements 8/22), teleportation, immobilite (meme quantum), tirs du joueur dans le
// trou, mort anterieure du tueur. Il mesure aussi la cadence des positions et ce qui se passe
// pendant les trous d'une piste vivante (deplacement a travers le trou).
//
// LIMITE CONNUE : les faits ne conservent ni le chunk ni l'index de paquet d'un echantillon de
// facon exploitable (un seul « chunk » par film, index 0) ; les lignes « reprise dans un AUTRE
// chunk » et « index de paquet » ne mesurent donc rien. REG_COLLECTEUR=1 construit le registre
// comme `killcollector.entreeDuRegistre` (sans table du film, tirs ni horloge). KP_DUMP_KS=<films>
// journalise les noms du fil des morts du film (`ff.Kills`).
//
//	L1_DEPOT=<checkout qui porte data/cache> KP_KILLS=<tsv film,time_ms,killer,victim> \
//	  go test -tags research -count=1 -run '^TestTrousDesPositionsDeMort$' -v ./internal/games/halo_infinite/film/replay/

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/JGtm/LevelUp/libs/film/internal/grammar"
	"github.com/JGtm/LevelUp/libs/film/internal/profile"
)

// trouBilan accumule les comptes de tout le parc.
type trouBilan struct {
	films                                                     int
	morts, placees                                            int
	ambigues, sansSiege                                       int
	sansEchantillon                                           int
	causesVictime                                             map[string]int
	ecartsVictime                                             []int64 // ms, plus proche echantillon d un siege nomme
	finDeVieMoinsMort                                         []int64 // ms : dernier echantillon du siege (<= t+2 s) - instant
	exemples                                                  []string
	kills, tueurPlace                                         int
	causesTueur                                               map[string]int
	ecartsTueur                                               []int64
	exemplesTueur                                             []string
	intervalles                                               []int64   // ms, entre deux echantillons consecutifs d un slot
	trous                                                     int       // intervalles > 120 ms au sein d une piste
	trousDeplacement                                          []float64 // m, deplacement a travers le trou
	trousMemeQuantum                                          int
	tirsDansTrouVictime                                       int
	victimesSansEchAvecTirsDansTrou                           int
	longsTrous, longsTrousNouveauChunk, longsTrousMemeQuantum int
	longsTrousIdxPaquet, longsTrousDuree                      []int64
	dureeChunk                                                []int64
	sautsTueur                                                map[string][]float64
}

func nouveauBilan() *trouBilan {
	return &trouBilan{causesVictime: map[string]int{}, causesTueur: map[string]int{}}
}

// contexteFilm : ce qui sert a classer un cote non place.
type contexteFilm struct {
	nom      string
	build    string
	g        *FilmFacts
	reg      IdentityRegistry
	pistes   map[uint32]slotTrack
	sieges   []uint32
	origine  int64
	boards   map[uint32][]uint64 // slot occupant -> instants d embarquement
	exits    map[uint32][]uint64
	telep    map[uint32][]uint64
	tirsPar  map[int][]uint64 // index de joueur -> instants de tir
	mortsPar map[uint64][]int64
	// vehicules : les echantillons de position des vehicules (`ti=40`), tries par instant.
	vehicules []grammar.BipedPosition
}

func preparerFilm(nom, build string, g *FilmFacts) (*contexteFilm, bool) {
	sorted := append([]grammar.BipedPosition(nil), g.Positions...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].TimestampUS < sorted[j].TimestampUS })
	if len(sorted) == 0 || g.FilmClockOriginUS == 0 {
		return nil, false
	}
	opt := g.options()
	origin := sorted[0].TimestampUS
	step := uint64(opt.frameIntervalMS()) * 1000
	in := IdentityInput{
		Positions: sorted, BipedCreations: g.BipedCreations,
		Deaths: g.Deaths, PlayerIndices: g.PlayerIndices, FilmTable: g.FilmTable,
		Bots: opt.Bots, Fire: fireRefs(g.Fire), RosterXUIDs: opt.RosterXUIDs,
		Participants: opt.Participants,
		Clock:        IdentityClock{OriginUS: origin, StepUS: step, FrameCount: frameSpan(sorted, origin, step)},
		MatchID:      g.Film,
	}
	if os.Getenv("REG_COLLECTEUR") != "" {
		// LE REGISTRE DU COLLECTEUR (`killcollector.entreeDuRegistre`) : ni table du film, ni tirs, ni horloge.
		in.FilmTable, in.Fire, in.Clock = grammar.FilmPlayerTable{}, nil, IdentityClock{}
	}
	reg := BuildIdentityRegistry(context.Background(), in)
	c := &contexteFilm{nom: nom, build: build, g: g, reg: reg, origine: int64(g.FilmClockOriginUS),
		boards: map[uint32][]uint64{}, exits: map[uint32][]uint64{}, telep: map[uint32][]uint64{},
		tirsPar: map[int][]uint64{}, mortsPar: map[uint64][]int64{}}
	c.pistes = indexBySlot(sorted)
	c.sieges = siegesTries(c.pistes)
	for _, e := range g.Vehicles.Events {
		if !e.OccupantPresent {
			continue
		}
		switch e.Kind {
		case grammar.EventBipedBoardVehicle:
			c.boards[e.OccupantSlot] = append(c.boards[e.OccupantSlot], e.TimestampUS)
		case grammar.EventUnitExitVehicle:
			c.exits[e.OccupantSlot] = append(c.exits[e.OccupantSlot], e.TimestampUS)
		}
	}
	c.vehicules = append([]grammar.BipedPosition(nil), g.Vehicles.Positions...)
	sort.SliceStable(c.vehicules, func(i, j int) bool { return c.vehicules[i].TimestampUS < c.vehicules[j].TimestampUS })
	for _, t := range g.Translocations {
		c.telep[t.Slot] = append(c.telep[t.Slot], t.TimestampUS)
	}
	for _, f := range g.Fire {
		if f.HasShooter {
			c.tirsPar[f.FilmIndex] = append(c.tirsPar[f.FilmIndex], f.TimestampUS)
		}
	}
	for _, d := range g.Deaths {
		if d.XUID != 0 {
			c.mortsPar[d.XUID] = append(c.mortsPar[d.XUID], d.TimeMS)
		}
	}
	return c, true
}

func (b *trouBilan) mesurerCadence(c *contexteFilm) {
	bornes := map[int][2]uint64{}
	for _, p := range c.g.Positions {
		bo, ok := bornes[p.Chunk]
		if !ok {
			bo = [2]uint64{p.TimestampUS, p.TimestampUS}
		}
		bo[0], bo[1] = min(bo[0], p.TimestampUS), max(bo[1], p.TimestampUS)
		bornes[p.Chunk] = bo
	}
	for _, bo := range bornes {
		b.dureeChunk = append(b.dureeChunk, int64(bo[1]-bo[0])/1000)
	}
	for _, s := range c.sieges {
		pts := c.pistes[s].pts
		for i := 1; i < len(pts); i++ {
			dt := int64(pts[i].TimestampUS-pts[i-1].TimestampUS) / 1000
			if dt <= 0 {
				continue
			}
			b.intervalles = append(b.intervalles, dt)
			if dt > 2000 {
				b.longsTrous++
				if pts[i].Chunk != pts[i-1].Chunk {
					b.longsTrousNouveauChunk++
				}
				if pts[i].Q == pts[i-1].Q {
					b.longsTrousMemeQuantum++
				}
				b.longsTrousIdxPaquet = append(b.longsTrousIdxPaquet, int64(pts[i].PacketIndex))
				b.longsTrousDuree = append(b.longsTrousDuree, dt)
			}
			if dt > 120 && dt <= 2000 && pts[i].HasWorld && pts[i-1].HasWorld {
				b.trous++
				b.trousDeplacement = append(b.trousDeplacement, distPos(pts[i], pts[i-1]))
				if pts[i].Q == pts[i-1].Q {
					b.trousMemeQuantum++
				}
			}
		}
	}
}

func (b *trouBilan) mesurerFilm(t *testing.T, c *contexteFilm, kills []tsvKill) {
	b.films++
	b.mesurerCadence(c)
	for _, d := range c.g.Deaths {
		if d.XUID == 0 {
			continue
		}
		b.morts++
		tUS := uint64(d.TimeMS*1000 + c.origine)
		r := c.diagnostiquer(d.XUID, tUS, false, d.TimeMS)
		if r.aFinDeVie {
			b.finDeVieMoinsMort = append(b.finDeVieMoinsMort, r.finDeVieMS)
		}
		if r.place {
			b.placees++
			continue
		}
		b.causesVictime[r.cause]++
		switch {
		case r.cause == "ambigu":
			b.ambigues++
		case strings.HasPrefix(r.cause, "sans-siege"):
			b.sansSiege++
		default:
			b.sansEchantillon++
			b.ecartsVictime = append(b.ecartsVictime, r.ecartMS)
			b.tirsDansTrouVictime += r.tirsDansTrou
			if r.tirsDansTrou > 0 {
				b.victimesSansEchAvecTirsDansTrou++
			}
			if r.avantMS < 1000 && len(b.exemples) < 60 {
				b.exemples = append(b.exemples, fmt.Sprintf("%s [%s] t=%d ms victime=%d cause=%s avant=%d ms apres=%d ms tirs-dans-trou=%d",
					c.nom, c.build, d.TimeMS, d.XUID, r.cause, r.avantMS, r.apresMS, r.tirsDansTrou))
			}
		}
	}
	var pfK, pfT, pfV, pfA int
	defer func() {
		if len(kills) > 0 {
			t.Logf("PARFILM %s %s kills=%d tueur_abs=%d victime_abs=%d aucune=%d", c.nom, c.build, pfK, pfT, pfV, pfA)
		}
	}()
	for _, k := range kills {
		b.kills++
		tUS := uint64(k.timeMS*1000 + c.origine)
		r := c.diagnostiquer(k.killer, tUS, true, k.timeMS)
		rv := c.diagnostiquer(k.victim, tUS, false, k.timeMS)
		pfK++
		if !r.place {
			pfT++
		}
		if !rv.place {
			pfV++
		}
		if !r.place && !rv.place {
			pfA++
		}
		if r.place {
			b.tueurPlace++
			continue
		}
		b.causesTueur[r.cause]++
		if r.aSaut {
			cle := strings.Fields(r.cause)[0]
			if b.sautsTueur == nil {
				b.sautsTueur = map[string][]float64{}
			}
			b.sautsTueur[cle] = append(b.sautsTueur[cle], r.saut)
		}
		if r.ecartMS >= 0 {
			b.ecartsTueur = append(b.ecartsTueur, r.ecartMS)
		}
		if len(b.exemplesTueur) < 30 && !strings.HasPrefix(r.cause, "sans-siege:tueur-deja") {
			b.exemplesTueur = append(b.exemplesTueur, fmt.Sprintf("%s [%s] t=%d ms tueur=%d victime=%d cause=%s avant=%d apres=%d tirs-dans-trou=%d",
				c.nom, c.build, k.timeMS, k.killer, k.victim, r.cause, r.avantMS, r.apresMS, r.tirsDansTrou))
		}
	}
	_ = t
}

type tsvKill struct {
	timeMS         int64
	killer, victim uint64
}

func lireKillsTSV(t *testing.T) map[string][]tsvKill {
	t.Helper()
	out := map[string][]tsvKill{}
	p := os.Getenv("KP_KILLS")
	if p == "" {
		return out
	}
	f, err := os.Open(p) //nolint:gosec // mesure
	if err != nil {
		t.Fatalf("KP_KILLS : %v", err)
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		c := strings.Split(sc.Text(), "\t")
		if len(c) != 4 {
			continue
		}
		tm, e1 := strconv.ParseInt(c[1], 10, 64)
		kx, e2 := strconv.ParseUint(c[2], 10, 64)
		vx, e3 := strconv.ParseUint(c[3], 10, 64)
		if e1 != nil || e2 != nil || e3 != nil {
			continue
		}
		out[c[0]] = append(out[c[0]], tsvKill{tm, kx, vx})
	}
	return out
}

func quantiles(v []int64) string {
	if len(v) == 0 {
		return "(vide)"
	}
	s := slices.Clone(v)
	slices.Sort(s)
	q := func(p float64) int64 { return s[int(float64(len(s)-1)*p)] }
	return fmt.Sprintf("n=%d min=%d p10=%d p50=%d p90=%d p99=%d max=%d", len(s), s[0], q(.1), q(.5), q(.9), q(.99), s[len(s)-1])
}

func quantilesF(v []float64) string {
	if len(v) == 0 {
		return "(vide)"
	}
	s := slices.Clone(v)
	slices.Sort(s)
	q := func(p float64) float64 { return s[int(float64(len(s)-1)*p)] }
	return fmt.Sprintf("n=%d p10=%.3f p50=%.3f p90=%.3f p99=%.3f", len(s), q(.1), q(.5), q(.9), q(.99))
}

// distPos : l adaptateur d une ligne vers `dist3`, la seule formule du paquet.
func distPos(a, b grammar.BipedPosition) float64 {
	return dist3([3]float32{a.X, a.Y, a.Z}, [3]float32{b.X, b.Y, b.Z})
}

func rendreBilan(t *testing.T, titre string, b *trouBilan) {
	t.Logf("==== %s : %d films", titre, b.films)
	t.Logf("VICTIMES : %d morts a xuid, placees=%d, non placees=%d (ambigu=%d, sans siege=%d, sans echantillon a +-120 ms=%d)",
		b.morts, b.placees, b.morts-b.placees, b.ambigues, b.sansSiege, b.sansEchantillon)
	for k, v := range b.causesVictime {
		t.Logf("   cause victime %-28s %d", k, v)
	}
	t.Logf("   ecart au plus proche echantillon (ms), victimes sans echantillon : %s", quantiles(b.ecartsVictime))
	t.Logf("   victimes sans echantillon AVEC tirs dans le trou : %d (tirs : %d)", b.victimesSansEchAvecTirsDansTrou, b.tirsDansTrouVictime)
	t.Logf("   fin de vie (dernier echantillon) - instant de la mort (ms), toutes morts : %s", quantiles(b.finDeVieMoinsMort))
	t.Logf("TUEURS : %d frags, places=%d, non places=%d", b.kills, b.tueurPlace, b.kills-b.tueurPlace)
	for k, v := range b.causesTueur {
		t.Logf("   cause tueur %-36s %d", k, v)
	}
	for k, v := range b.sautsTueur {
		t.Logf("   tueur %-34s deplacement (m) entre le dernier point avant et le premier apres le frag : %s", k, quantilesF(v))
	}
	t.Logf("   ecart au plus proche echantillon (ms), tueurs sans echantillon : %s", quantiles(b.ecartsTueur))
	t.Logf("   trous > 2 s au sein d une piste : %d ; reprise dans un AUTRE chunk : %d ; meme quantum : %d ; index de paquet de la reprise : %s ; duree : %s ; duree des chunks (ms) : %s",
		b.longsTrous, b.longsTrousNouveauChunk, b.longsTrousMemeQuantum, quantiles(b.longsTrousIdxPaquet), quantiles(b.longsTrousDuree), quantiles(b.dureeChunk))
	t.Logf("CADENCE : intervalles entre echantillons d un slot (ms) : %s", quantiles(b.intervalles))
	t.Logf("   trous 120 ms-2 s au sein d une piste : %d ; meme quantum de part et d autre : %d ; deplacement (m) : %s",
		b.trous, b.trousMemeQuantum, quantilesF(b.trousDeplacement))
	for _, e := range b.exemples {
		t.Logf("   EX victime %s", e)
	}
	for _, e := range b.exemplesTueur {
		t.Logf("   EX tueur %s", e)
	}
}

func TestTrousDesPositionsDeMort(t *testing.T) {
	kills := lireKillsTSV(t)
	fix := nouveauBilan()
	for _, b := range goldenBuilds() {
		g, _ := chargerGoldenBuild(t, b)
		c, ok := preparerFilm(b.Short8, b.Build, g)
		if !ok {
			continue
		}
		fix.mesurerFilm(t, c, kills[b.Short8])
	}
	rendreBilan(t, "CORPUS VERSIONNE", fix)
	depot := os.Getenv("L1_DEPOT")
	if depot == "" {
		return
	}
	loc := nouveauBilan()
	cat, err := profile.LoadMapQuantCatalog(filepath.Join(depot, "data", "titles", "halo_infinite", "reference", "map_quant_bounds.json"))
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	dir := filepath.Join(depot, "data", "cache", "film_facts", "halo_infinite")
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("cache de faits : %v", err)
	}
	for _, e := range entrees {
		if !strings.HasSuffix(e.Name(), ".filmfacts.bin") {
			continue
		}
		blob, err := os.ReadFile(filepath.Join(dir, e.Name())) //nolint:gosec // mesure
		if err != nil {
			continue
		}
		head, err := DecodeFilmFactsEntete(blob)
		if err != nil {
			continue
		}
		var entry profile.MapQuantEntry
		for _, m := range cat.Maps {
			if m.Module == head.MapModule {
				entry = m
				break
			}
		}
		ff, err := DecodeFilmFactsFile(blob, entry)
		if err != nil {
			continue
		}
		nom := strings.TrimSuffix(e.Name(), ".filmfacts.bin")
		if ff.Kills != nil && strings.Contains(os.Getenv("KP_DUMP_KS"), nom) {
			noms := map[string][2]int{}
			for _, k := range ff.Kills.Kills {
				v := noms[k.Victim]
				v[0]++
				noms[k.Victim] = v
				w := noms[k.Feed.Killer]
				w[1]++
				noms[k.Feed.Killer] = w
			}
			t.Logf("KS %s : %d kills du fil du film ; noms (victime, tueur) : %v", nom, len(ff.Kills.Kills), noms)
		}
		build := "?"
		if ff.Identity != nil {
			build = ff.Identity.Build
		}
		c, ok := preparerFilm(nom, build, &ff.Facts)
		if !ok {
			continue
		}
		loc.mesurerFilm(t, c, kills[nom])
	}
	rendreBilan(t, "FAITS LOCAUX", loc)
}
