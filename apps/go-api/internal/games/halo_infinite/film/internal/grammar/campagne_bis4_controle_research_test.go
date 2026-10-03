//go:build research

package grammar

// campagne_bis4_controle_research_test.go — MESURES BIS 4 DE LA CAMPAGNE DE GRAMMAIRE
// (2026-10-02), ITEM 1.3 : QUAND UN JOUEUR A-T-IL UNE ENTREE DANS LA VUE C D UN PAQUET ?
//
// La lecture du jeu (`.ai/V7.5/film_re/campagne_grammaire_2026-10-01/ghidra_13/`) dit : l entree
// kind 0 du joueur k entre dans le tampon par joueur que l enregistreur `FUN_142f2c3b0` recopie
// quand `FUN_14076b0e8` est APPELE pour (k, 0) dans la trame, c est-a-dire quand l ordonnanceur
// d au moins un PAIR qui envoie un paquet dans cette trame (`FUN_140516fa0` -> `FUN_14076aca4`)
// tente ce candidat. La decision d envoyer (`FUN_1405185b0`) depend de l horloge murale et du
// debit estime ; le candidat existe si le bit k est en attente pour ce pair (`vue+0x2550`, pose a
// chaque tick ou le joueur a un controle valide ET une unite, `FUN_141f85a84`, efface par une
// ecriture reussie). Rien de cela n est dans le film.
//
// Cette sonde ne mesure que ce que le film montre : les paquets FERMES (vue C lue au bit prouve).
//
//	cadence     : l ecart d horodatage entre deux paquets delta consecutifs d un chunk ;
//	vues vides  : paquets fermes dont la vue C n a que son terminateur ;
//	transitoire : k present dans p-1 ET p+1, absent de p (p-1, p, p+1 consecutifs et fermes),
//	              ventile par vue C de p vide ou non vide ; compte une seconde fois quand les trois
//	              paquets sont SAINS (aucun invariant de l ecrivain contredit, juge [cmJuge]) ;
//	trous       : dans une suite de paquets fermes consecutifs, l ecart entre deux presences de k
//	              (en paquets et en duree).
//
// Un trou d un ou deux paquets ne peut pas etre une mort (la reapparition prend des secondes) :
// une absence transitoire dans une vue C NON vide est une entree que l ordonnanceur n a pas
// tentee ce tick-la.
//
// Rejouable (un film a la fois, plafond 4 Gio) :
//
//	CAMPAGNE_RACINE=<film_chunks> CAMPAGNE_FILMS=<id,id> CAMPAGNE_SORTIE=<dir hors data> \
//	  go test -tags=research -count=1 -timeout 120m -run '^TestCampagneBis4Controle$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"math/bits"
	"os"
	"sort"
	"testing"
	"time"

	"levelup/go-api/internal/filmproc"
)

// b4Paquet : ce que la sonde garde d un paquet delta.
type b4Paquet struct {
	ts     uint64
	fermee bool
	sain   bool   // ferme et ne contredit aucun invariant de l ecrivain (juge [cmJuge])
	masque uint32 // index de controle presents (vue C fermee)
	n      int    // entrees lues
}

// b4Mesure : les compteurs d un film (ou d un build, par somme).
type b4Mesure struct {
	cur  []b4Paquet
	c    map[string]int // compteurs nommes
	dt   map[uint64]int // ecart d horodatage (us) -> paquets
	juge *cmJuge        // juge de la reference, place AVANT la mesure dans le diffuseur
}

func nouvelleB4Mesure() *b4Mesure {
	return &b4Mesure{c: map[string]int{}, dt: map[uint64]int{}}
}

func (m *b4Mesure) debutDeChunk(int, []byte, []FilmPacket, *World) { m.solder() }
func (m *b4Mesure) finDeFilm()                                     { m.solder() }

