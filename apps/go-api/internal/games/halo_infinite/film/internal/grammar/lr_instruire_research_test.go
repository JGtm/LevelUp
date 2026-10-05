//go:build research

package grammar

// lr_instruire_research_test.go — SONDE DU LOT LR : l instruction des paquets sains perdus. Pour
// chaque paquet de LR_PERDUS (un fichier : `film chunk paquet` par ligne), la sonde ecrit ce que la
// marche de la carte v2 ([cmMarcher]) y lit : le rang du debut, la sortie de la vue B et l eid
// rejete, la regle contredite, puis chaque record avec sa liaison et, pour un record lu sous une
// liaison posee par un NEW, l ORIGINE de cette liaison (le paquet, le bit et le rang du debut de la
// marche qui a lu ce NEW). Elle se compile sur la base du lot comme sur sa tete : jouee sur les deux,
// elle dit d ou venait ce que la base lisait ; elle ecrit aussi les DEL lus au second rang
// (lignes `D`). Un instrument : aucune sortie de production ne change.
//
//	LR_RACINE=<film_chunks> LR_SORTIE=<dir hors data> LR_PERDUS=<fichier> [LT_MPP=8/3] \
//	  go test -tags=research -count=1 -timeout 120m -run '^TestLRInstruire$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
)

// lrInstruction suit les liaisons posees par les NEW lus et decrit les paquets cibles.
type lrInstruction struct {
	lrEcouteur
	origines map[uint32]string
}

func (e *lrInstruction) paquet(c int, p *cmPaquet, _ *World) {
	rang := e.rang
	e.rang = lecture.DebutEnTete
	if e.cibles[[2]int{c, p.pk.Index}] {
		e.lignes = append(e.lignes, fmt.Sprintf("%s\t%d\t%d\tP\tdebut=%d rang=%d fermeAuBit=%v ferme=%v sortie=%v eidRejete=%d regle=%v",
			e.film, c, p.pk.Index, p.debut, rang, p.d.FermeeAuBit, p.d.Fermee, p.d.Sortie, p.d.EIDRejete&0x3fffffff, p.d.Invariant))
		for k, r := range p.recs {
			origine := ""
			if r.Type == recDelta && r.Liaison == lecture.LiaisonLueNeuf {
				origine = e.origines[r.Slot]
			}
			e.lignes = append(e.lignes, fmt.Sprintf("%s\t%d\t%d\tR\t%02d type=%d slot=%d ti=%d bit=%d desync=%d liaison=%d origine=%s",
				e.film, c, p.pk.Index, k, r.Type, r.Slot, r.TypeIndex, r.HeaderBit, r.DesyncAt, r.Liaison, origine))
		}
	}
	for _, r := range p.recs {
		switch {
		case r.Type == recNew && r.Liaison == lecture.LiaisonLueNeuf:
			e.origines[r.Slot] = fmt.Sprintf("%d:%d@%d/r%d/ti%d", c, p.pk.Index, r.HeaderBit, rang, r.TypeIndex)
		case r.Type == recDel:
			delete(e.origines, r.Slot)
			if rang == lecture.DebutParFermetureAuBit {
				e.lignes = append(e.lignes, fmt.Sprintf("%s\t%d\t%d\tD\tDEL second rang slot=%d bit=%d",
					e.film, c, p.pk.Index, r.Slot, r.HeaderBit))
			}
		}
	}
}

// lrLirePerdus lit LR_PERDUS, par film.
func lrLirePerdus(t *testing.T) map[string]map[[2]int]bool {
	t.Helper()
	brut, err := os.ReadFile(os.Getenv("LR_PERDUS"))
	if err != nil {
		t.Skip("LR_PERDUS requis")
	}
	out := map[string]map[[2]int]bool{}
	for _, l := range strings.Split(strings.TrimSpace(string(brut)), "\n") {
		c := strings.Split(strings.TrimSpace(l), "\t")
		if len(c) != 3 {
			continue
		}
		ch, err1 := strconv.Atoi(c[1])
		pq, err2 := strconv.Atoi(c[2])
		if err1 != nil || err2 != nil {
			t.Fatalf("LR_PERDUS : %q illisible", l)
		}
		if out[c[0]] == nil {
			out[c[0]] = map[[2]int]bool{}
		}
		out[c[0]][[2]int{ch, pq}] = true
	}
	return out
}

// TestLRInstruire ecrit `lr_instruction.tsv`.
func TestLRInstruire(t *testing.T) {
	parFilm := lrLirePerdus(t)
	racine, sortie := os.Getenv("LR_RACINE"), os.Getenv("LR_SORTIE")
	if racine == "" || sortie == "" {
		t.Skip("LR_RACINE et LR_SORTIE requis")
	}
	utiles := cmUtiles(t)
	var lignes []string
	for id, cibles := range parFilm {
		f, ok := ltOuvrir(t, racine, id, utiles)
		if !ok {
			continue
		}
		e := &lrInstruction{lrEcouteur: lrEcouteur{film: id, rang: lecture.DebutEnTete, cibles: cibles},
			origines: map[uint32]string{}}
		cmMarcher(f, cmVariante{tete: e.localiser}, e)
		lignes = append(lignes, e.lignes...)
	}
	if err := os.MkdirAll(sortie, 0o750); err != nil {
		t.Fatal(err)
	}
	ltEcrire(t, sortie, "lr_instruction.tsv", lignes)
}
