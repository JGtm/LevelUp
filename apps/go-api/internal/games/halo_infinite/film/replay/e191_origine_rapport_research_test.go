package replay

// e191_origine_rapport_research_test.go — LOT 1.9.1 : LES TABLEAUX DE LA MESURE.
//
// Le contexte, la table des films et l annotation d une pose vivent dans
// `e191_origine_mesure_research_test.go` ; ce fichier ne porte que les rendus.
//
// CE QUI EST COMPTE, ET DANS QUEL ORDRE :
//
//  1. par FAMILLE x ORIGINE x REGLE — l etat des lieux, ce que chaque regle decide aujourd hui ;
//  2. la DESIGNATION par le type 103 — ce que la grammaire couvre, par famille et par origine ;
//  3. les SIGNAUX ECRITS des appareils PORTES — mort du poseur, prise du poseur, les deux,
//     aucun — avec la DISTRIBUTION des ecarts, qui est ce qui fixera la tolerance ;
//  4. la CONFRONTATION signal ecrit contre fenetre de 200 ms : accord, contradiction, silence.
//
// LECTURE SEULE.

import (
	"fmt"
	"sort"
	"testing"
)

// e191SeuilsMS : les paliers de l histogramme des ecarts, en millisecondes. Ils COUVRENT trois
// ordres de grandeur parce que c est l echelle du sujet : les lachers a la mort tombent a
// 20-40 ms de la fin de vie et les deploiements a 14-42 SECONDES (mesure F.1).
var e191SeuilsMS = []float64{1, 10, 50, 100, 200, 500, 1000, 2000, 5000, 20000, 60000}

// e191Rapport ecrit les quatre tableaux d une population de poses.
func e191Rapport(t *testing.T, poses []e191Pose) {
	t.Helper()
	if len(poses) == 0 {
		t.Logf("  aucune pose")
		return
	}
	e191TableRegles(t, poses)
	e191TableDesignation(t, poses)
	e191TableSignaux(t, poses)
	e191TableConfrontation(t, poses)
	e191TableVerdict(t, poses)
	e191TableF1(t, poses)
}

// e191TableF1 — LES POSES REQUALIFIEES PAR F.1, REJUGEES UNE A UNE PAR LA GRAMMAIRE.
//
// L item F.1 du 2026-09-13 a retire la clause de DISTANCE d `equipmentOrigin` : 22 poses du
// corpus de l epoque (21 films, 5 363 poses) sont passees de `deployed` a `dropped`. Ce sont
// exactement les poses ou la regle d AVANT (`f1OrigineAvant`, le temoin conserve) differe de la
// regle de la fenetre seule. Cette table les retrouve sur le corpus de CE lot et dit, pour
// chacune, ce que la GRAMMAIRE en fait — et si elle est d accord avec F.1.
func e191TableF1(t *testing.T, poses []e191Pose) {
	t.Helper()
	n, accord, desaccord := 0, 0, 0
	t.Logf("  [6] poses REQUALIFIEES par F.1 (regle d avant != fenetre seule), rejugees par la grammaire")
	for _, p := range poses {
		if p.OrigineF1Avant == "" || p.OrigineF1Avant == p.Origine {
			continue
		}
		n++
		lu, prov := e191Cascade(p)
		verdict := "ACCORD avec F.1"
		if lu != p.Origine {
			verdict = "DESACCORD avec F.1"
			desaccord++
		} else {
			accord++
		}
		t.Logf("     %s %-20s %s ecart %6.1f ms distance %5.2f m : F.1 avant %-9s -> fenetre %-9s "+
			"| grammaire %-9s par %-22s %s",
			p.Film, p.Famille, p.ID, p.FenetreMS, p.DistM, p.OrigineF1Avant, p.Origine, lu, prov, verdict)
	}
	t.Logf("     TOTAL : %d pose(s) requalifiee(s) par F.1 — %d accord(s), %d desaccord(s)",
		n, accord, desaccord)
}

// Les TOLERANCES RETENUES, et ce qui les fixe — la mesure du 2026-09-15 sur 13 films, 4 583
// poses. Elles ne sont pas choisies, elles sont LUES dans la separation des populations :
//
//   - MORT ECRITE, 200 ms : les poses publiees `dropped` dont le poseur a une mort ecrite l ont
//     TOUTES a 171,7 ms au plus (min 5,6, p50 37,1) ; la plus proche mort d une pose publiee
//     `deployed` est a 205,3 ms. Aucun recouvrement, un intervalle vide de 33,6 ms entre les
//     deux ;
//   - PRISE ECRITE, 50 ms : 108 poses `deployed` portent un `taken` du poseur a 50 ms au plus
//     (dont 103 a MOINS D UNE MILLISECONDE : l emission est simultanee), contre UNE seule pose
//     `dropped`. A 100 ms la seconde population passe a 8 : la coupure est donc entre les deux ;
//   - DESIGNATION 103, 200 ms apres la creation : les 115 poses de panneau designees le sont
//     entre +32,2 et +70,2 ms, TOUTES POSITIVES (l evenement SUIT la creation), et aucun autre
//     objet du parc n est designe (0 sur 4 459).
const (
	e191TolMortMS  float64 = 200
	e191TolPriseMS float64 = 50
	e191TolSpawnMS float64 = 200
)

