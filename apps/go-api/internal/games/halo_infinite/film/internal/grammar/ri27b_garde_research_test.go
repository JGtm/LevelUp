//go:build research

package grammar

// ri27b_garde_research_test.go — INSTRUCTION DE 2.7.b : la garde des generations vivantes datees
// (lot R2-bis), que l ancrage applique a chaque record bipede delta, contre les records que la
// marche des trames rend aux huit lecteurs. Par film, et selon le verdict de la trame (fermee,
// refusee, queue opaque) : records de la marche, records que la garde refuse, et les publications
// d arme portee qu ils portent.
//
//	RI27B_FILMS=000d5950 RI27B_RACINE=<film_chunks> go test -tags=research -count=1 \
//	  -run '^TestRI27bGarde$' -v ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// ri27bVerdicts retient le verdict et le debut de chaque trame que la marche rend.
type ri27bVerdicts struct {
	verdict map[paquetDuFlux]lecture.Verdict
	debut   map[paquetDuFlux]lecture.DebutDeVueB
	vueB    map[paquetDuFlux]uint32
}

func (c *ri27bVerdicts) Interets() []Interet                      { return nil }
func (c *ri27bVerdicts) Clore(BilanDeMarche)                      {}
func (c *ri27bVerdicts) Brancher(*Observation, *MarcheDistribuee) {}
func (c *ri27bVerdicts) Trame(p *lecture.Paquet) {
	k := paquetDuFlux{p.Chunk, p.Index}
	c.verdict[k], c.debut[k] = p.Fermeture.Verdict, p.Debut
	if c.vueB != nil {
		c.vueB[k] = p.VueB.Debut
	}
}

func TestRI27bGarde(t *testing.T) {
	films, racine := os.Getenv("RI27B_FILMS"), os.Getenv("RI27B_RACINE")
	if films == "" || racine == "" {
		t.Skip("instrument : RI27B_FILMS et RI27B_RACINE requis")
	}
	noms := map[lecture.Verdict]string{lecture.VerdictFerme: "fermee", lecture.VerdictRefuse: "refusee",
		lecture.VerdictQueueOpaque: "opaque", lecture.VerdictNonRendu: "non_rendue"}
	for _, court := range strings.Split(films, ",") {
		fc := ri27bContexte(t, filepath.Join(racine, court), ri27bCarte(t, court))
		arch, err := fc.bipedArchetype()
		if err != nil {
			t.Fatal(err)
		}
		armes := weaponEmplacements(arch)
		canal := nouveauCanalDesLecturesBipedes(fc)
		v := &ri27bVerdicts{verdict: map[paquetDuFlux]lecture.Verdict{}, debut: map[paquetDuFlux]lecture.DebutDeVueB{}}
		if err := Distribuer(fc, canal, v); err != nil {
			t.Fatalf("%s : %v", court, err)
		}
		type cpt struct{ records, refuses, armes, armesRefusees int }
		par := map[string]*cpt{}
		var exemples []string
		for i := range fc.recup.lectures.records {
			r := &fc.recup.lectures.records[i]
			cle := "recupere"
			if !r.Recupere {
				cle = noms[v.verdict[paquetDuFlux{r.Chunk, r.Packet.Index}]]
			}
			if par[cle] == nil {
				par[cle] = &cpt{}
			}
			c := par[cle]
			c.records++
			nArmes := 0
			for _, a := range r.appels {
				if _, ok := armes[a.composant]; ok {
					nArmes++
				}
			}
			c.armes += nArmes
			if !fc.GenerationsVivantesA(r.Packet.TimestampUS).Accepte(types.LifeKey{Slot: r.Slot, Gen: r.Gen}) {
				c.refuses++
				c.armesRefusees += nArmes
				if len(exemples) < 8 {
					exemples = append(exemples, fmt.Sprintf("%s ts=%d slot=%d gen=%d paquet=%d:%d debut=%d masque=%x",
						cle, r.Packet.TimestampUS, r.Slot, r.Gen, r.Chunk, r.Packet.Index,
						v.debut[paquetDuFlux{r.Chunk, r.Packet.Index}], r.masque))
				}
			}
		}
		for _, cle := range []string{"fermee", "refusee", "opaque", "non_rendue", "recupere"} {
			if c := par[cle]; c != nil {
				t.Logf("%s\t%s\trecords=%d\tgarde_refuse=%d\tappels_arme=%d\tarme_refusee=%d", court, cle,
					c.records, c.refuses, c.armes, c.armesRefusees)
			}
		}
		for _, e := range exemples {
			t.Logf("%s\texemple\t%s", court, e)
		}
	}
}

// TestRI27bAncres liste les records que l ancrage trouve pour des slots donnes dans une fenetre
// d instants (RI27B_DE, RI27B_A, en microsecondes), avec le verdict de leur trame et les slots que
// la marche y a lus.
func TestRI27bAncres(t *testing.T) {
	film, racine, slots := os.Getenv("RI27B_FILM"), os.Getenv("RI27B_RACINE"), os.Getenv("RI27B_SLOTS")
	if film == "" || racine == "" || slots == "" {
		t.Skip("instrument : RI27B_FILM, RI27B_RACINE et RI27B_SLOTS requis")
	}
	var de, a uint64
	fmt.Sscan(os.Getenv("RI27B_DE"), &de)
	fmt.Sscan(os.Getenv("RI27B_A"), &a)
	carte := os.Getenv("RI27B_CARTE")
	if carte == "" {
		carte = ri27bCarte(t, film)
	}
	fc := ri27bContexte(t, filepath.Join(racine, film), carte)
	voulus := map[uint32]bool{}
	for _, s := range strings.Split(slots, ",") {
		var x uint32
		fmt.Sscan(s, &x)
		voulus[x] = true
	}
	canal := nouveauCanalDesLecturesBipedes(fc)
	v := &ri27bVerdicts{verdict: map[paquetDuFlux]lecture.Verdict{}, debut: map[paquetDuFlux]lecture.DebutDeVueB{}}
	if err := Distribuer(fc, canal, v); err != nil {
		t.Fatal(err)
	}
	fc.parcourirLesAncresBipedes(func(r deltaBipedRecord) {
		ts := r.Packet.TimestampUS
		if !voulus[r.Slot] || ts < de || (a > 0 && ts > a) {
			return
		}
		k := paquetDuFlux{r.Chunk, r.Packet.Index}
		tr, vu := canal.trames[k]
		t.Logf("ancre ts=%d slot=%d gen=%d masque=%v paquet=%d:%d verdict=%d debut=%d vu=%v slots_marche=%v", ts, r.Slot, r.Gen,
			r.Mask, r.Chunk, r.Packet.Index, v.verdict[k], v.debut[k], vu, tr.slots)
	})
}

