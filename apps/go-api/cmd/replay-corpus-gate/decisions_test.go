package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/replaydiff"
	"levelup/go-api/internal/replayverite"
)

// TestUneDecisionAccepteLeFauxDeSesTemoins : une décision ne requalifie que sur ses témoins (tous
// quand la liste est vide), et compte ce qu'elle a servi.
func TestUneDecisionAccepteLeFauxDeSesTemoins(t *testing.T) {
	ds := []Decision{
		{Mesure: "R-1 repli repli_a", Temoins: []string{"aaaa0001"}, Decodeur: "v1.3.6", Date: "2026-11-13", Raison: "r"},
		{Mesure: "R-1 repli repli_b", Decodeur: "v1.3.7", Date: "2026-11-14", Raison: "r"},
		{Mesure: "R-1 repli repli_c", Decodeur: "v1.3.7", Date: "2026-11-14", Raison: "r"},
	}
	vues := decisionsAppliquees{}
	faux := func() *replayverite.Comparaison {
		return &replayverite.Comparaison{Statut: replayverite.StatutFaux, Constats: []replayverite.Constat{
			{Mesure: "R-1 repli repli_a", Sens: replayverite.StatutFaux},
			{Mesure: "R-1 repli repli_b", Sens: replayverite.StatutFaux},
		}}
	}
	sien, autre := faux(), faux()
	appliquerLesDecisions(sien, "aaaa0001", ds, vues)
	appliquerLesDecisions(autre, "bbbb0002", ds, vues)
	if sien.Statut != replayverite.StatutOK {
		t.Fatalf("temoin vise : statut %s, attendu ok", sien.Statut)
	}
	if autre.Statut != replayverite.StatutFaux || autre.Constats[0].Sens != replayverite.StatutFaux {
		t.Fatalf("autre temoin : la decision limitee a aaaa0001 ne doit pas s y appliquer")
	}
	if vues[0] != 1 || vues[1] != 2 || vues[2] != 0 {
		t.Fatalf("decisions servies : %v, attendu {0:1 1:2 2:0}", vues)
	}
}

// TestUneDecisionIncompleteEstRefusee : le manifeste refuse une décision sans raison.
func TestUneDecisionIncompleteEstRefusee(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "m.toml")
	toml := "[meta]\ntitle_slug = \"halo_infinite\"\n[[temoin]]\nid = \"aaaa0001\"\nfamille = \"f\"\n" +
		"[[decision]]\nmesure = \"R-1 repli repli_a\"\ndecodeur = \"v1.3.7\"\ndate = \"2026-11-14\"\n"
	if err := os.WriteFile(path, []byte(toml), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadManifest(path); err == nil || !strings.Contains(err.Error(), "decision #0") {
		t.Fatalf("decision sans raison acceptee : %v", err)
	}
}

// TestUneDecisionAccepteUnFiletDeSaFeuille : le filet de la feuille nommée sort des pertes bloquantes
// et reste au rapport ; un autre filet reste bloquant.
func TestUneDecisionAccepteUnFiletDeSaFeuille(t *testing.T) {
	ds := []Decision{{Mesure: "coverage.shots.aim.record", Decodeur: "v1.3.7", Date: "2026-11-14", Raison: "r"}}
	vues := decisionsAppliquees{}
	l := ligneRapport{Temoin: Temoin{ID: "aaaa0001"}, Filets: []replaydiff.Difference{
		{Metrique: "coverage.shots.aim.record"}, {Metrique: "coverage.shots.noSlot"},
	}}
	accepterLesFilets(&l, ds, vues)
	if len(l.Filets) != 1 || l.Filets[0].Metrique != "coverage.shots.noSlot" {
		t.Fatalf("filets bloquants restants : %+v", l.Filets)
	}
	if len(l.FiletsAcceptes) != 1 || vues[0] != 1 {
		t.Fatalf("filet accepte : %+v, servi %d", l.FiletsAcceptes, vues[0])
	}
	if l.statutVerite() != statutPerte {
		t.Fatalf("un filet non accepte reste une PERTE")
	}
}

// TestLesDecisionsSansEffetNeSeJugentQueSurUneExecutionComplete : manifeste filtré, témoin absent ou
// en erreur : pas de verdict « sans effet ».
func TestLesDecisionsSansEffetNeSeJugentQueSurUneExecutionComplete(t *testing.T) {
	m := Manifest{Temoins: []Temoin{{ID: "a"}, {ID: "b"}}}
	ok := []ligneRapport{{Temoin: Temoin{ID: "a"}}, {Temoin: Temoin{ID: "b"}}}
	if !executionComplete(ok, m, false) {
		t.Fatal("execution complete non reconnue")
	}
	if executionComplete(ok, m, true) || executionComplete(ok[:1], m, false) {
		t.Fatal("un manifeste filtre ou ampute n est pas une execution complete")
	}
	absent := []ligneRapport{{Temoin: Temoin{ID: "a"}, Absent: true}, {Temoin: Temoin{ID: "b"}}}
	if executionComplete(absent, m, false) {
		t.Fatal("un temoin absent rend l execution incomplete")
	}
}

// TestUneCleInconnueOuUnTemoinHorsManifesteEstRefuse : le manifeste refuse une clé mal orthographiée et
// une décision qui nomme un témoin qu'il ne porte pas.
func TestUneCleInconnueOuUnTemoinHorsManifesteEstRefuse(t *testing.T) {
	tete := "[meta]\ntitle_slug = \"halo_infinite\"\n[[temoin]]\nid = \"aaaa0001\"\nfamille = \"f\"\n" +
		"[[decision]]\nmesure = \"m\"\ndecodeur = \"v1.3.7\"\ndate = \"2026-11-14\"\nraison = \"r\"\n"
	for nom, suite := range map[string]string{
		"cle inconnue":       "temoin = [\"aaaa0001\"]\n",
		"temoin hors corpus": "temoins = [\"zzzz9999\"]\n",
	} {
		path := filepath.Join(t.TempDir(), "m.toml")
		if err := os.WriteFile(path, []byte(tete+suite), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadManifest(path); err == nil {
			t.Fatalf("%s : manifeste accepte", nom)
		}
	}
}

// TestLeNombreDeReplisSuitSesReplisAcceptes : une hausse de `coverage.fallbacks/n` égale au nombre de
// replis nouveaux acceptés est acceptée ; une hausse plus grande reste bloquante.
func TestLeNombreDeReplisSuitSesReplisAcceptes(t *testing.T) {
	verite := &replayverite.Comparaison{Constats: []replayverite.Constat{
		{Mesure: "R-1 repli repli_a", Sens: replayverite.StatutAccepte, Avant: "0", Apres: "3"},
	}}
	for _, c := range []struct {
		apres  string
		accept bool
	}{{"32", true}, {"33", false}} {
		l := ligneRapport{Temoin: Temoin{ID: "x"}, Verite: verite,
			Filets: []replaydiff.Difference{{Metrique: metriqueNombreDeReplis, Ancien: "31", Nouveau: c.apres}}}
		accepterLesFilets(&l, nil, decisionsAppliquees{})
		if got := len(l.FiletsAcceptes) == 1; got != c.accept {
			t.Fatalf("31 -> %s avec un repli nouveau accepte : accepte=%v, attendu %v", c.apres, got, c.accept)
		}
	}
}
