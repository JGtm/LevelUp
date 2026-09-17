package main

// deux_passes_test.go — LES GARDES DU MODE S8, SANS UN OCTET DE FILM.

import (
	"strings"
	"testing"

	"levelup/go-api/internal/replaybuild"
)

// TestVerifierLaBrancheServie : LA GARDE ANTI-EQUIVALENCE-VACUANTE.
//
// Les deux sens comptent. Une passe `faits` qui aurait redecode en silence ferait comparer deux
// decodages — identiques par construction — et rendrait un vert qui ne prouve rien ; une passe
// `film` qui aurait relu les faits ne mesurerait pas le decodage.
func TestVerifierLaBrancheServie(t *testing.T) {
	etape := replaybuild.EtapeRejeuDepuisLesFaits
	cas := []struct {
		nom      string
		passe    string
		booleens map[string]bool
		refuse   bool
	}{
		{"hors mode S8, rien n est exige", "", nil, false},
		{"passe film, decodage servi", passeFilm, map[string]bool{etape: false}, false},
		{"passe faits, rejeu servi", passeFaits, map[string]bool{etape: true}, false},
		{"passe faits, REDECODAGE silencieux", passeFaits, map[string]bool{etape: false}, true},
		{"passe film, rejeu servi malgre le forcage", passeFilm, map[string]bool{etape: true}, true},
		{"etape de branche JAMAIS observee", passeFaits, map[string]bool{}, true},
	}
	for _, c := range cas {
		err := verifierLaBrancheServie(c.passe, c.booleens)
		if (err != nil) != c.refuse {
			t.Errorf("%s : erreur = %v, refus attendu = %v", c.nom, err, c.refuse)
		}
	}
}

// TestEtapeDeBrancheVientDeLaProduction : le nom n est pas recopie.
//
// Une copie du nom d etape ici se serait desynchronisee au premier renommage, et le harnais aurait
// alors compte la branche comme une divergence sur les vingt films — un faux positif sur tout le
// corpus, a chaque passe.
func TestEtapeDeBrancheVientDeLaProduction(t *testing.T) {
	if etapeDeBranche() != replaybuild.EtapeRejeuDepuisLesFaits {
		t.Fatalf("etapeDeBranche = %q, production %q", etapeDeBranche(),
			replaybuild.EtapeRejeuDepuisLesFaits)
	}
	if !contient(etapesAttendues(), etapeDeBranche()) {
		t.Errorf("l etape de branche %q n est pas dans les etapes attendues : le harnais ne la "+
			"verrait pas passer", etapeDeBranche())
	}
}

// TestEtapesDivergentesExclutLaBranche : la branche DOIT differer, et n est pas une divergence.
func TestEtapesDivergentesExclutLaBranche(t *testing.T) {
	memeChose := func(v string) map[string]string {
		out := map[string]string{}
		for _, e := range etapesAttendues() {
			out[e] = v
		}
		return out
	}
	a, b := memeChose("1 aaa"), memeChose("1 aaa")
	// La branche differe TOUJOURS entre les deux passes : c est son travail.
	b[etapeDeBranche()] = "1 bbb"
	if got := etapesDivergentes(a, b); len(got) != 0 {
		t.Errorf("l etape de branche est comptee comme divergence : %v", got)
	}
	b["killsource"] = "1 ccc"
	got := etapesDivergentes(a, b)
	if len(got) != 1 || got[0] != "killsource" {
		t.Errorf("etapesDivergentes = %v, attendu [killsource]", got)
	}
	delete(b, "zones")
	if got := etapesDivergentes(a, b); !contient(got, "zones(absente d une passe)") {
		t.Errorf("une etape ABSENTE d une passe doit se dire comme telle : %v", got)
	}
}

// TestDeuxPassesEtUpdateSontExclusifs : le mode S8 ne re-fige rien, et ne peut pas etre confondu.
//
// C est le point du mode : le regime ordinaire compare a une reference FIGEE, et un fige se
// regenere par accident (« -update pour voir »). Ici il n y a rien a regenerer, et la ligne de
// commande le refuse explicitement plutot que d ignorer le drapeau.
func TestDeuxPassesEtUpdateSontExclusifs(t *testing.T) {
	o := options{repoRoot: "C:/depot", deuxPasses: true, update: true}
	if err := valider(o); err == nil {
		t.Fatal("-deux-passes -update accepte : le mode S8 ne lit et n ecrit AUCUNE reference")
	} else if !strings.Contains(err.Error(), "exclusifs") {
		t.Errorf("le refus doit dire POURQUOI ; obtenu : %v", err)
	}
	if err := valider(options{repoRoot: "C:/depot", passe: "autre"}); err == nil {
		t.Error("une passe inconnue est acceptee : le harnais jouerait une branche au hasard")
	}
	for _, p := range []string{"", passeFilm, passeFaits} {
		if err := valider(options{repoRoot: "C:/depot", passe: p}); err != nil {
			t.Errorf("passe %q refusee : %v", p, err)
		}
	}
}

func contient(liste []string, v string) bool {
	for _, s := range liste {
		if s == v {
			return true
		}
	}
	return false
}
