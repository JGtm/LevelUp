package replay

// visee_lunette_balayage_verdict_test.go — LA MOITIE « RESTITUTION » du balayage. L'instrument,
// ses seuils et ses populations sont dans `visee_lunette_balayage_research_test.go` ; ce fichier
// ne porte que la journalisation, le verdict et le releve TSV.
//
// Il vit a part pour la meme raison que les autres moities de ce chantier : reunis, les deux
// fichiers depassaient le seuil de 500 lignes du depot.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// adsSweepLigne : le resultat d'UN index de composant, sur les trois tables.
type adsSweepLigne struct {
	index                             int
	presSans, presAvec                float64
	dernSans, dernAvec                float64
	temSans, temAvec                  float64
	candidatPresence, candidatDernier bool
	temoinContamine                   bool
}

// adsSweepTaux rend le taux d'une case, et 0 quand le denominateur est nul.
func adsSweepTaux(num, den int) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den)
}

// adsSepare applique la regle ECRITE AVANT LA MESURE : ecart >= adsSweepEcartSeuil ET facteur
// >= adsSweepFacteur, dans un sens ou dans l'autre.
func adsSepare(a, b float64) bool {
	hi, lo := a, b
	if hi < lo {
		hi, lo = lo, hi
	}
	if hi-lo < adsSweepEcartSeuil {
		return false
	}
	if lo == 0 {
		return hi >= adsSweepEcartSeuil
	}
	return hi/lo >= adsSweepFacteur
}

// adsSweepLignes calcule les 64 lignes du balayage.
func adsSweepLignes(b adsSweepBilan) []adsSweepLigne {
	out := make([]adsSweepLigne, 0, adsSweepMaxIndex)
	for i := 0; i < adsSweepMaxIndex; i++ {
		l := adsSweepLigne{index: i}
		l.presSans = adsSweepTaux(b.presence.vus[adsSansLunette][i], b.presence.instants[adsSansLunette])
		l.presAvec = adsSweepTaux(b.presence.vus[adsAvecLunette][i], b.presence.instants[adsAvecLunette])
		l.dernSans = adsSweepTaux(b.dernier.vus[adsSansLunette][i], b.dernier.instants[adsSansLunette])
		l.dernAvec = adsSweepTaux(b.dernier.vus[adsAvecLunette][i], b.dernier.instants[adsAvecLunette])
		l.temSans = adsSweepTaux(b.temoin.vus[adsSansLunette][i], b.temoin.instants[adsSansLunette])
		l.temAvec = adsSweepTaux(b.temoin.vus[adsAvecLunette][i], b.temoin.instants[adsAvecLunette])
		l.candidatPresence = adsSepare(l.presSans, l.presAvec)
		l.candidatDernier = adsSepare(l.dernSans, l.dernAvec)
		l.temoinContamine = adsSepare(l.temSans, l.temAvec)
		out = append(out, l)
	}
	return out
}

// adsSweepJournalise publie les denominateurs et les index qui bougent.
func adsSweepJournalise(t *testing.T, b adsSweepBilan) {
	t.Helper()
	t.Logf("POPULATIONS — presence : %d %s / %d %s", b.presence.instants[adsSansLunette],
		adsSansLunette, b.presence.instants[adsAvecLunette], adsAvecLunette)
	t.Logf("              dernier  : %d / %d · temoin : %d / %d",
		b.dernier.instants[adsSansLunette], b.dernier.instants[adsAvecLunette],
		b.temoin.instants[adsSansLunette], b.temoin.instants[adsAvecLunette])

	t.Log("BALAYAGE — frequence de presence au masque, index par index (sans / avec lunette)")
	for _, l := range adsSweepLignes(b) {
		if l.presSans == 0 && l.presAvec == 0 && l.dernSans == 0 && l.dernAvec == 0 {
			continue // index jamais vu : ne rien afficher plutot que 64 lignes de zeros
		}
		marque := "  "
		switch {
		case (l.candidatPresence || l.candidatDernier) && l.temoinContamine:
			marque = "~ " // separe, mais le temoin separe aussi : effet de population
		case l.candidatPresence || l.candidatDernier:
			marque = "**"
		}
		t.Logf("  %s i%-2d presence %6.2f / %6.2f · dernier %6.2f / %6.2f · temoin %6.2f / %6.2f",
			marque, l.index, 100*l.presSans, 100*l.presAvec, 100*l.dernSans, 100*l.dernAvec,
			100*l.temSans, 100*l.temAvec)
	}
}

