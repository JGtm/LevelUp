//go:build research

package replay

// death_context_lieu_unique_research_test.go — E8.1 (D-7, D-16) : COMBIEN DE CONTEXTES DE MORT
// PERD-ON QUAND LA VICTIME EST PLACEE PAR LA LECTURE DE `kill_positions` ?
//
// # LA QUESTION
//
// `ContextesDesMorts` placait la victime sur le dernier echantillon anterieur a l'instant (jusqu'a
// 1 s), sur le calage du registre ; elle est desormais placee par la lecture a l'instant de
// `kill_positions` (echantillon a ± 120 ms, tenue, siege), sur l'origine d'horloge du film. Le plan
// exige zero contexte PERDU ; chaque perte est ventilee par la voie de la lecture. L'occupation et
// les vehicules viennent des faits (`Vehicles.Occupancy`, `Vehicles.Positions`). REG_COLLECTEUR=1
// construit le registre comme le collecteur.
//
// # CE QUI EST MESURE, ET SUR QUOI
//
// Pour chaque film : les morts du fil du FILM dont la victime a un xuid forment le journal (la
// base porte le meme fil, `match_kill_events`) ; chaque victime est seule de son camp, de sorte
// que le contexte sort si et seulement si la victime est LOCALISEE. L'ancienne localisation est
// relue par `indexerParXUID(...).visibleA`, toujours en production pour les coequipiers. Le
// registre est celui de l'assemblage (memes entrees que `BuildFromPositions`).
//
// Les faits viennent des fixtures versionnes (huit builds) et, avec L1_DEPOT, du cache de faits
// du poste (`data/cache/film_facts/halo_infinite`). AUCUN OCTET DE FILM n'est lu.
//
//	L1_DEPOT=<checkout qui porte data/cache> \
//	  go test -tags research -count=1 -run '^TestLieuUniqueDesMorts$' -v ./internal/games/halo_infinite/film/replay/

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// lieuMesure : le bilan d'un film.
type lieuMesure struct {
	film                  string
	victimes, avant       int
	apres, gagnes         int
	perdus                int
	perdusSansEchantillon int
	perdusAmbigus         int
	perdusHorsPont        int
	ecartsDesPerdus       []int64   // ms, au plus proche echantillon d un siege de la victime
	ecarts                []float64 // m, victimes localisees des deux cotes
	origineMoinsCalageMS  int64
	pontPubliable         bool
	// voies : la voie de la lecture pour chaque victime ; causes : la voie des PERDUES.
	voies   [6]int
	causes  map[string]int
	details []string
	// tenuesApresUnChangement : victimes tenues, localisees avant, dont les deux derniers echantillons
	// different — celles qu une tenue reservee au quantum repete perdrait.
	tenuesApresUnChangement int
}

// nomDeVoie nomme une voie de la lecture pour le rapport.
func nomDeVoie(v voieDuLieu) string {
	return [...]string{"muette", "echantillon", "tenue", "siege", "a-bord-sans-vehicule", "debarque"}[v]
}

