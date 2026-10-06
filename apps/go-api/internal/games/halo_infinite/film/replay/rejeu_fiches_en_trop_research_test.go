//go:build research

package replay

// rejeu_fiches_en_trop_research_test.go — INSTRUMENT des trois restes de places du lot « toute entree
// du roster a l'equipe que le film ecrit » (D2, D3, D6 ; consigne de l'utilisateur du 2026-10-06 :
// « Y a pas de places en trop »). Lecture seule des faits persistes et de l'artefact.
//
//	RJE_FAITS=<dossier des faits> RJE_ARTEFACTS=<dossier des artefacts> RJE_CATALOGUE=<map_quant_bounds.json> \
//	  go test -tags research -count=1 -run '^TestRJEFichesEnTrop$' -v ./internal/games/halo_infinite/film/replay/

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

func TestRJEFichesEnTrop(t *testing.T) {
	faitsDir, artDir := rjeEnv(t, "RJE_FAITS"), rjeEnv(t, "RJE_ARTEFACTS")
	catalogue := rjeEnv(t, "RJE_CATALOGUE")
	for _, c := range []struct {
		id       string
		de, a    int
		indexVus []int
	}{
		{"859da825", 3100, 3300, []int{2, 9}},
		{"bf2a9f05", 950, 1900, []int{8}},
		{"d1dfbc02", 5700, 5986, nil},
	} {
		r := rjeCharger(t, c.id, faitsDir, artDir, catalogue)
		t.Logf("===== %s (frames %d..%d)", c.id, c.de, c.a)
		rjeImagesCles(t, r)
		rjeEntites(t, r, c.indexVus)
		rjeCreations(t, r, c.de, c.a)
		rjeBotsDuFilm(t, r)
		rjeDernieresVies(t, r)
		rjeMortsApres(t, r, c.de)
		t.Logf("table des index (motif) : %v (lectures %d, desaccords %d)", r.faits.Facts.PlayerIndices.ByXUID,
			r.faits.Facts.PlayerIndices.Readings, r.faits.Facts.PlayerIndices.Disagreements)
	}
}

func rjeImagesCles(t *testing.T, r *rjeFilm) {
	t.Helper()
	var b strings.Builder
	for k := range r.scan.KeyframesUS {
		fmt.Fprintf(&b, " %d:%d", k, r.rjeFrameKF(k))
	}
	t.Logf("images-cles porteuses (rang:frame) :%s", b.String())
}

func rjeEntites(t *testing.T, r *rjeFilm, index []int) {
	t.Helper()
	for k, e := range r.scan.Entities {
		garder := len(index) == 0
		for _, i := range index {
			garder = garder || e.Index == i
		}
		if !garder {
			continue
		}
		apres, prouvee := r.scan.AbsenceProuveeApres(e)
		t.Logf("  entite %d : slot %d index %d equipe %d KF %d..%d (frames %d..%d) vue %d instable %v absence apres %d/%v",
			k, e.Slot, e.Index, e.Team, e.FirstKF, e.LastKF, r.rjeFrameKF(e.FirstKF), r.rjeFrameKF(e.LastKF), e.Seen,
			e.Unstable, apres, prouvee)
	}
}

func rjeCreations(t *testing.T, r *rjeFilm, de, a int) {
	t.Helper()
	for _, c := range r.faits.Facts.BipedCreations {
		f := frameBrute(r.h, c.TimestampUS)
		if (f < de || f > a) && c.Slot != rjeSlotSuivi {
			continue
		}
		t.Logf("  creation : frame %d slot %d gen %d index %d (porte %v) version %d repr %#x", f, c.Slot,
			c.Generation, c.ParticipantIndex, c.HasIndex, c.Version, c.Representation)
	}
}

func rjeBotsDuFilm(t *testing.T, r *rjeFilm) {
	t.Helper()
	if r.faits.Kills == nil {
		return
	}
	non := map[int]bool{}
	for _, b := range r.faits.Kills.Roster.UnpinnedBots {
		non[b.BotID] = true
	}
	for _, b := range r.faits.Kills.Roster.Bots {
		var d []string
		for _, x := range b.Declarations {
			fin := -1
			if x.ToUS != 0 {
				fin = frameBrute(r.h, x.ToUS) - 1
			}
			d = append(d, fmt.Sprintf("[%d..%d]", frameBrute(r.h, x.FromUS), fin))
		}
		t.Logf("  bot du film : %q slot %d bid %d equipe %s non epingle %v declarations %s", b.Name, b.Slot, b.BotID,
			rjeEquipe(b.Team), non[b.BotID], strings.Join(d, " "))
	}
}

