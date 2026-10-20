//go:build research

package grammar

// campagne_l8_research_test.go — SONDE DU LOT L8 (campagne de grammaire) : le detail record par
// record des paquets nommes, sous la marche de production ([cmMarcher]). Elle sert a instruire,
// paquet par paquet, ce que le portage de `ti=3` change : la meme sonde jouee sur deux arbres (ou
// sous `-overlay`) rend deux TSV que l on compare ligne a ligne.
//
//	CAMPAGNE_RACINE=<film_chunks> CAMPAGNE_SORTIE=<dossier hors data> \
//	CAMPAGNE_PAQUETS='fb1a1a72:7:92;1c4c63c2:24:510' \
//	go test -tags=research -run '^TestCampagneL8Paquets$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// l8Cible : un paquet nomme (film, chunk, index).
type l8Cible struct {
	film          string
	chunk, paquet int
}

// l8Cibles lit CAMPAGNE_PAQUETS (`film:chunk:paquet`, separes par `;`).
func l8Cibles(t *testing.T) []l8Cible {
	t.Helper()
	var out []l8Cible
	for _, x := range strings.Split(os.Getenv("CAMPAGNE_PAQUETS"), ";") {
		c := strings.Split(strings.TrimSpace(x), ":")
		if len(c) != 3 {
			continue
		}
		ch, err1 := strconv.Atoi(c[1])
		pq, err2 := strconv.Atoi(c[2])
		if err1 != nil || err2 != nil {
			t.Fatalf("CAMPAGNE_PAQUETS : %q illisible", x)
		}
		out = append(out, l8Cible{film: c[0], chunk: ch, paquet: pq})
	}
	return out
}

// l8Detail ecrit une ligne par record des paquets cibles d un film.
type l8Detail struct {
	film   string
	cibles map[[2]int]bool
	lignes []string
}

func (e *l8Detail) debutDeChunk(int, []byte, []FilmPacket, *World) {}
func (e *l8Detail) finDeFilm()                                     {}

func (e *l8Detail) paquet(c int, p *cmPaquet, _ *World) {
	if !e.cibles[[2]int{c, p.pk.Index}] {
		return
	}
	tete := fmt.Sprintf("%s\t%d\t%d\tdebut=%d fermeAuBit=%v ferme=%v cause=%q regle=%v",
		e.film, c, p.pk.Index, p.debut, p.d.FermeeAuBit, p.d.Fermee, p.d.Cause, p.d.Invariant)
	e.lignes = append(e.lignes, tete+"\t-")
	for k, r := range p.recs {
		var comps []string
		for _, cr := range r.Trace.Comps {
			comps = append(comps, fmt.Sprintf("i%d %s@%d porte=%v", cr.Index, cr.Name, cr.StartBit, cr.Ported))
		}
		e.lignes = append(e.lignes, fmt.Sprintf("%s\t%d\t%d\trec %02d type=%d slot=%d gen=%d ti=%d bit=%d fin=%d masque=%#x desync=%d masqueNonEcrit=%v\t%s",
			e.film, c, p.pk.Index, k, r.Type, r.Slot, r.ID>>30, r.TypeIndex, r.HeaderBit, r.Trace.EndBit,
			r.Trace.Mask, r.DesyncAt, r.Trace.MasqueNonEcrit, strings.Join(comps, " | ")))
	}
}

// TestCampagneL8Paquets ecrit `l8_paquets.tsv` : le detail des paquets cibles.
func TestCampagneL8Paquets(t *testing.T) {
	racine, sortie := os.Getenv("CAMPAGNE_RACINE"), os.Getenv("CAMPAGNE_SORTIE")
	cibles := l8Cibles(t)
	if racine == "" || sortie == "" || len(cibles) == 0 {
		t.Skip("CAMPAGNE_RACINE, CAMPAGNE_SORTIE et CAMPAGNE_PAQUETS requis")
	}
	parFilm := map[string]map[[2]int]bool{}
	var ordre []string
	for _, c := range cibles {
		if parFilm[c.film] == nil {
			parFilm[c.film] = map[[2]int]bool{}
			ordre = append(ordre, c.film)
		}
		parFilm[c.film][[2]int{c.chunk, c.paquet}] = true
	}
	utiles := cmUtiles(t)
	lignes := []string{"film\tchunk\tpaquet\trecord\tcomposants"}
	for _, id := range ordre {
		f, ok := cmOuvrir(t, racine, id, utiles)
		if !ok {
			continue
		}
		e := &l8Detail{film: id, cibles: parFilm[id]}
		cmMarcher(f, cmVariante{}, e)
		lignes = append(lignes, e.lignes...)
	}
	if err := os.MkdirAll(sortie, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sortie+"/l8_paquets.tsv", []byte(strings.Join(lignes, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}
