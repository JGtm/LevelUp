//go:build research

package replay

// kill_positions_e81bc_mesure_research_test.go — E8.1 (b) (c) : MESURE AVANT / APRES DE LA TENUE ET DU
// SIEGE SUR `kill_positions`.
//
// Meme lecture que `kill_positions_e81_mesure_research_test.go` (faits cuits, couples du journal
// KP_KILLS, lignes de la base KP_BASE, registre du collecteur avec REG_COLLECTEUR=1), trois etats :
//
//	base   ce que la base porte ;
//	ad     (a) + (d) : couples du journal, tueur posthume, AUCUNE lecture du bord (etat de HEAD) ;
//	bc     ad + la lecture du bord des faits (`Vehicles.Occupancy`, `Vehicles.Positions`) : tenue et siege.
//
// Un cote PERDU est place par la reference et nil apres. Pour les cotes encore nil, la voie de la
// lecture dit pourquoi (muette, a bord sans vehicule, debarque) et la classe de l'enquete precise.
//
//	L1_DEPOT=<checkout> KP_KILLS=<tsv> KP_BASE=<tsv> REG_COLLECTEUR=1 \
//	  go test -tags research -count=1 -run '^TestMesureE81bcKillPositions$' -v ./internal/games/halo_infinite/film/replay/

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JGtm/LevelUp/libs/film/internal/profile"
)

type e81bcBilan struct {
	films, kills                 int
	tueurAbs, victimeAbs, aucune [3]int // base, ad, bc
	perdusContreBase             [2]int
	perdusContreAD               [2]int
	ecartsAD                     [2]int // cotes places des deux cotes (ad, bc) et differents de plus d'1 mm
	rep                          KillPosReport
	voiesTueur, voiesVictime     map[string]int
}

func (b *e81bcBilan) mesurer(c *contexteFilm, kills []tsvKill, base map[cleKP]ligneBase, short string) {
	if len(kills) == 0 {
		return
	}
	b.films++
	refs := make([]KillRef, 0, len(kills))
	vus := map[[2]int64]bool{}
	var morts []MortDuJournal
	for _, k := range kills {
		refs = append(refs, KillRef{KillerXUID: k.killer, VictimXUID: k.victim, TimeMS: k.timeMS})
		cle := [2]int64{int64(k.victim), k.timeMS} //nolint:gosec // mesure
		if !vus[cle] {
			vus[cle] = true
			morts = append(morts, MortDuJournal{VictimeXUID: k.victim, TempsMS: k.timeMS})
		}
	}
	for _, d := range c.g.Deaths {
		cle := [2]int64{int64(d.XUID), d.TimeMS} //nolint:gosec // mesure
		if d.XUID != 0 && !vus[cle] {
			vus[cle] = true
			morts = append(morts, MortDuJournal{VictimeXUID: d.XUID, TempsMS: d.TimeMS})
		}
	}
	e := EntreeDesPositions{Positions: c.g.Positions, Registre: c.reg, Kills: refs, Morts: morts, OrigineUS: c.origine}
	posAD, _ := BuildKillPositions(e)
	e.Bord = LectureDuBord{Lue: true, Occupation: c.g.Vehicles.Occupancy, Vehicules: c.g.Vehicles.Positions}
	posBC, rep := BuildKillPositions(e)
	b.rep.Held += rep.Held
	b.rep.Seated += rep.Seated
	b.rep.AboardNoVehicle += rep.AboardNoVehicle
	b.rep.Disembarked += rep.Disembarked
	b.rep.Posthumous += rep.Posthumous
	b.rep.PosthumousUnplaced += rep.PosthumousUnplaced
	indexer := func(l []KillPosition) map[KillRef]KillPosition {
		m := map[KillRef]KillPosition{}
		for _, p := range l {
			m[p.KillRef] = p
		}
		return m
	}
	iad, ibc := indexer(posAD), indexer(posBC)
	lecture := lectureALInstant{tracks: c.pistes, bord: lireLeBord(e.Bord, c.g.Positions, nil),
		morts: nouvellesMortsDuTueur(morts, c.reg), offsetUS: c.origine}
	for _, r := range refs {
		b.kills++
		lb := base[cleKP{short, r.KillerXUID, r.TimeMS}]
		la, lc := iad[r], ibc[r]
		b.compter(0, lb.killer, lb.victim)
		b.compter(1, la.Killer, la.Victim)
		b.compter(2, lc.Killer, lc.Victim)
		perdu := func(ref, apres *Vec3, dst *[2]int, cote int) {
			if ref != nil && apres == nil {
				dst[cote]++
			}
		}
		perdu(lb.killer, lc.Killer, &b.perdusContreBase, 0)
		perdu(lb.victim, lc.Victim, &b.perdusContreBase, 1)
		perdu(la.Killer, lc.Killer, &b.perdusContreAD, 0)
		perdu(la.Victim, lc.Victim, &b.perdusContreAD, 1)
		ecart := func(x, y *Vec3, cote int) {
			if x != nil && y != nil && (x.X-y.X)*(x.X-y.X)+(x.Y-y.Y)*(x.Y-y.Y)+(x.Z-y.Z)*(x.Z-y.Z) > 1e-6 {
				b.ecartsAD[cote]++
			}
		}
		ecart(la.Killer, lc.Killer, 0)
		ecart(la.Victim, lc.Victim, 1)
		tUS := uint64(r.TimeMS*1000 + c.origine) //nolint:gosec // mesure
		if lc.Killer == nil {
			_, _, v := lecture.placer(siegesDe(c.reg, c.sieges, r.KillerXUID, tUS), r.KillerXUID, tUS)
			b.voiesTueur[nomDeVoie(v)+" / "+strings.Fields(c.diagnostiquer(r.KillerXUID, tUS, true, r.TimeMS).cause)[0]]++
		}
		if lc.Victim == nil {
			_, _, v := lecture.placer(siegesDe(c.reg, c.sieges, r.VictimXUID, tUS), r.VictimXUID, tUS)
			b.voiesVictime[nomDeVoie(v)+" / "+strings.Fields(c.diagnostiquer(r.VictimXUID, tUS, false, r.TimeMS).cause)[0]]++
		}
	}
}