func (m *b4Mesure) paquet(_ int, p *cmPaquet, _ *World) {
	x := b4Paquet{ts: p.d.TimestampUS, fermee: p.d.Fermee}
	m.c["paquets delta"]++
	if p.d.Fermee {
		m.c["paquets fermes"]++
		if m.juge != nil && !m.juge.contreRef[[2]int{p.d.Chunk, p.d.Index}] {
			x.sain = true
			m.c["paquets fermes sains"]++
		}
		for _, e := range p.d.VueC.Entrees {
			if e.Index < 0 || e.Index >= 32 {
				m.c["entrees hors des 32 places"]++
				continue
			}
			if x.masque&(1<<uint(e.Index)) != 0 {
				m.c["entrees en double"]++
			}
			x.masque |= 1 << uint(e.Index)
			x.n++
			if e.Bloc {
				m.c["entrees utiles fermees"]++
			}
		}
		m.c["entrees fermees"] += x.n
		if x.n == 0 {
			m.c["paquets fermes a vue C vide"]++
			if x.sain {
				m.c["paquets fermes sains a vue C vide"]++
			}
		}
		if x.sain {
			vue := "non vide"
			if x.n == 0 {
				vue = "vide"
			}
			m.c["sains par ecart au precedent : "+b4ClasseEcart(m.cur, x.ts)+", vue C "+vue]++
		}
	}
	m.cur = append(m.cur, x)
}

// b4ClasseEcart range l ecart d horodatage entre un paquet et le paquet delta precedent du chunk.
func b4ClasseEcart(avant []b4Paquet, ts uint64) string {
	if len(avant) == 0 {
		return "premier du chunk"
	}
	prec := avant[len(avant)-1].ts
	switch {
	case ts < prec:
		return "negatif"
	case ts-prec < 12_000:
		return "a < 12 ms"
	case ts-prec < 20_000:
		return "b 12-20 ms"
	case ts-prec < 40_000:
		return "c 20-40 ms"
	}
	return "d >= 40 ms"
}

// b4ClasseTrou range un trou (en paquets).
func b4ClasseTrou(n int) string {
	switch {
	case n == 1:
		return "1"
	case n == 2:
		return "2"
	case n <= 5:
		return "3-5"
	case n <= 29:
		return "6-29"
	case n <= 299:
		return "30-299"
	}
	return ">=300"
}

// b4ClasseDuree range la duree entre deux presences (us).
func b4ClasseDuree(us uint64) string {
	switch {
	case us < 100_000:
		return "a < 0,1 s"
	case us < 1_000_000:
		return "b 0,1-1 s"
	case us < 3_000_000:
		return "c 1-3 s"
	}
	return "d >= 3 s"
}

// solder analyse les paquets du chunk courant, puis les oublie.
func (m *b4Mesure) solder() {
	cur := m.cur
	m.cur = nil
	for i := 1; i < len(cur); i++ {
		if cur[i].ts >= cur[i-1].ts {
			m.dt[cur[i].ts-cur[i-1].ts]++
		} else {
			m.c["ecart d horodatage negatif"]++
		}
	}
	for i := 1; i+1 < len(cur); i++ {
		a, p, b := cur[i-1], cur[i], cur[i+1]
		if !a.fermee || !p.fermee || !b.fermee {
			continue
		}
		pop := []string{"transitoire (fermes) : "}
		if a.sain && p.sain && b.sain {
			pop = append(pop, "transitoire (sains) : ")
		}
		deux := a.masque & b.masque
		for deux != 0 {
			k := uint(bits.TrailingZeros32(deux))
			deux &^= 1 << k
			cas := "absent de p, vue C de p non vide"
			switch {
			case p.masque&(1<<k) != 0:
				cas = "present dans p-1, p, p+1"
			case p.n == 0:
				cas = "absent de p, vue C de p vide"
			}
			for _, x := range pop {
				m.c[x+cas]++
			}
			if len(pop) == 2 && p.masque&(1<<k) == 0 {
				m.c["transitoire (sains) : "+cas+", ecart "+b4ClasseEcart(cur[:i], p.ts)]++
			}
		}
	}
	m.trous(cur)
}

// trous mesure, dans chaque suite de paquets fermes consecutifs, les ecarts entre deux presences
// d un meme index.
func (m *b4Mesure) trous(cur []b4Paquet) {
	for debut := 0; debut < len(cur); {
		if !cur[debut].fermee {
			debut++
			continue
		}
		fin := debut
		for fin < len(cur) && cur[fin].fermee {
			fin++
		}
		for k := uint(0); k < 32; k++ {
			dernier := -1
			for j := debut; j < fin; j++ {
				if cur[j].masque&(1<<k) == 0 {
					continue
				}
				m.c["presences dans les suites fermees"]++
				if dernier >= 0 && j-dernier > 1 {
					trou := j - dernier - 1
					m.c["trous (paquets) "+b4ClasseTrou(trou)]++
					m.c["trous (duree) "+b4ClasseDuree(cur[j].ts-cur[dernier].ts)]++
					m.c["absences entre deux presences"] += trou
					m.c["absences entre deux presences, trou < 0,1 s"] += b4Si(cur[j].ts-cur[dernier].ts < 100_000, trou)
					for g := dernier + 1; g < j; g++ {
						if cur[g].n == 0 {
							m.c["absences entre deux presences, vue C vide"]++
						}
					}
				}
				dernier = j
			}
		}
		debut = fin
	}
}