func rjeDernieresVies(t *testing.T, r *rjeFilm) {
	t.Helper()
	parCle := map[string][]Track{}
	for _, tr := range r.doc.Tracks {
		parCle[cleDePiste(tr)] = append(parCle[cleDePiste(tr)], tr)
	}
	for _, e := range r.doc.Roster {
		vies := parCle[cleDeRoster(e)]
		sort.Slice(vies, func(i, j int) bool { return vies[i].StartFrame < vies[j].StartFrame })
		var b strings.Builder
		for _, v := range vies[max(0, len(vies)-3):] {
			fmt.Fprintf(&b, " %d[%d..%d]", v.Slot, v.StartFrame, v.EndFrame)
		}
		t.Logf("  %-22q index %2d place %2d (%s) equipe %s presence %v vies %d, dernieres :%s", e.Name, e.FilmIndex,
			e.Seat, e.SeatSource, rjeEquipe(e.Team), e.Presence, len(vies), b.String())
	}
	for _, tr := range r.doc.Tracks {
		if cleDePiste(tr) == "" {
			t.Logf("  vie SANS NOM : slot %d [%d..%d]", tr.Slot, tr.StartFrame, tr.EndFrame)
		}
	}
}

func rjeMortsApres(t *testing.T, r *rjeFilm, de int) {
	t.Helper()
	pas := max(1, r.doc.FrameIntervalMS)
	for _, d := range r.faits.Facts.Deaths {
		if f := int(d.TimeMS) / pas; f >= de-100 {
			t.Logf("  mort (horloge du match, frame ~%d) : %q (xuid %d)", f, d.Gamertag, d.XUID)
		}
	}
}

// rjeSlotSuivi : un slot dont toutes les creations se listent, quel que soit leur instant.
const rjeSlotSuivi = 548

func TestRJESlot548(t *testing.T) {
	faitsDir, artDir := rjeEnv(t, "RJE_FAITS"), rjeEnv(t, "RJE_ARTEFACTS")
	r := rjeCharger(t, "859da825", faitsDir, artDir, rjeEnv(t, "RJE_CATALOGUE"))
	n := 0
	for _, p := range r.faits.Facts.Positions {
		if p.Slot != rjeSlotSuivi {
			continue
		}
		if f := frameBrute(r.h, p.TimestampUS); f >= 1300 {
			n++
			if n <= 12 || n%40 == 0 {
				t.Logf("  position slot %d frame %d (%.2f %.2f %.2f) Q %v monde %v chunk %d paquet %d", p.Slot, f, p.X, p.Y, p.Z, p.Q, p.HasWorld, p.Chunk, p.PacketIndex)
			}
		}
	}
	t.Logf("positions du slot %d apres la frame 1300 : %d", rjeSlotSuivi, n)
}

func TestRJEPositionsDuChunk0(t *testing.T) {
	faitsDir, artDir := rjeEnv(t, "RJE_FAITS"), rjeEnv(t, "RJE_ARTEFACTS")
	for _, id := range []string{"859da825", "bf2a9f05", "d1dfbc02"} {
		r := rjeCharger(t, id, faitsDir, artDir, rjeEnv(t, "RJE_CATALOGUE"))
		parChunk := map[int][2]int{}
		compte := map[int]int{}
		for _, p := range r.faits.Facts.Positions {
			f := frameBrute(r.h, p.TimestampUS)
			b, vu := parChunk[p.Chunk]
			if !vu {
				b = [2]int{f, f}
			}
			parChunk[p.Chunk] = [2]int{min(b[0], f), max(b[1], f)}
			compte[p.Chunk]++
		}
		var cles []int
		for c := range parChunk {
			cles = append(cles, c)
		}
		sort.Ints(cles)
		var s strings.Builder
		for _, c := range cles[:min(6, len(cles))] {
			fmt.Fprintf(&s, " chunk %d : %d positions frames %d..%d ;", c, compte[c], parChunk[c][0], parChunk[c][1])
		}
		t.Logf("%s :%s", id, s.String())
		for _, p := range r.faits.Facts.Positions {
			if p.Chunk == 0 && frameBrute(r.h, p.TimestampUS) > 100 {
				t.Logf("  %s chunk 0 tardif : slot %d frame %d", id, p.Slot, frameBrute(r.h, p.TimestampUS))
			}
		}
	}
}