// e191TableVerdict : la cascade TELLE QU ELLE SERA BRANCHEE, aux tolerances retenues, confrontee
// a l origine publiee aujourd hui. C est le tableau qui dit ce que le lot change.
func e191TableVerdict(t *testing.T, poses []e191Pose) {
	t.Helper()
	m := map[string]int{}
	for _, p := range poses {
		o, prov := e191Cascade(p)
		bascule := "="
		if o != p.Origine {
			bascule = "BASCULE"
		}
		m[fmt.Sprintf("%-20s %-10s -> %-10s %-22s %s", p.Famille, p.Origine, o, prov, bascule)]++
	}
	t.Logf("  [5] VERDICT de la cascade (mort %g ms, prise %g ms, 103 %g ms) : origine publiee "+
		"AVANT le lot -> origine LUE / provenance ; BASCULE = ce que la decision du 2026-09-15 "+
		"change", e191TolMortMS, e191TolPriseMS, e191TolSpawnMS)
	e191LogTable(t, "     ", m)
}

// e191Cascade rejoue la cascade de PRODUCTION (`origineDeLaPose`) telle que la decision
// utilisateur du 2026-09-15 la fixe : `deployed` seulement pour ce qu'un 103 designe (plus la
// piece engendree en repli), `dropped` pour tout lacher d'appareil porte quelle qu'en soit la
// cause, `unknown` quand le film ne dit rien.
func e191Cascade(p e191Pose) (origine, provenance string) {
	switch {
	case p.SpawnDtMS >= 0 && p.SpawnDtMS <= e191TolSpawnMS:
		return OriginDeployed, "spawn_event"
	case e191PieceEngendree(p.ID):
		return OriginDeployed, "manifest_piece (repli)"
	case !p.AvecPoseur:
		return OriginUnknown, "no_owner"
	case p.MortMS <= e191TolMortMS && p.PriseMS <= e191TolPriseMS:
		return OriginDropped, "both (contradiction)"
	case p.MortMS <= e191TolMortMS:
		return OriginDropped, "death_written"
	case p.PriseMS <= e191TolPriseMS:
		return OriginDropped, "taken_written"
	}
	return OriginUnknown, "none"
}

// e191PieceEngendree : le predicat de production, appele par son nom pour que la mesure et la
// regle ne divergent pas.
func e191PieceEngendree(id string) bool { return equipmentIsSpawnedPiece(id) }

// e191TableRegles : famille x origine x regle — l etat des lieux.
func e191TableRegles(t *testing.T, poses []e191Pose) {
	t.Helper()
	m := map[string]int{}
	for _, p := range poses {
		nature := "deployable"
		if p.Portee {
			nature = "portee"
		}
		m[fmt.Sprintf("%-20s %-9s %-11s %-10s", p.Famille, nature, p.Origine, p.Regle)]++
	}
	t.Logf("  [1] famille / nature / origine publiee / regle qui decide")
	e191LogTable(t, "     ", m)
}

// e191TableDesignation : ce que le type 103 couvre, par OBJET et par ecart de temps.
//
// LA CLE (slot, generation) NE SUFFIT PAS : la generation fait 2 bits, donc elle reboucle. Cette
// table donne, par objet, la distribution des ecarts entre la creation et l evenement de meme
// cle le plus proche — c est elle qui dit ou la fenetre d appariement tombe.
func e191TableDesignation(t *testing.T, poses []e191Pose) {
	t.Helper()
	parObjet := map[string][]e191Pose{}
	for _, p := range poses {
		parObjet[fmt.Sprintf("%-20s %s", p.Famille, p.ID)] = append(
			parObjet[fmt.Sprintf("%-20s %s", p.Famille, p.ID)], p)
	}
	t.Logf("  [2] designation par le type 103 : ecart SIGNE creation -> evenement de meme cle")
	for _, k := range e191ClesTriees(parObjet) {
		l := parObjet[k]
		t.Logf("     %s n=%d %s", k, len(l),
			e191Histo(l, func(p e191Pose) float64 { return e191Abs(p.SpawnDtMS) }))
		t.Logf("       signes : %s", e191Signes(l))
	}
}