// b4Si rend v si la condition tient, 0 sinon.
func b4Si(ok bool, v int) int {
	if ok {
		return v
	}
	return 0
}

// ajouter somme une mesure dans une autre (agregat par build).
func (m *b4Mesure) ajouter(x *b4Mesure) {
	for k, v := range x.c {
		m.c[k] += v
	}
	for k, v := range x.dt {
		m.dt[k] += v
	}
}

// lignes rend les compteurs, puis l ecart d horodatage entre paquets delta consecutifs, arrondi a
// la milliseconde (au-dela de 200 ms : une seule classe).
func (m *b4Mesure) lignes(prefixe string) []string {
	var out []string
	cles := make([]string, 0, len(m.c))
	for k := range m.c {
		cles = append(cles, k)
	}
	sort.Strings(cles)
	for _, k := range cles {
		out = append(out, fmt.Sprintf("%s\t%s\t%d", prefixe, k, m.c[k]))
	}
	parMs := map[uint64]int{}
	for us, n := range m.dt {
		ms := (us + 500) / 1000
		if ms > 201 {
			ms = 201
		}
		parMs[ms] += n
	}
	for ms, n := range parMs {
		cle := fmt.Sprintf("ecart d horodatage %03d ms", ms)
		if ms > 200 {
			cle = "ecart d horodatage > 200 ms"
		}
		out = append(out, fmt.Sprintf("%s\t%s\t%d", prefixe, cle, n))
	}
	return out
}

// TestCampagneBis4Controle : les entrees de controle des paquets fermes, film par film.
func TestCampagneBis4Controle(t *testing.T) {
	racine, sortie, films := b2Env(t)
	utiles := cmUtiles(t)
	parFilm := []string{"film\tbuild\tmesure\tvaleur"}
	parBuild := map[string]*b4Mesure{}
	for _, id := range films {
		garde := filmproc.Arm("campagne/bis4-controle", 4, func(pic uint64) {
			fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
			os.Exit(3)
		})
		debut := time.Now()
		f, ok := cmOuvrir(t, racine, id, utiles)
		if !ok {
			garde.Disarm()
			continue
		}
		m := nouvelleB4Mesure()
		m.juge = cmNouveauJuge(f, cmLireBlocs(f), nil)
		rep, _, _ := cmMarcher(f, cmVariante{}, cmMux{m.juge, m})
		if rep.PaquetsFermes != m.c["paquets fermes"] || rep.Paquets != m.c["paquets delta"] {
			t.Errorf("%s : rapport %d / %d paquets fermes, sonde %d / %d", id, rep.PaquetsFermes, rep.Paquets,
				m.c["paquets fermes"], m.c["paquets delta"])
		}
		parFilm = append(parFilm, m.lignes(id+"\t"+f.build)...)
		if parBuild[f.build] == nil {
			parBuild[f.build] = nouvelleB4Mesure()
		}
		parBuild[f.build].ajouter(m)
		t.Logf("%s %s : %d paquets, %d fermes ; pic %d Mio, %s", id, f.build, m.c["paquets delta"],
			m.c["paquets fermes"], garde.Peak()>>20, time.Since(debut).Round(time.Second))
		garde.Disarm()
	}
	agg := []string{"build\tmesure\tvaleur"}
	corpus := nouvelleB4Mesure()
	for b, m := range parBuild {
		agg = append(agg, m.lignes(b)...)
		corpus.ajouter(m)
	}
	agg = append(agg, corpus.lignes("corpus")...)
	b2Ecrire(t, sortie, "mb4_controle.tsv", parFilm)
	b2Ecrire(t, sortie, "mb4_controle_par_build.tsv", agg)
}
