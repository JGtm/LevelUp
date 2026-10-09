package grammar

// keyframe_etats_fenetre_test.go — les fenetres de bits DERRIERE la lecture de l etat complet du
// bipede (D1.2 du plan `.ai/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md`) : un record admis ne
// passe jamais par une fenetre, un record non admis y passe, est compte et marque recupere.
//
// Mutations jouees le 2026-10-09, chacune rouge puis retiree : la fenetre rendue AVANT la lecture
// (sa valeur ecrase celle d un record admis) ; le compte d un repli oublie.

import (
	"fmt"
	"slices"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
)

// verifierLesRepliesDeLaBobine : chaque record non admis est donne aux trois fenetres et compte ;
// chaque record bipede rend un inventaire (la grammaire ou la fenetre) ; les dotations sont celles
// des admis plus celles que la fenetre trouve ; un record recupere n est jamais un admis.
func verifierLesRepliesDeLaBobine(t *testing.T, court string, e EtatsDesImagesCles) {
	t.Helper()
	a := e.Admission
	nonAdmis := a.Bipedes - a.Admis
	if a.FenetresArmes != nonAdmis || a.FenetresInventaire != nonAdmis || a.FenetresMarque != nonAdmis {
		t.Errorf("%s : replis %d / %d / %d, attendu %d records non admis chacun", court, a.FenetresArmes,
			a.FenetresInventaire, a.FenetresMarque, nonAdmis)
	}
	if len(e.Inventaire) != a.Bipedes {
		t.Errorf("%s : %d inventaires pour %d bipedes", court, len(e.Inventaire), a.Bipedes)
	}
	recuperes, armes := map[string]bool{}, 0
	for _, k := range e.Recuperes {
		recuperes[fmt.Sprintf("%d/%d", k.TimestampUS, k.Slot)] = true
		if k.Armes {
			armes++
		}
	}
	if len(e.Recuperes) > nonAdmis || len(e.Loadouts) != a.Admis+armes {
		t.Errorf("%s : %d recuperes (%d non admis), %d dotations pour %d admis et %d par la fenetre", court,
			len(e.Recuperes), nonAdmis, len(e.Loadouts), a.Admis, armes)
	}
}

// canalDeControle enveloppe le canal de production et compare, paquet par paquet, la dotation
// publiee d un record ADMIS a la lecture de la grammaire ; il compte les records admis dont la
// fenetre aurait rendu autre chose (le test ne mord que s il y en a).
type canalDeControle struct {
	*canalDeLEtatCompletBipede
	admis, ecartsFenetre, publieDifferent int
}

func (c *canalDeControle) ImageCle(p *lecture.Paquet, m *MarcheDistribuee) {
	avant := len(c.out.Loadouts)
	c.canalDeLEtatCompletBipede.ImageCle(p, m)
	publies := map[uint32][]uint32{}
	for _, l := range c.out.Loadouts[avant:] {
		publies[l.Slot] = l.Families
	}
	fenetre := map[uint32][]uint32{}
	for _, rf := range familiesByRecordRecs(p.Payload, p.Records, c.known, keyframeBipedTI) {
		fenetre[rf.Rec.Debut] = rf.Families
	}
	ctx := c.fc.ContexteDeLecture()
	for i := range p.Records {
		r := &p.Records[i]
		if int(r.TI) != keyframeBipedTI || r.Desync == lecture.CorpsNonParcouru {
			continue
		}
		l := lireLEtatComplet(p, r, c.arch, c.roles, ctx)
		if admettre(r, &l, c.dernierEmplacement) != refusAucun {
			continue
		}
		c.admis++
		lu := l.armesDe(r.Vie.Slot).Families
		if !slices.Equal(publies[r.Vie.Slot], lu) {
			c.publieDifferent++
		}
		if !slices.Equal(fenetre[r.Debut], lu) {
			c.ecartsFenetre++
		}
	}
}

// TestLaFenetreNeVoitQueLesRecordsNonAdmis : sur les bobines du golden, la dotation d un record
// admis est TOUJOURS celle de la grammaire, y compris la ou la fenetre lirait autre chose.
func TestLaFenetreNeVoitQueLesRecordsNonAdmis(t *testing.T) {
	total, mordants := 0, 0
	for _, court := range closureMiniFilms() {
		fc := contexteDeLaBobine(t, court)
		c := &canalDeControle{canalDeLEtatCompletBipede: nouveauCanalDeLEtatComplet(fc, catalogueDesFamilles(),
			DefaultGrenadeMax)}
		distribuerLesImagesClesSeules(fc, []Canal{c})
		if c.publieDifferent != 0 {
			t.Errorf("%s : %d record(s) admis publient autre chose que la lecture de la grammaire", court,
				c.publieDifferent)
		}
		total += c.admis
		mordants += c.ecartsFenetre
	}
	if total == 0 || mordants == 0 {
		t.Fatalf("%d records admis dont %d ou la fenetre lirait autre chose : le test ne mord plus", total, mordants)
	}
	t.Logf("%d records admis, %d ou la fenetre lirait autre chose", total, mordants)
}