// mesurerLieuUnique compare l'ancienne et la nouvelle localisation des victimes d'un film.
func mesurerLieuUnique(film string, g *FilmFacts) lieuMesure {
	sorted := append([]grammar.BipedPosition(nil), g.Positions...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].TimestampUS < sorted[j].TimestampUS })
	m := lieuMesure{film: film, causes: map[string]int{}}
	c, ok := preparerFilm(film, "", g)
	if !ok {
		return m
	}
	reg := c.reg
	m.pontPubliable = reg.PontPubliable()
	m.origineMoinsCalageMS = int64(g.FilmClockOriginUS)/1000 - reg.DeathOffsetMS()

	var journal []MortDuJournal
	equipes := map[uint64]int{}
	for _, d := range g.Deaths {
		if d.XUID == 0 {
			continue
		}
		journal = append(journal, MortDuJournal{VictimeXUID: d.XUID, TempsMS: d.TimeMS})
		if _, ok := equipes[d.XUID]; !ok {
			equipes[d.XUID] = len(equipes)
		}
	}
	m.victimes = len(journal)
	bord := LectureDuBord{Lue: true, Occupation: g.Vehicles.Occupancy, Vehicules: g.Vehicles.Positions}
	entree := EntreeContexteMorts{
		Positions: sorted, Registre: reg, Journal: journal, Equipes: equipes,
		OrigineDuFilmUS: int64(g.FilmClockOriginUS), Bord: bord,
	}
	apres := map[[2]int64]ContexteMort{}
	for _, c := range ContextesDesMorts(entree) {
		apres[[2]int64{int64(c.VictimeXUID), c.TempsMS}] = c
	}
	anciennes := indexerParXUID(sorted, reg)
	pistes := indexBySlot(sorted)
	sieges := siegesTries(pistes)
	lecture := lectureALInstant{tracks: pistes, bord: lireLeBord(bord, sorted, nil),
		morts: nouvellesMortsDuTueur(journal, reg), offsetUS: int64(g.FilmClockOriginUS)}
	for _, j := range journal {
		cle := [2]int64{int64(j.VictimeXUID), j.TempsMS}
		ancien, avait := anciennes.visibleA(j.VictimeXUID, j.TempsMS)
		nouveau, a := apres[cle]
		if avait {
			m.avant++
		}
		if a {
			m.apres++
		}
		tUS := uint64(j.TempsMS*1000 + int64(g.FilmClockOriginUS))
		_, _, voie := lecture.placer(siegesDe(reg, sieges, j.VictimeXUID, tUS), j.VictimeXUID, tUS)
		m.voies[voie]++
		if voie == voieTenue && avait && !dernierQuantumRepete(lecture, reg, sieges, j.VictimeXUID, tUS) {
			m.tenuesApresUnChangement++
		}
		if avait && !a {
			m.causes[nomDeVoie(voie)]++
			m.details = append(m.details, detailDePerte(lecture, reg, sieges, j, tUS))
		}
		switch {
		case avait && !a:
			m.perdus++
			switch causeDePerte(reg, pistes, sieges, j, int64(g.FilmClockOriginUS)) {
			case "ambigu":
				m.perdusAmbigus++
			case "aucun-siege":
				m.perdusHorsPont++
			default:
				m.perdusSansEchantillon++
				m.ecartsDesPerdus = append(m.ecartsDesPerdus, ecartAuPlusProche(reg, pistes, sieges, j, int64(g.FilmClockOriginUS)))
			}
		case !avait && a:
			m.gagnes++
		case avait && a:
			m.ecarts = append(m.ecarts, math.Hypot(ancien.x-nouveau.Lieu.X, ancien.y-nouveau.Lieu.Y))
		}
	}
	return m
}

// causeDePerte dit POURQUOI `positionOf` ne place pas la victime : aucun siege nomme a cet
// instant, deux corps dans la tolerance (ambiguite), ou aucun echantillon a ± 120 ms.
func causeDePerte(reg IdentityRegistry, pistes map[uint32]slotTrack, sieges []uint32,
	j MortDuJournal, origineUS int64,
) string {
	tUS := uint64(j.TempsMS*1000 + origineUS)
	siegesDuJoueur := siegesDe(reg, sieges, j.VictimeXUID, tUS)
	if len(siegesDuJoueur) == 0 {
		return "aucun-siege"
	}
	proches := 0
	for _, s := range siegesDuJoueur {
		if p, d := pistes[s].at(tUS); d <= killPosToleranceUS && p.HasWorld {
			proches++
		}
	}
	if proches > 1 {
		return "ambigu"
	}
	return "aucun-echantillon"
}

