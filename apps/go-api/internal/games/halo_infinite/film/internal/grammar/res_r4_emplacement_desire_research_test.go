//go:build research

package grammar

// res_r4_emplacement_desire_research_test.go — LE SENS DE LA VALEUR 2 DE L EMPLACEMENT DESIRE (plan
// `.ai/V7.5/PLAN_RI_RESIDU_IMAGES_CLES_2026-10-14.md`, R4.1 ; D-30 du plan clos).
//
// Pour chaque record bipede d image-cle lu, une ligne : film, rang de l image-cle
// dans le film (0 = la premiere), horodatage, slot, admis, emplacement desire (param[1], -9 : i42 non
// lu) et seconde main (param[2]), emplacement que la FENETRE publie (-9 : aucune lecture de fenetre),
// puis l occupation des quatre emplacements d arme (famille haute ou « vide »). La lecture est
// celle de la production (`lireLEtatComplet`, regle d admission `admettre`).
//
//	RI27C_FILMS=<id,...> RI27C_RACINE=<film_chunks> RI27C_OUT=<dossier> RI27C_CARTES=<id=Carte;...> \
//	  go test -tags=research -count=1 -run '^TestR4EmplacementDesire$' ./...grammar/
//
// Sortie : RI27C_OUT/emplacement_desire.tsv.

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/weaponv3"
)

// r4Canal relit les records bipedes d une phase des images-cles.
type r4Canal struct {
	*canalDeLEtatCompletBipede
	court  string
	rang   int
	lignes []string
}

func (c *r4Canal) ImageCle(p *lecture.Paquet, _ *MarcheDistribuee) {
	ctx := c.fc.ContexteDeLecture()
	fen := map[uint32]int{} // slot -> emplacement de la fenetre (DrawnSlot), -1 absent
	for _, inv := range keyframeInventoriesDe(p.Payload, invRecordSpansDe(p.Payload, p.Records), c.known, DefaultGrenadeMax) {
		fen[inv.Slot] = inv.DrawnSlot
	}
	for i := range p.Records {
		r := &p.Records[i]
		if int(r.TI) != keyframeBipedTI || r.Desync == lecture.CorpsNonParcouru || !c.lisible {
			continue
		}
		l := lireLEtatComplet(p, r, c.arch, c.roles, ctx)
		admis := admettre(r, &l, c.dernierEmplacement) == refusAucun
		principal, second := -9, -9
		if l.jeuLu {
			principal, second = l.jeu.Principal, l.jeu.Second
		}
		df, okf := fen[r.Vie.Slot]
		if !okf {
			df = -9
		}
		occ := make([]string, 0, emplacementsDArme)
		for k := range emplacementsDArme {
			if l.armeLue[k] && l.armeHaute[k] != noVariant {
				occ = append(occ, fmt.Sprintf("%08x", l.armeHaute[k]))
			} else {
				occ = append(occ, "vide")
			}
		}
		c.lignes = append(c.lignes, fmt.Sprintf("D\t%s\t%d\t%d\t%d\t%v\t%d\t%d\t%d\t%s", c.court, c.rang, p.TS,
			r.Vie.Slot, admis, principal, second, df, strings.Join(occ, ",")))
	}
	c.rang++
}

func TestR4EmplacementDesire(t *testing.T) {
	films, racine, sortie := ri27cEnv(t)
	var lignes []string
	for _, court := range films {
		fc := ri27cContexte(t, filepath.Join(racine, court), court, true)
		c := &r4Canal{canalDeLEtatCompletBipede: nouveauCanalDeLEtatComplet(fc, weaponv3.FamillesConnues(),
			DefaultGrenadeMax), court: court}
		distribuerLesImagesClesSeules(fc, []Canal{c})
		lignes = append(lignes, c.lignes...)
		t.Logf("%s : %d records bipedes lus", court, len(c.lignes))
	}
	ri27cEcrire(t, filepath.Join(sortie, "emplacement_desire.tsv"), lignes)
}