func TestRJECorpsDuSlot548(t *testing.T) {
	faitsDir, artDir := rjeEnv(t, "RJE_FAITS"), rjeEnv(t, "RJE_ARTEFACTS")
	r := rjeCharger(t, "859da825", faitsDir, artDir, rjeEnv(t, "RJE_CATALOGUE"))
	c := corpsParSlot(r.faits.Facts.BipedCreations)[rjeSlotSuivi]
	for _, d := range c.dates {
		t.Logf("  record : tUS %d (frame %d) index %d gen %d", d.tUS, frameBrute(r.h, uint64(d.tUS)), d.index, d.gen)
	}
	var premier, dernier uint64
	for _, p := range r.faits.Facts.Positions {
		if p.Slot == rjeSlotSuivi && frameBrute(r.h, p.TimestampUS) > 3000 {
			if premier == 0 || p.TimestampUS < premier {
				premier = p.TimestampUS
			}
			dernier = max(dernier, p.TimestampUS)
		}
	}
	d, ok := c.recordA(int64(premier))
	t.Logf("vie du slot %d : us %d..%d ; record au debut : index %d tUS %d (%v)", rjeSlotSuivi, premier, dernier,
		d.index, d.tUS, ok)
}

// TestRJEViesApresDepart mesure l'empreinte de la regle proposee pour D2 sur les films de RJE_FILMS :
// les vies d'une entree qui commencent apres le depart PROUVE de sa derniere entite (image-cle qui
// prouve l'absence), et, parmi elles, celles dont le corps a ete cree avant ce depart.
func TestRJEViesApresDepart(t *testing.T) {
	faitsDir, artDir := rjeEnv(t, "RJE_FAITS"), rjeEnv(t, "RJE_ARTEFACTS")
	catalogue := rjeEnv(t, "RJE_CATALOGUE")
	total, avantDepart := 0, 0
	for _, id := range strings.Split(rjeEnv(t, "RJE_FILMS"), ",") {
		r := rjeCharger(t, id, faitsDir, artDir, catalogue)
		corps := corpsParSlot(r.faits.Facts.BipedCreations)
		parCle := map[string][]Track{}
		for _, tr := range r.doc.Tracks {
			parCle[cleDePiste(tr)] = append(parCle[cleDePiste(tr)], tr)
		}
		for i, e := range r.doc.Roster {
			depart, ok := r.departProuve(r.occ.parEntree[i].entites)
			if !ok {
				continue
			}
			for _, v := range parCle[cleDeRoster(e)] {
				if v.StartFrame < depart {
					continue
				}
				total++
				d, lu := corps[uint32(v.Slot)].recordA(int64(instantDeFrame(r.h, v.StartFrame)))
				cree := frameBrute(r.h, uint64(max(d.tUS, 0)))
				if lu && cree < depart {
					avantDepart++
				}
				t.Logf("%s : %q depart prouve %d, vie slot %d [%d..%d], corps cree %d (index %d, lu %v)", id, e.Name,
					depart, v.Slot, v.StartFrame, v.EndFrame, cree, d.index, lu)
			}
		}
	}
	t.Logf("vies apres un depart prouve : %d, dont corps cree avant le depart : %d", total, avantDepart)
}

// departProuve rend la frame de l'image-cle qui prouve l'absence apres la DERNIERE entite d'une
// entree ; faux sans entite, ou quand l'absence apres l'une d'elles n'est pas prouvee.
func (r *rjeFilm) departProuve(entites []int) (int, bool) {
	depart := -1
	for _, k := range entites {
		rang, prouvee := r.scan.AbsenceProuveeApres(r.scan.Entities[k])
		if !prouvee {
			return 0, false
		}
		depart = max(depart, r.rjeFrameKF(rang))
	}
	return depart, depart >= 0
}