func TestLieuUniqueDesMorts(t *testing.T) {
	var tous []lieuMesure
	for _, b := range goldenBuilds() {
		g, _ := chargerGoldenBuild(t, b)
		tous = append(tous, mesurerLieuUnique(b.Short8+" (fixture "+b.Build+")", g))
	}
	fixtures := len(tous)
	if depot := os.Getenv("L1_DEPOT"); depot != "" {
		tous = append(tous, mesurerFaitsLocaux(t, depot)...)
	}
	rendre := func(titre string, ms []lieuMesure) {
		tot := lieuMesure{causes: map[string]int{}}
		var ecarts, offs []float64
		var perdusEcarts []int64
		for _, m := range ms {
			tot.victimes += m.victimes
			tot.avant += m.avant
			tot.apres += m.apres
			tot.perdus += m.perdus
			tot.perdusSansEchantillon += m.perdusSansEchantillon
			tot.perdusAmbigus += m.perdusAmbigus
			tot.perdusHorsPont += m.perdusHorsPont
			tot.gagnes += m.gagnes
			tot.tenuesApresUnChangement += m.tenuesApresUnChangement
			for i, n := range m.voies {
				tot.voies[i] += n
			}
			for k, n := range m.causes {
				tot.causes[k] += n
			}
			ecarts = append(ecarts, m.ecarts...)
			perdusEcarts = append(perdusEcarts, m.ecartsDesPerdus...)
			offs = append(offs, float64(m.origineMoinsCalageMS))
			if m.perdus > 0 {
				t.Logf("  PERTE %s : %d/%d (aucun-echantillon=%d ambigu=%d aucun-siege=%d)", m.film,
					m.perdus, m.victimes, m.perdusSansEchantillon, m.perdusAmbigus, m.perdusHorsPont)
				for _, d := range m.details {
					t.Logf("    %s", d)
				}
			}
		}
		slices.Sort(ecarts)
		slices.Sort(perdusEcarts)
		if len(perdusEcarts) > 0 {
			t.Logf("  ecart (ms) au plus proche echantillon des %d victimes perdues : min=%d p50=%d p90=%d max=%d",
				len(perdusEcarts), perdusEcarts[0], perdusEcarts[len(perdusEcarts)/2], perdusEcarts[len(perdusEcarts)*9/10], perdusEcarts[len(perdusEcarts)-1])
		}
		slices.Sort(offs)
		q := func(v []float64, p float64) float64 {
			if len(v) == 0 {
				return 0
			}
			return v[int(float64(len(v)-1)*p)]
		}
		t.Logf("%s : %d films, %d morts avec xuid ; localisees avant=%d apres=%d ; PERDUES=%d "+
			"(aucun echantillon a +-120 ms=%d, ambiguite de corps=%d, aucun siege nomme=%d) ; GAGNEES=%d ; "+
			"ecart de lieu (m) sur %d communes : p50=%.3f p90=%.3f p99=%.3f max=%.3f ; origine-calage (ms) min=%.0f p50=%.0f max=%.0f",
			titre, len(ms), tot.victimes, tot.avant, tot.apres, tot.perdus, tot.perdusSansEchantillon,
			tot.perdusAmbigus, tot.perdusHorsPont, tot.gagnes, len(ecarts),
			q(ecarts, .5), q(ecarts, .9), q(ecarts, .99), q(ecarts, 1), q(offs, 0), q(offs, .5), q(offs, 1))
		t.Logf("  voies de la lecture (toutes victimes) : muette=%d echantillon=%d tenue=%d siege=%d a-bord-sans-vehicule=%d debarque=%d ; voies des PERDUES : %v",
			tot.voies[0], tot.voies[1], tot.voies[2], tot.voies[3], tot.voies[4], tot.voies[5], tot.causes)
		t.Logf("  victimes tenues localisees avant dont la derniere emission etait un changement : %d", tot.tenuesApresUnChangement)
	}
	rendre("CORPUS VERSIONNE", tous[:fixtures])
	if len(tous) > fixtures {
		rendre("FAITS LOCAUX", tous[fixtures:])
	}
}

