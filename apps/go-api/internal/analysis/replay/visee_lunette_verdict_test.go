package replay

// visee_lunette_verdict_test.go — LA MOITIE « RESTITUTION » de l'instrument de mesure de la
// lunette. L'instrument lui-meme, ses seuils et ses populations sont dans
// `visee_lunette_research_test.go` ; ce fichier ne porte que la journalisation, le verdict et le
// releve TSV.
//
// Il vit a part pour la meme raison que `visee_elevation_oracle_test.go` : reunies, les deux
// moities poussaient le fichier au-dela du seuil de 500 lignes du depot.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// adsJournalise ecrit les trois populations, drapeau par drapeau.
func adsJournalise(t *testing.T, pops ...adsPop) {
	t.Helper()
	t.Log("POPULATIONS — taux d'allumage de chaque drapeau d'i21")
	for _, p := range pops {
		r0, l0 := adsTaux(p.f0, p.n)
		r1, l1 := adsTaux(p.f1, p.n)
		r2, l2 := adsTaux(p.f2, p.nB)
		rB, lB := adsTaux(p.nB, p.n)
		ecart := 0.0
		if p.nB > 0 {
			ecart = p.ecartAB / float64(p.nB)
		}
		t.Logf("  %-34s fenetres=%d echantillons=%d", p.nom, p.fenetres, p.n)
		t.Logf("      flag0 %6.2f %% (%s) · flag1 %6.2f %% (%s) · second vecteur %6.2f %% (%s)",
			100*r0, l0, 100*r1, l1, 100*rB, lB)
		t.Logf("      flag2 %6.2f %% (%s, sur les seuls porteurs du second vecteur)"+
			" · ecart de cap A-B moyen %.2f deg", 100*r2, l2, ecart)
	}
}

// adsVerdict applique la regle ECRITE AVANT LA MESURE et publie un verdict par drapeau.
func adsVerdict(t *testing.T, mesure, temoin, fond adsPop, clos bool) {
	t.Helper()
	if mesure.fenetres < adsMinFenetres || mesure.n < adsMinEchantillons ||
		temoin.n < adsMinEchantillons {
		t.Logf("ORACLE — POPULATION INSUFFISANTE : %d fenetres de precision (seuil %d),"+
			" %d echantillons de mesure et %d de temoin (seuil %d chacun). L'oracle du fusil a"+
			" lunette ne conclut pas sur ce film.",
			mesure.fenetres, adsMinFenetres, mesure.n, temoin.n, adsMinEchantillons)
		if clos {
			t.Log("VERDICT — NEGATIF PUBLIE PAR LA CONSTANCE : les drapeaux d'i21 ne varient pas," +
				" donc aucun oracle ne peut les separer. Le film NE TRANSMET PAS l'etat de lunette" +
				" dans i21 ; conjugue au registre de replication (aucun composant de zoom sur aucun" +
				" archetype), l'epaulement vu dans Theater est reconstruit cote client.")
		}
		return
	}
	candidats := 0
	for _, d := range []struct {
		nom        string
		num, den   int
		numT, denT int
		numF, denF int
	}{
		{"flag0", mesure.f0, mesure.n, temoin.f0, temoin.n, fond.f0, fond.n},
		{"flag1", mesure.f1, mesure.n, temoin.f1, temoin.n, fond.f1, fond.n},
		{"flag2", mesure.f2, mesure.nB, temoin.f2, temoin.nB, fond.f2, fond.nB},
		{"second vecteur present", mesure.nB, mesure.n, temoin.nB, temoin.n, fond.nB, fond.n},
	} {
		rm, _ := adsTaux(d.num, d.den)
		rt, _ := adsTaux(d.numT, d.denT)
		rf, _ := adsTaux(d.numF, d.denF)
		facteurOK := rt > 0 && rm >= adsFacteurSeuil*rt
		ecartOK := rm-rt >= adsEcartSeuil
		verdict := "NEGATIF"
		if facteurOK && ecartOK {
			verdict = "CANDIDAT LUNETTE"
			candidats++
		}
		t.Logf("  %-24s precision %6.2f %% · temoin tir %6.2f %% · fond %6.2f %%"+
			" · facteur %s · ecart %s -> %s", d.nom, 100*rm, 100*rt, 100*rf,
			adsOuiNon(facteurOK), adsOuiNon(ecartOK), verdict)
	}
	if candidats == 0 {
		t.Log("VERDICT — NEGATIF PUBLIE : aucun des trois drapeaux d'i21 ni la presence du second" +
			" vecteur ne se distingue avant un kill au fusil a lunette. Conjugue au registre de" +
			" replication (aucun composant de zoom sur aucun archetype), le film NE TRANSMET PAS" +
			" l'etat de lunette : l'epaulement vu dans Theater est reconstruit cote client.")
		return
	}
	t.Logf("VERDICT — %d CANDIDAT(S) : la piste reste ouverte, il faut une seconde chaine"+
		" independante avant toute publication.", candidats)
}

func adsOuiNon(b bool) string {
	if b {
		return "OUI"
	}
	return "non"
}

// adsEcrisTSV depose les trois populations : la piece qui permet de refaire le calcul ailleurs
// sans re-decoder le film.
func adsEcrisTSV(t *testing.T, dir string, pops ...adsPop) {
	t.Helper()
	out := os.Getenv(adsTSVEnv)
	if out == "" {
		return
	}
	var sb strings.Builder
	sb.WriteString("population\tfenetres\tn\tflag0\tflag1\tn_second_vecteur\tflag2\tecart_cap_deg\n")
	for _, p := range pops {
		ecart := 0.0
		if p.nB > 0 {
			ecart = p.ecartAB / float64(p.nB)
		}
		fmt.Fprintf(&sb, "%s\t%d\t%d\t%d\t%d\t%d\t%d\t%.3f\n",
			p.nom, p.fenetres, p.n, p.f0, p.f1, p.nB, p.f2, ecart)
	}
	path := filepath.Join(out, fmt.Sprintf("visee_lunette_%s.tsv", filepath.Base(dir)))
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("ecriture du releve : %v", err)
	}
	t.Logf("RELEVE — %s", path)
}