// adsSweepVerdict applique la regle ecrite avant la mesure et conclut.
func adsSweepVerdict(t *testing.T, b adsSweepBilan) {
	t.Helper()
	rare := b.presence.instants[adsAvecLunette]
	if rare < adsSweepMinClasse {
		t.Logf("VERDICT — POPULATION INSUFFISANTE : %d instants « avec lunette » exploitables"+
			" (seuil %d). RIEN N'EST CONCLU : ni positif, ni negatif.", rare, adsSweepMinClasse)
		return
	}
	var retenus, ecartes []adsSweepLigne
	for _, l := range adsSweepLignes(b) {
		if !l.candidatPresence && !l.candidatDernier {
			continue
		}
		if l.temoinContamine {
			ecartes = append(ecartes, l)
			continue
		}
		retenus = append(retenus, l)
	}
	for _, l := range ecartes {
		t.Logf("  ECARTE i%d — il separe les deux classes MAIS le temoin les separe aussi"+
			" (%.2f / %.2f a -%d s) : c'est un effet de population, pas la lunette.",
			l.index, 100*l.temSans, 100*l.temAvec, adsSweepTemoinMS/1000)
	}
	if len(retenus) == 0 {
		t.Logf("VERDICT — NEGATIF MESURE. Sur %d instants « sans lunette » et %d « avec lunette »"+
			" etiquetes par le JEU, aucun des index de composant du bipede ne separe les deux"+
			" classes au seuil ecrit avant la mesure (ecart >= %.0f points ET facteur >= %.1f)."+
			" Le masque de replication du bipede ne porte donc pas l'etat de lunette.",
			b.presence.instants[adsSansLunette], rare, 100*adsSweepEcartSeuil, adsSweepFacteur)
		t.Log("  PORTEE EXACTE, et elle est etroite : ce negatif porte sur la PRESENCE d'un" +
			" composant au masque. Il n'exclut pas qu'un composant deja present porte l'etat dans" +
			" un de ses bits de charge utile — c'est le balayage suivant, au bit pres.")
		return
	}
	for _, l := range retenus {
		t.Logf("VERDICT — CANDIDAT i%d : presence %.2f %% sans lunette contre %.2f %% avec"+
			" (dernier record : %.2f / %.2f), et le temoin ne separe pas. A confirmer par une"+
			" seconde chaine avant toute publication.",
			l.index, 100*l.presSans, 100*l.presAvec, 100*l.dernSans, 100*l.dernAvec)
	}
}

// adsSweepTSV depose le tableau complet : la piece qui permet de refaire le calcul sans re-decoder
// les films.
func adsSweepTSV(t *testing.T, b adsSweepBilan) {
	t.Helper()
	out := os.Getenv(adsTSVEnv)
	if out == "" {
		return
	}
	var sb strings.Builder
	sb.WriteString("index\tpres_sans\tpres_avec\tdern_sans\tdern_avec\ttem_sans\ttem_avec\n")
	for _, l := range adsSweepLignes(b) {
		fmt.Fprintf(&sb, "%d\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\t%.4f\n", l.index,
			l.presSans, l.presAvec, l.dernSans, l.dernAvec, l.temSans, l.temAvec)
	}
	path := filepath.Join(out, "visee_lunette_balayage.tsv")
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("ecriture du balayage : %v", err)
	}
	t.Logf("RELEVE — %s", path)
}