// mesurerFaitsLocaux relit, un par un, les fichiers de faits du cache du poste.
func mesurerFaitsLocaux(t *testing.T, depot string) []lieuMesure {
	t.Helper()
	cat, err := profile.LoadMapQuantCatalog(filepath.Join(depot, "data", "titles", "halo_infinite", "reference", "map_quant_bounds.json"))
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	dir := filepath.Join(depot, "data", "cache", "film_facts", "halo_infinite")
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("cache de faits %s : %v", dir, err)
	}
	var out []lieuMesure
	for _, e := range entrees {
		if !strings.HasSuffix(e.Name(), ".filmfacts.bin") {
			continue
		}
		blob, err := os.ReadFile(filepath.Join(dir, e.Name())) //nolint:gosec // mesure
		if err != nil {
			t.Logf("%s illisible : %v", e.Name(), err)
			continue
		}
		head, err := DecodeFilmFactsEntete(blob)
		if err != nil {
			t.Logf("%s : en-tete illisible : %v", e.Name(), err)
			continue
		}
		var entry profile.MapQuantEntry
		for _, c := range cat.Maps {
			if c.Module == head.MapModule {
				entry = c
				break
			}
		}
		ff, err := DecodeFilmFactsFile(blob, entry)
		if err != nil {
			t.Logf("%s : faits illisibles : %v", e.Name(), err)
			continue
		}
		out = append(out, mesurerLieuUnique(strings.TrimSuffix(e.Name(), ".filmfacts.bin"), &ff.Facts))
	}
	return out
}

// ecartAuPlusProche rend, en ms, l ecart entre l instant de la mort et l echantillon le plus
// proche d un siege de la victime (tous sieges nommes a cet instant).
func ecartAuPlusProche(reg IdentityRegistry, pistes map[uint32]slotTrack, sieges []uint32,
	j MortDuJournal, origineUS int64,
) int64 {
	tUS := uint64(j.TempsMS*1000 + origineUS)
	mini := int64(math.MaxInt64)
	for _, s := range siegesDe(reg, sieges, j.VictimeXUID, tUS) {
		if p, d := pistes[s].at(tUS); p.HasWorld && int64(d)/1000 < mini {
			mini = int64(d) / 1000
		}
	}
	return mini
}

// detailDePerte decrit une victime perdue : ses sieges a l'instant, leur echantillon le plus proche,
// et les lectures d'occupation de chacun autour de l'instant.
func detailDePerte(l lectureALInstant, reg IdentityRegistry, sieges []uint32, j MortDuJournal, tUS uint64) string {
	var b strings.Builder
	fmt.Fprintf(&b, "victime %d a %d ms (film %d us) :", j.VictimeXUID, j.TempsMS, tUS)
	for _, s := range siegesDe(reg, sieges, j.VictimeXUID, tUS) {
		p, d := l.tracks[s].at(tUS)
		fmt.Fprintf(&b, " [siege %d proche a %d us (%v, monde=%v, %.1f %.1f %.1f, %d points)", s, d, int64(p.TimestampUS)-int64(tUS), p.HasWorld, p.X, p.Y, p.Z, len(l.tracks[s].pts))
		for _, o := range l.bord.occupation[s] {
			if o.TimestampUS+5_000_000 >= tUS && o.TimestampUS <= tUS+1_000_000 {
				fmt.Fprintf(&b, " occ@%d attache=%v parent=%d", int64(o.TimestampUS)-int64(tUS), o.Attached, o.ParentSlot)
				if o.Attached {
					vp, vd := l.bord.vehicules[o.ParentSlot].at(tUS)
					bp, bd := l.tracks[o.ParentSlot].at(tUS)
					fmt.Fprintf(&b, " (vehicule proche a %d us, monde=%v ; bipede du parent proche a %d us, %.1f %.1f %.1f)", vd, vp.HasWorld, bd, bp.X, bp.Y, bp.Z)
				}
			}
		}
		b.WriteString("]")
	}
	return b.String()
}

// dernierQuantumRepete dit que les deux derniers echantillons du siege tenu portent le meme quantum.
func dernierQuantumRepete(l lectureALInstant, reg IdentityRegistry, sieges []uint32, xuid uint64, tUS uint64) bool {
	_, slot, ok := dernierEchantillonAvant(l.tracks, siegesDe(reg, sieges, xuid, tUS), tUS)
	if !ok {
		return false
	}
	pts := l.tracks[slot].pts
	i := sort.Search(len(pts), func(k int) bool { return pts[k].TimestampUS > tUS })
	return i >= 2 && pts[i-1].Q == pts[i-2].Q
}