func (b *e81bcBilan) compter(i int, k, v *Vec3) {
	if k == nil {
		b.tueurAbs[i]++
	}
	if v == nil {
		b.victimeAbs[i]++
	}
	if k == nil && v == nil {
		b.aucune[i]++
	}
}

func (b *e81bcBilan) rendre(t *testing.T, titre string) {
	t.Logf("==== %s : %d films, %d frags du journal", titre, b.films, b.kills)
	t.Logf("| | avant (base) | (a)+(d), HEAD | (a)+(d)+(b)(c) |")
	t.Logf("| tueur sans position | %d | %d | %d |", b.tueurAbs[0], b.tueurAbs[1], b.tueurAbs[2])
	t.Logf("| victime sans position | %d | %d | %d |", b.victimeAbs[0], b.victimeAbs[1], b.victimeAbs[2])
	t.Logf("| ligne absente (aucun des deux) | %d | %d | %d |", b.aucune[0], b.aucune[1], b.aucune[2])
	t.Logf("voies (cotes) : tenus=%d sieges=%d a-bord-sans-vehicule=%d debarques=%d ; posthumes=%d sans lieu=%d",
		b.rep.Held, b.rep.Seated, b.rep.AboardNoVehicle, b.rep.Disembarked, b.rep.Posthumous, b.rep.PosthumousUnplaced)
	t.Logf("PERDUS contre la base : tueur=%d victime=%d ; contre (a)+(d) : tueur=%d victime=%d ; ecarts > 1 mm contre (a)+(d) : tueur=%d victime=%d",
		b.perdusContreBase[0], b.perdusContreBase[1], b.perdusContreAD[0], b.perdusContreAD[1], b.ecartsAD[0], b.ecartsAD[1])
	rendreCauses(t, "tueur (voie / classe de l enquete)", b.voiesTueur)
	rendreCauses(t, "victime (voie / classe de l enquete)", b.voiesVictime)
}

func TestMesureE81bcKillPositions(t *testing.T) {
	kills, base := lireKillsTSV(t), lireBaseKP(t)
	nouveau := func() *e81bcBilan {
		return &e81bcBilan{voiesTueur: map[string]int{}, voiesVictime: map[string]int{}}
	}
	fix := nouveau()
	for _, b := range goldenBuilds() {
		g, _ := chargerGoldenBuild(t, b)
		if c, ok := preparerFilm(b.Short8, b.Build, g); ok {
			fix.mesurer(c, kills[b.Short8], base, b.Short8)
		}
	}
	fix.rendre(t, "CORPUS VERSIONNE")
	depot := os.Getenv("L1_DEPOT")
	if depot == "" {
		return
	}
	loc := nouveau()
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
		if c, ok := preparerFilm(nom, "", &ff.Facts); ok {
			loc.mesurer(c, kills[nom], base, nom)
		}
	}
	loc.rendre(t, "FAITS LOCAUX")
}
