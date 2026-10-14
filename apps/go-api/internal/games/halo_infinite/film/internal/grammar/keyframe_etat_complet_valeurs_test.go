package grammar

// keyframe_etat_complet_valeurs_test.go — LES VALEURS QUE LA LECTURE DE L ETAT COMPLET DU BIPEDE
// PUBLIE, SUR DES RECORDS REELS (revue D1.4.6, constat 4).
//
// Les bobines versionnees du golden (`replay/testdata/minifilm_*`, registre et images-cles reelles)
// sont lues en contexte de cuisson ; pour un echantillon fixe de records ADMIS, chaque valeur passe
// par [lireLEtatComplet] puis par la publication ([lectureDEtatComplet.inventaireDe],
// [lectureDEtatComplet.armesDe], [lectureDEtatComplet.porteLaMarque]) et se compare au golden
// `testdata/etat_complet_bobines.golden` : compteurs i22, chargeurs, jauges, reserves, surchauffes,
// jeu d armes i42 et emplacement desire publie, masque et selection d i47 (brute en base 1, publiee
// en base 0), rang de capacite, armes, configuration de la marque de portage.
//
// L ECHANTILLON : sur chaque bobine, les 12 premiers records admis, plus chaque record admis qui
// porte une valeur rare (marque de portage, seconde main, rang de capacite publie, surchauffe ou
// jauge lue) — 8 au plus par bobine. Deux invariants independants du golden sont tenus en plus : le masque d i47 egale la bitmap
// des compteurs (T2) et une selection publiee designe un type du masque.
//
// REGENERATION : `go test ./internal/games/halo_infinite/film/internal/grammar/ -run
// TestLesValeursDeLEtatCompletDesBobines -update-etat-complet-bobines` (ligne d historique ci-dessous).
//
// HISTORIQUE : 2026-10-09, creation (revue D1.4.6). Mutations jouees le 2026-10-09, chacune rouge
// puis retiree : les roles du chargeur et de la reserve permutes dans [nomsDesRoles] ; la largeur des
// drapeaux de vitalites (R(5)) lue a 4 bits dans [lectureDEtatComplet.lireLaConfiguration] ;
// l echelle lue sur 2 bits.

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

var updateEtatCompletBobines = flag.Bool("update-etat-complet-bobines", false,
	"reecrire testdata/etat_complet_bobines.golden depuis les bobines versionnees")

const (
	premiersAdmisParBobine = 12
	rarsParBobine          = 8
)

// bilanDeBobine est ce que la marche de controle d une bobine releve : l echantillon des valeurs
// ([TestLesValeursDeLEtatCompletDesBobines]), les invariants violes, et le controle des dotations des
// records admis contre la grammaire et contre la fenetre ([TestLaFenetreNeVoitQueLesRecordsNonAdmis]).
type bilanDeBobine struct {
	lignes, ecarts                        []string
	admis, ecartsFenetre, publieDifferent int
}

// LA MARCHE DE CONTROLE EST MEMORISEE PAR BOBINE et partagee par les deux tests : sous la couverture
// de la CI, chaque rebalayage des sept bobines coute 70 a 80 s (`.github/workflows/ci.yml`).
var (
	memoBilansMu sync.Mutex
	memoBilans   = map[string]bilanDeBobine{}
)

// bilanDeLaBobine rend le bilan de la marche de controle de la bobine `court`, calcule une fois.
func bilanDeLaBobine(t *testing.T, court string) bilanDeBobine {
	t.Helper()
	memoBilansMu.Lock()
	defer memoBilansMu.Unlock()
	if b, ok := memoBilans[court]; ok {
		return b
	}
	fc := contexteDeLaBobine(t, court)
	c := &canalDesValeurs{canalDeLEtatCompletBipede: nouveauCanalDeLEtatComplet(fc, catalogueDesFamilles(),
		DefaultGrenadeMax), court: court}
	distribuerLesImagesClesSeules(fc, []Canal{c})
	memoBilans[court] = c.b
	return c.b
}

// canalDesValeurs enveloppe le canal de production : pour chaque record ADMIS, il compare la dotation
// publiee a la lecture de la grammaire et a ce que la fenetre lirait, et il rend la ligne de valeurs
// de l echantillon.
type canalDesValeurs struct {
	*canalDeLEtatCompletBipede
	court          string
	premiers, rars int
	b              bilanDeBobine
}

func (c *canalDesValeurs) ImageCle(p *lecture.Paquet, m *MarcheDistribuee) {
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
		c.b.admis++
		lu := l.armesDe(r.Vie.Slot).Families
		if !slices.Equal(publies[r.Vie.Slot], lu) {
			c.b.publieDifferent++
		}
		if !slices.Equal(fenetre[r.Debut], lu) {
			c.b.ecartsFenetre++
		}
		inv, tu := l.inventaireDe(r.Vie.Slot)
		rare := l.porteLaMarque() || l.jeu.Second >= 0 || inv.AbilityRank >= 0 || l.porteUneSurchauffeOuUneJauge()
		switch {
		case c.premiers < premiersAdmisParBobine:
			c.premiers++
		case rare && c.rars < rarsParBobine:
			c.rars++
		default:
			continue
		}
		c.verifierLesInvariants(p, r, &l, inv.SelectedGrenadeRank)
		c.b.lignes = append(c.b.lignes, ligneDesValeurs(c.court, p.TS, r.Vie.Slot, &l, inv, tu))
	}
}

