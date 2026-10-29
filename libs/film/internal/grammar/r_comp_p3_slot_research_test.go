//go:build research && campagne_overlay

package grammar

// r_comp_p3_slot_research_test.go — CHANTIER « comp », R-P3 : la CHRONIQUE d un slot. Pour un
// film et un slot (CAMPAGNE_SLOT=<film>:<slot>), ecrit dans l ordre du film :
//   - chaque declaration d image-cle du slot (chunk, archetype, tete lue) ;
//   - chaque record de la vue B qui vise le slot (genre, generation de l eid, archetype de la
//     liaison, masque, composants lus), et le sort du paquet (ferme / hors cadre / autre) ;
//   - l etat du slot dans le bloc de type 1 de chaque chunk (generation, vivante, pont).
// Instrument de recherche : aucun fichier de production touche.
//
//	CAMPAGNE_RACINE=... CAMPAGNE_SORTIE=... CAMPAGNE_FILMS=<id> CAMPAGNE_SLOT=<id>:<slot> \
//	  go test -tags=research -count=1 -run '^TestRCompP3ChroniqueDuSlot$' ...

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// rp3Chronique ecoute la marche et garde les records qui visent le slot.
type rp3Chronique struct {
	f      *cmFilm
	b      *cmBlocs
	slot   uint32
	lignes []string
	chunk  int
}

func (c *rp3Chronique) debutDeChunk(n int, data []byte, pks []FilmPacket, _ *World) {
	c.chunk = n
	marche := c.f.fc.MarcheDImageCle()
	for _, pk := range pks {
		if pk.Type != PacketTypeKeyframe {
			continue
		}
		recs := marche.Records(pk.Payload(data))
		for k, r := range recs {
			if r.Slot == int(c.slot) {
				c.lignes = append(c.lignes, fmt.Sprintf("%d\t%d\timage-cle\tti=%d tete=%d", n, pk.Index, r.TI, r.Gen))
			}
			// Les voisins dans l ORDRE DE LA MARCHE (slots croissants) : un record d un autre archetype
			// insere entre deux records du meme pool designe une ancre fortuite ou un decalage.
			if d := r.Slot - int(c.slot); d >= -3 && d <= 3 {
				suivant := -1
				if k+1 < len(recs) {
					suivant = recs[k+1].Bit
				}
				c.lignes = append(c.lignes, fmt.Sprintf("%d\t%d\timage-cle voisin\tslot %d ti=%d tete=%d bit %d (suivant %d)",
					n, pk.Index, r.Slot, r.TI, r.Gen, r.Bit, suivant))
			}
		}
	}
	e, ok := c.b.entree(n, c.slot)
	if ok {
		ti, st := c.b.pontDuBloc(e)
		c.lignes = append(c.lignes, fmt.Sprintf("%d\t-1\tbloc de type 1\tgen %d vivante %v pont %s ti=%d",
			n, e.Gen, e.Vivante(), st, ti))
	}
}

func (c *rp3Chronique) finDeFilm() {}

func (c *rp3Chronique) paquet(_ int, p *cmPaquet, _ *World) {
	issue := "hors cadre"
	switch {
	case p.d.Fermee:
		issue = "ferme"
	case p.d.Cause != CauseHorsCadre:
		issue = p.d.Cause
	}
	for _, n := range p.refuses {
		if n.slot == c.slot {
			c.lignes = append(c.lignes, fmt.Sprintf("%d\t%d\tNEW REFUSE\tneuf ti=%d sur slot vivant ti=%d · paquet %s",
				c.chunk, p.pk.Index, n.neuf, n.vivant, issue))
		}
	}
	for k, r := range p.recs {
		if r.Slot != c.slot {
			continue
		}
		dernier := ""
		if k == len(p.recs)-1 && p.d.Sortie == SortieVueBRejetHorsDatum {
			dernier = " · DERNIER RECORD AVANT UN REJET"
		}
		c.lignes = append(c.lignes, fmt.Sprintf("%d\t%d\t%s gen %d\tlie ti=%d masque %#x %s · paquet %s%s", c.chunk,
			p.pk.Index, b3Genre(r.Type), r.ID>>30, r.TypeIndex, r.Trace.Mask, rp3Comps(r), issue, dernier))
	}
}

// TestRCompP3ChroniqueDuSlot ecrit `r_comp_p3_chronique_<film>_<slot>.tsv`.
func TestRCompP3ChroniqueDuSlot(t *testing.T) {
	racine, sortie, _ := b2Env(t)
	id, s, ok := strings.Cut(os.Getenv("CAMPAGNE_SLOT"), ":")
	if !ok {
		t.Skip("CAMPAGNE_SLOT=<film>:<slot> requis")
	}
	slot, err := strconv.Atoi(s)
	if err != nil {
		t.Fatal(err)
	}
	f, ok := cmOuvrir(t, racine, id, cmUtiles(t))
	if !ok {
		return
	}
	c := &rp3Chronique{f: f, b: cmLireBlocs(f), slot: uint32(slot)} //nolint:gosec // slot < 8192
	cmMarcher(f, cmVariante{}, c)
	lignes := append([]string{"chunk\tpaquet\tgenre\tdetail"}, c.lignes...)
	brut := strings.Join(lignes, "\n") + "\n"
	nom := fmt.Sprintf("%s/r_comp_p3_chronique_%s_%d.tsv", sortie, id, slot)
	if err := os.WriteFile(nom, []byte(brut), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("%s slot %d : %d lignes", id, slot, len(c.lignes))
}