// e191Abs rend la valeur absolue en preservant [e191SansSignal].
func e191Abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// e191Signes compte les ecarts NEGATIFS (l evenement precede la creation) — un appariement juste
// n en porte aucun : le 103 SUIT la creation d une a deux images.
func e191Signes(poses []e191Pose) string {
	avant, apres, sans := 0, 0, 0
	for _, p := range poses {
		switch {
		case e191Abs(p.SpawnDtMS) >= e191SansSignal:
			sans++
		case p.SpawnDtMS < 0:
			avant++
		default:
			apres++
		}
	}
	return fmt.Sprintf("evenement AVANT la creation %d, APRES %d, aucun %d", avant, apres, sans)
}

// e191TableSignaux : pour les APPAREILS PORTES, ce que les deux signaux ecrits disent.
//
// LA TOLERANCE N EST PAS POSEE : chaque ligne porte l histogramme des ecarts, et c est lui qui
// dit ou la coupure tombe — ou s il n y en a pas.
func e191TableSignaux(t *testing.T, poses []e191Pose) {
	t.Helper()
	parOrigine := map[string][]e191Pose{}
	for _, p := range poses {
		if !p.Portee || !p.AvecPoseur {
			continue
		}
		parOrigine[p.Origine] = append(parOrigine[p.Origine], p)
	}
	t.Logf("  [3] appareils PORTES a poseur mesure : ecart au signal ECRIT le plus proche")
	for _, o := range e191ClesTriees(parOrigine) {
		l := parOrigine[o]
		t.Logf("     origine %-10s : %d poses", o, len(l))
		t.Logf("       mort ECRITE du poseur  %s", e191Histo(l, func(p e191Pose) float64 { return p.MortMS }))
		t.Logf("       prise ECRITE du poseur %s", e191Histo(l, func(p e191Pose) float64 { return p.PriseMS }))
		t.Logf("       fenetre de 200 ms      %s", e191Histo(l, func(p e191Pose) float64 { return p.FenetreMS }))
	}
}

// e191TableConfrontation : le signal ecrit contre la fenetre, a chaque tolerance candidate.
//
// ELLE EST LE CŒUR DE LA MESURE : elle dit, tolerance par tolerance, combien de poses PORTEES
// ont une mort ecrite, une prise ecrite, les DEUX (une contradiction a compter) ou AUCUNE (le
// repli), et comment ces quatre cas se repartissent sur l origine publiee aujourd hui.
func e191TableConfrontation(t *testing.T, poses []e191Pose) {
	t.Helper()
	t.Logf("  [4] appareils PORTES : cause ECRITE par tolerance, contre l origine publiee")
	for _, tol := range e191SeuilsMS {
		m := map[string]int{}
		for _, p := range poses {
			if !p.Portee || !p.AvecPoseur {
				continue
			}
			m[fmt.Sprintf("%-10s %s", p.Origine, e191Cause(p, tol))]++
		}
		if len(m) == 0 {
			continue
		}
		t.Logf("     tolerance %g ms", tol)
		e191LogTable(t, "       ", m)
	}
}

// e191Cause rend la cause ECRITE d une pose a une tolerance donnee.
func e191Cause(p e191Pose, tolMS float64) string {
	mort, prise := p.MortMS <= tolMS, p.PriseMS <= tolMS
	switch {
	case mort && prise:
		return "both (contradiction)"
	case mort:
		return "death_written"
	case prise:
		return "taken_written"
	}
	return "window_only (repli)"
}

// e191Histo rend l histogramme cumulatif d une grandeur : combien de poses sous chaque palier.
func e191Histo(poses []e191Pose, of func(e191Pose) float64) string {
	n := len(poses)
	if n == 0 {
		return "(aucune)"
	}
	var vals []float64
	sans := 0
	for _, p := range poses {
		v := of(p)
		if v >= e191SansSignal {
			sans++
			continue
		}
		vals = append(vals, v)
	}
	sort.Float64s(vals)
	out := fmt.Sprintf("n=%d sans signal=%d", n, sans)
	if len(vals) > 0 {
		out += fmt.Sprintf(" min=%.1f p50=%.1f p90=%.1f max=%.1f",
			vals[0], e191Quantile(vals, 0.5), e191Quantile(vals, 0.9), vals[len(vals)-1])
	}
	out += " | cumul :"
	for _, s := range e191SeuilsMS {
		k := sort.SearchFloat64s(vals, s+1e-9)
		out += fmt.Sprintf(" <=%gms:%d", s, k)
	}
	return out
}

// e191Quantile rend le quantile d une tranche DEJA TRIEE.
func e191Quantile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	i := int(q * float64(len(sorted)-1))
	return sorted[i]
}

// e191LogTable ecrit une table cle -> compte, triee par cle.
func e191LogTable(t *testing.T, indent string, m map[string]int) {
	t.Helper()
	if len(m) == 0 {
		t.Logf("%s(aucune)", indent)
		return
	}
	for _, k := range e191ClesTrieesInt(m) {
		t.Logf("%s%-60s %6d", indent, k, m[k])
	}
}

func e191ClesTrieesInt(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func e191ClesTriees(m map[string][]e191Pose) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