// TestRI27bAncresSansLaMarche compte, par verdict et par debut de vue B de la trame, les records
// ancres dont la marche n a lu aucun record du meme slot dans le paquet, et ceux d entre eux qui
// portent un composant des huit lecteurs.
func TestRI27bAncresSansLaMarche(t *testing.T) {
	films, racine := os.Getenv("RI27B_FILMS"), os.Getenv("RI27B_RACINE")
	if films == "" || racine == "" {
		t.Skip("instrument : RI27B_FILMS et RI27B_RACINE requis")
	}
	for _, court := range strings.Split(films, ",") {
		fc := ri27bContexte(t, filepath.Join(racine, court), ri27bCarte(t, court))
		canal := nouveauCanalDesLecturesBipedes(fc)
		v := &ri27bVerdicts{verdict: map[paquetDuFlux]lecture.Verdict{}, debut: map[paquetDuFlux]lecture.DebutDeVueB{},
			vueB: map[paquetDuFlux]uint32{}}
		if err := Distribuer(fc, canal, v); err != nil {
			t.Fatal(err)
		}
		// avant : l i0 du record precede le debut de la vue B que la marche a lue.
		type cle struct {
			verdict lecture.Verdict
			debut   lecture.DebutDeVueB
			avant   bool
		}
		seuls, utiles := map[cle]int{}, map[cle]int{}
		// parPaquet : les trames fermees dont le debut a ete trouve par la fermeture, et leurs ancres
		// seules d avant ce debut (RI27B_PAQUETS les ecrit, une ligne par paquet).
		parPaquet := map[paquetDuFlux][2]int{}
		fc.parcourirLesAncresBipedes(func(r deltaBipedRecord) {
			k := paquetDuFlux{r.Chunk, r.Packet.Index}
			tr, vu := canal.trames[k]
			if !vu || slices.Contains(tr.slots, r.Slot) {
				return
			}
			c := cle{v.verdict[k], v.debut[k], uint32(r.I0) < v.vueB[k]}
			seuls[c]++
			lu := masqueDesIndex(r.Mask)&canal.utiles != 0
			if lu {
				utiles[c]++
			}
			if c.verdict == lecture.VerdictFerme && c.debut == lecture.DebutParFermeture && c.avant {
				n := parPaquet[k]
				n[0]++
				if lu {
					n[1]++
				}
				parPaquet[k] = n
			}
		})
		if chemin := os.Getenv("RI27B_PAQUETS"); chemin != "" {
			f, err := os.OpenFile(chemin, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err != nil {
				t.Fatal(err)
			}
			for k, n := range parPaquet {
				fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%d\t%d\n", court, k.chunk, k.index, v.vueB[k], n[0], n[1])
			}
			f.Close()
		}
		for c, n := range seuls {
			t.Logf("%s\tverdict=%d\tdebut=%d\tavant=%v\tancres_seules=%d\tavec_composant_lu=%d", court, c.verdict, c.debut,
				c.avant, n, utiles[c])
		}
	}
}

// TestRI27bPopulation ecrit, film par film, la population des huit lecteurs (RI27B_OUT, une ligne
// par record : film, chunk, paquet, slot, recupere, verdict et debut de la trame, composants
// annonces que les lecteurs lisent). Deux versions du code se comparent ligne a ligne.
func TestRI27bPopulation(t *testing.T) {
	films, racine, sortie := os.Getenv("RI27B_FILMS"), os.Getenv("RI27B_RACINE"), os.Getenv("RI27B_OUT")
	if films == "" || racine == "" || sortie == "" {
		t.Skip("instrument : RI27B_FILMS, RI27B_RACINE et RI27B_OUT requis")
	}
	f, err := os.OpenFile(sortie, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, court := range strings.Split(films, ",") {
		fc := ri27bContexte(t, filepath.Join(racine, court), ri27bCarte(t, court))
		canal := nouveauCanalDesLecturesBipedes(fc)
		v := &ri27bVerdicts{verdict: map[paquetDuFlux]lecture.Verdict{}, debut: map[paquetDuFlux]lecture.DebutDeVueB{}}
		if err := Distribuer(fc, canal, v); err != nil {
			t.Fatalf("%s : %v", court, err)
		}
		for i := range fc.recup.lectures.records {
			r := &fc.recup.lectures.records[i]
			k := paquetDuFlux{r.Chunk, r.Packet.Index}
			fmt.Fprintf(f, "%s\t%d\t%d\t%d\t%v\t%d\t%d\t%x\n", court, r.Chunk, r.Packet.Index, r.Slot, r.Recupere,
				v.verdict[k], v.debut[k], r.masque&canal.utiles)
		}
	}
}