// verifierLesInvariants tient les deux regles independantes du golden.
func (c *canalDesValeurs) verifierLesInvariants(p *lecture.Paquet, r *lecture.Record, l *lectureDEtatComplet, sel int) {
	if l.masque != bitmapDesCompteurs(l.compteurs) {
		c.b.ecarts = append(c.b.ecarts, fmt.Sprintf("%s %d/%d : masque %04b, compteurs %v", c.court, p.TS, r.Vie.Slot,
			l.masque, l.compteurs))
	}
	if sel >= 0 && l.masque&(1<<uint(sel)) == 0 { //nolint:gosec // 0..3
		c.b.ecarts = append(c.b.ecarts, fmt.Sprintf("%s %d/%d : selection publiee %d hors du masque %04b", c.court,
			p.TS, r.Vie.Slot, sel, l.masque))
	}
}

// ligneDesValeurs rend une ligne : ce que la grammaire a lu, puis ce qu elle publie.
func ligneDesValeurs(court string, ts uint64, slot uint32, l *lectureDEtatComplet,
	inv types.KeyframeInventory, tu refusDInventaire) string {
	var mun []string
	for k := range emplacementsDArme {
		c, j := "-", "-"
		if l.aChargeur[k] {
			c = fmt.Sprint(l.chargeur[k])
		}
		if l.aJauge[k] {
			j = fmt.Sprint(l.jauge[k])
		}
		mun = append(mun, fmt.Sprintf("%s/%s/%d/%d.%d", c, j, l.reserve[k], l.surchauffe[k], l.drapeaux[k]))
	}
	var armes []string
	for k := range emplacementsDArme {
		armes = append(armes, fmt.Sprintf("%08x", l.armeHaute[k]))
	}
	return fmt.Sprintf("%s\t%d\t%d\ti22=%v\tmun=%s\ti42=%d,%d,%d\ti47=%04b,%d\ti48=%d\tarmes=%s\tmarque=%v,%v,%d\t"+
		"publie=%s\ttu=%v,%v", court, ts, slot, l.compteurs, strings.Join(mun, " "), l.jeu.Demande, l.jeu.Principal,
		l.jeu.Second, l.masque, l.selection, l.rang, strings.Join(armes, ","), l.mortParDefaut, l.echelleAUn,
		l.vitalites, publication(inv), tu.capaciteHorsDomaine, tu.selectionHorsMasque)
}

// TestLesValeursDeLEtatCompletDesBobines : les valeurs lues et publiees de l echantillon egalent le
// golden.
func TestLesValeursDeLEtatCompletDesBobines(t *testing.T) {
	var lignes []string
	for _, court := range closureMiniFilms() {
		b := bilanDeLaBobine(t, court)
		for _, e := range b.ecarts {
			t.Error(e)
		}
		if len(b.lignes) == 0 {
			t.Fatalf("%s : aucun record admis dans l echantillon", court)
		}
		lignes = append(lignes, b.lignes...)
	}
	got := strings.Join(lignes, "\n") + "\n"
	chemin := filepath.Join("testdata", "etat_complet_bobines.golden")
	if *updateEtatCompletBobines {
		if err := os.WriteFile(chemin, []byte(got), 0o600); err != nil {
			t.Fatalf("ecriture : %v", err)
		}
		t.Fatalf("golden reecrit : %s (%d lignes) ; relancer sans -update-etat-complet-bobines", chemin, len(lignes))
	}
	want, err := os.ReadFile(chemin) //nolint:gosec // chemin fige
	if err != nil {
		t.Fatalf("golden absent : %v", err)
	}
	if string(want) == got {
		return
	}
	lw, lg := strings.Split(string(want), "\n"), strings.Split(got, "\n")
	for i := 0; i < len(lw) || i < len(lg); i++ {
		var a, b string
		if i < len(lw) {
			a = lw[i]
		}
		if i < len(lg) {
			b = lg[i]
		}
		if a != b {
			t.Fatalf("valeurs de l etat complet differentes du golden, ligne %d :\n  golden : %s\n  lu     : %s", i+1, a, b)
		}
	}
}

// publication rend ce que l inventaire publie : compteurs, selection (base 0), emplacement desire,
// rang de capacite, puis chargeur / jauge / reserve des quatre emplacements.
func publication(inv types.KeyframeInventory) string {
	var mun []string
	for _, a := range inv.Ammo {
		c, j, r := "-", "-", "-"
		if a.Mag != nil {
			c = fmt.Sprint(*a.Mag)
		}
		if a.Gauge != nil {
			j = fmt.Sprintf("%.4f", *a.Gauge)
		}
		if a.Res != nil {
			r = fmt.Sprint(*a.Res)
		}
		mun = append(mun, c+"/"+j+"/"+r)
	}
	return fmt.Sprintf("g=%v,%v;sel=%d;d=%d;cap=%d;am=%v,%s", inv.Grenades, inv.GrenadesRead, inv.SelectedGrenadeRank,
		inv.DrawnSlot, inv.AbilityRank, inv.AmmoRead, strings.Join(mun, " "))
}

// porteUneSurchauffeOuUneJauge dit si un emplacement porte une surchauffe non nulle ou une jauge lue.
func (l *lectureDEtatComplet) porteUneSurchauffeOuUneJauge() bool {
	for k := range emplacementsDArme {
		if l.surchauffe[k] != 0 || l.drapeaux[k] != 0 || l.aJauge[k] {
			return true
		}
	}
	return false
}
