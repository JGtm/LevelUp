//go:build research

package grammar

// ri27b_deux_sources_research_test.go — INSTRUCTION DE 2.7.b : ce que les huit lecteurs de
// composants rendent sous l ancrage seul (la source d avant le lot) et sous la marche des trames
// suivie de l ancrage, sur un film. L ancrage seul est reconstruit par la recuperation sans aucune
// trame vue : elle rend alors tous les records ancres, comme les lecteurs les marchaient.
//
//	RI27B_FILMS=000d5950 RI27B_RACINE=<film_chunks> go test -tags=research -count=1 \
//	  -run '^TestRI27bDeuxSources$' -v ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// ri27bAncrageSeul pose sur le contexte les lectures de l ancrage seul.
func ri27bAncrageSeul(fc *FilmContext) {
	c := nouveauCanalDesLecturesBipedes(fc)
	rec := c.recuperer()
	rangerDansLeFlux(rec, fc.ChunkNumbers())
	fc.recup.lectures = &lecturesBipedes{records: rec, examines: c.lu.examines, recuperes: len(rec)}
}

// ri27bResume rend les changements d arme par genre, et les repetitions (meme famille que la
// precedente de l emplacement).
func ri27bResume(chs []types.HeldWeaponChange) string {
	genres := map[string]int{}
	repetes := 0
	for _, c := range chs {
		genres[string(c.Kind)]++
		if c.Previous == c.Family {
			repetes++
		}
	}
	var cles []string
	for k := range genres {
		cles = append(cles, k)
	}
	sort.Strings(cles)
	var b strings.Builder
	for _, k := range cles {
		fmt.Fprintf(&b, "%s=%d ", k, genres[k])
	}
	return fmt.Sprintf("%d changements (%s) dont %d repetes", len(chs), b.String(), repetes)
}

func TestRI27bDeuxSources(t *testing.T) {
	films, racine := os.Getenv("RI27B_FILMS"), os.Getenv("RI27B_RACINE")
	if films == "" || racine == "" {
		t.Skip("instrument : RI27B_FILMS et RI27B_RACINE requis")
	}
	for _, court := range strings.Split(films, ",") {
		for _, source := range []string{"ancrage seul", "marche puis ancrage"} {
			fc := ri27bContexte(t, filepath.Join(racine, court), ri27bCarte(t, court))
			if source == "ancrage seul" {
				ri27bAncrageSeul(fc)
			}
			chs, st, err := ScanHeldWeaponChanges(fc, nil)
			if err != nil {
				t.Fatalf("%s : %v", court, err)
			}
			t.Logf("%s [%s] : records %d, avec emplacement %d, emissions %d, repetitions %d ; %s", court, source,
				st.Records, st.WithComponent, st.Emissions, st.Repeats, ri27bResume(chs))
			for i, c := range chs {
				if i >= 12 {
					break
				}
				t.Logf("    %d slot=%d emp=%d %s fam=%x prev=%x", c.TimestampUS, c.Slot, c.Emplacement, c.Kind,
					c.Family, c.Previous)
			}
			eq, est, err := ScanEquipmentChanges(fc, nil)
			if err != nil {
				t.Fatalf("%s : equipement : %v", court, err)
			}
			doublons := map[string]int{}
			for _, e := range eq {
				doublons[fmt.Sprintf("%d/%d/%d/%d", e.Slot, e.Chunk, e.PacketIndex, e.Counter)]++
			}
			n := 0
			for _, v := range doublons {
				if v > 1 {
					n += v - 1
				}
			}
			t.Logf("%s [%s] : changements d equipement %d (recuperes %d, vies %d), doublons (slot, paquet, compteur) %d ; "+
				"marche i48 : records %d, annonces %d, lus %d, illisibles %d", court, source, len(eq), est.Recovered, est.Lives,
				n, est.Walk.Records, est.Walk.WithI48, est.Walk.Read, est.Walk.Unread)
		}
	}
}

// ri27bTemoinDesVies releve, pour des slots donnes, les records bipedes de la marche : instant,
// genre, generation, dead-state, si le masque annonce un emplacement d arme, la position (i0) et
// le rang de capacite (i48). RI27B_TOUT=1 garde chaque record, pas seulement les armes, les morts
// et les NEW.
type ri27bTemoinDesVies struct {
	m      *MarcheDistribuee
	slots  map[uint32]bool
	armes  map[int]int
	lignes []string
}

func (c *ri27bTemoinDesVies) Interets() []Interet                          { return nil }
func (c *ri27bTemoinDesVies) Clore(BilanDeMarche)                          {}
func (c *ri27bTemoinDesVies) Brancher(_ *Observation, m *MarcheDistribuee) { c.m = m }
func (c *ri27bTemoinDesVies) Trame(p *lecture.Paquet) {
	recs, lus := c.m.recordsDeLaTrame()
	if !lus {
		return
	}
	for i := range recs {
		r := &recs[i]
		if r.TypeIndex != BipedTypeIndex || !c.slots[r.Slot] {
			continue
		}
		arme := false
		for id := range c.armes {
			if id < 64 && r.Trace.Mask>>uint(id)&1 == 1 {
				arme = true
			}
		}
		c.lignes = append(c.lignes, fmt.Sprintf("ts=%d slot=%d type=%d gen=%d mort=%v arme=%v i0=%v i48=%v masque=%x debut=%d verdict=%d paquet=%d:%d",
			p.TS, r.Slot, r.Type, r.ID>>30, r.Trace.Dead != nil, arme, r.Trace.Mask&1 == 1,
			r.Trace.Mask>>i48Index&1 == 1, r.Trace.Mask, p.Debut, p.Fermeture.Verdict, p.Chunk, p.Index))
	}
}

func TestRI27bTemoinDesVies(t *testing.T) {
	film, racine, slots := os.Getenv("RI27B_FILM"), os.Getenv("RI27B_RACINE"), os.Getenv("RI27B_SLOTS")
	if film == "" || racine == "" || slots == "" {
		t.Skip("instrument : RI27B_FILM, RI27B_RACINE et RI27B_SLOTS requis")
	}
	// RI27B_CARTE nomme la carte d un film hors du corpus d equivalence (un temoin du gate).
	carte := os.Getenv("RI27B_CARTE")
	if carte == "" {
		carte = ri27bCarte(t, film)
	}
	fc := ri27bContexte(t, filepath.Join(racine, film), carte)
	arch, err := fc.bipedArchetype()
	if err != nil {
		t.Fatal(err)
	}
	c := &ri27bTemoinDesVies{slots: map[uint32]bool{}, armes: weaponEmplacements(arch)}
	for _, s := range strings.Split(slots, ",") {
		var v uint32
		fmt.Sscan(s, &v)
		c.slots[v] = true
	}
	cre, _, _ := fc.CreationsDeBipede()
	for _, x := range cre {
		if c.slots[x.Slot] {
			t.Logf("creation ts=%d slot=%d gen=%d", x.TimestampUS, x.Slot, x.Generation)
		}
	}
	if err := Distribuer(fc, c); err != nil {
		t.Fatal(err)
	}
	tout := os.Getenv("RI27B_TOUT") == "1"
	for _, l := range c.lignes {
		if tout || strings.Contains(l, "arme=true") || strings.Contains(l, "mort=true") || strings.Contains(l, "type=1") {
			t.Log(l)
		}
	}
}
