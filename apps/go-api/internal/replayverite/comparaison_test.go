package replayverite

import (
	"strings"
	"testing"
)

func comparerDocs(avant, apres *Document, reg RegistreReplis) Comparaison {
	return Comparer(Noter(avant, faitsJustes(), nil), Noter(apres, faitsJustes(), nil), reg)
}

func exigerConstat(t *testing.T, c Comparaison, prefixe string, sens Statut) Constat {
	t.Helper()
	for _, x := range c.Constats {
		if strings.HasPrefix(x.Mesure, prefixe) && x.Sens == sens {
			return x
		}
	}
	t.Fatalf("aucun constat %s %q dans %+v", sens, prefixe, c.Constats)
	return Constat{}
}

// TestComparer_IdentiqueEstOk : un meme artefact des deux cotes rend ok sans constat.
func TestComparer_IdentiqueEstOk(t *testing.T) {
	c := comparerDocs(documentJuste(), documentJuste(), nil)
	if c.Statut != StatutOK || len(c.Constats) != 0 {
		t.Fatalf("statut %s, constats %+v", c.Statut, c.Constats)
	}
}

// TestComparer_FauxPositifQuiMonteEstFaux : un kill en trop apres est FAUX, avec le joueur nomme.
func TestComparer_FauxPositifQuiMonteEstFaux(t *testing.T) {
	apres := documentJuste()
	apres.ScoreTimeline.Players[0].Kills = serie(Pas{0, 0}, Pas{50, 4})
	c := comparerDocs(documentJuste(), apres, nil)
	if c.Statut != StatutFaux {
		t.Fatalf("statut %s, veut FAUX", c.Statut)
	}
	x := exigerConstat(t, c, ScoreKills, StatutFaux)
	if len(x.Detail) != 1 || !strings.Contains(x.Detail[0], "111") {
		t.Errorf("detail %v, veut le joueur 111", x.Detail)
	}
}

// TestComparer_FauxNegatifQuiMonteEstManque : un kill perdu est MANQUE ; le rendre est un gain.
func TestComparer_FauxNegatifQuiMonteEstManque(t *testing.T) {
	apres := documentJuste()
	apres.ScoreTimeline.Players[0].Kills = serie(Pas{0, 0}, Pas{50, 2})
	c := comparerDocs(documentJuste(), apres, nil)
	if c.Statut != StatutManque {
		t.Fatalf("statut %s, veut MANQUE", c.Statut)
	}
	exigerConstat(t, c, ScoreKills, StatutManque)
	retour := comparerDocs(apres, documentJuste(), nil)
	if retour.Statut != StatutOK {
		t.Fatalf("le correctif doit rendre ok, rend %s", retour.Statut)
	}
	exigerConstat(t, retour, ScoreKills, sensGain)
}

// TestComparer_FauxPrimeSurManque : un temoin qui porte les deux est FAUX.
func TestComparer_FauxPrimeSurManque(t *testing.T) {
	apres := documentJuste()
	apres.ScoreTimeline.Players[0].Kills = serie(Pas{0, 0}, Pas{50, 2})
	apres.Tracks = append(apres.Tracks, piste(520, "111", 90, 120))
	if c := comparerDocs(documentJuste(), apres, nil); c.Statut != StatutFaux {
		t.Fatalf("statut %s, veut FAUX", c.Statut)
	}
	// L'ordre des constats ne decide pas : un MANQUE (preuve) constate APRES un FAUX (score).
	apres = documentJuste()
	apres.ScoreTimeline.Players[0].Kills = serie(Pas{0, 0}, Pas{50, 4})
	apres.Coverage.ContinuousFire.Closed--
	if c := comparerDocs(documentJuste(), apres, nil); c.Statut != StatutFaux {
		t.Fatalf("FAUX puis MANQUE : statut %s, veut FAUX", c.Statut)
	}
}

// TestComparer_ViolationNouvelleNommee : une instance nouvelle est FAUX et nommee ; une instance
// qui disparait est un gain.
func TestComparer_ViolationNouvelleNommee(t *testing.T) {
	apres := documentJuste()
	apres.Grenades = []Action{{T: 5, Slot: ptr(999)}}
	c := comparerDocs(documentJuste(), apres, nil)
	x := exigerConstat(t, c, ViolHorsVie, StatutFaux)
	if len(x.Detail) != 1 || x.Detail[0] != "+ grenade slot 999 @5" {
		t.Errorf("detail %v", x.Detail)
	}
	exigerConstat(t, comparerDocs(apres, documentJuste(), nil), ViolHorsVie, sensGain)
}

// TestComparer_PreuveDegradeeOuDisparueEstManque : moins de paquets fermes, un verdict qui descend,
// une preuve disparue, une contradiction de plus.
func TestComparer_PreuveDegradeeOuDisparueEstManque(t *testing.T) {
	cas := map[string]func(d *Document){
		PreuveFermeture: func(d *Document) { d.Coverage.ContinuousFire.Closed-- },
		PreuveVerdicts:  func(d *Document) { d.Coverage.Verdict["shots"] = "partiel : x" },
		PreuveContradic: func(d *Document) { d.Coverage.Keyframes.Refutations++ },
	}
	for id, abimer := range cas {
		apres := documentJuste()
		abimer(apres)
		c := comparerDocs(documentJuste(), apres, nil)
		if c.Statut != StatutManque {
			t.Errorf("%s : statut %s, veut MANQUE", id, c.Statut)
		}
	}
	apres := documentJuste()
	apres.Coverage.ContinuousFire = nil
	exigerConstat(t, comparerDocs(documentJuste(), apres, nil), PreuveFermeture, StatutManque)
}

// TestComparer_ScoreDisparuEstManque : un oracle qui n'est plus mesurable apres (calque perdu).
func TestComparer_ScoreDisparuEstManque(t *testing.T) {
	apres := documentJuste()
	apres.Coverage.Teams = nil
	exigerConstat(t, comparerDocs(documentJuste(), apres, nil), ScoreEquipes, StatutManque)
}

// TestComparer_RepliNouveauSelonLeRegistreDAvant : un repli nouveau est FAUX, sauf si le registre
// d'avant dit son compteur non branche ; une hausse de declenchements est informative.
func TestComparer_RepliNouveauSelonLeRegistreDAvant(t *testing.T) {
	avant, apres := documentJuste(), documentJuste()
	avant.Coverage.Fallbacks = []Repli{{Name: "repli_ancien", Hits: 1}}
	apres.Coverage.Fallbacks = []Repli{{Name: "repli_ancien", Hits: 5}, {Name: "repli_neuf", Hits: 2}}
	cas := []struct {
		nom  string
		reg  RegistreReplis
		veut Statut
	}{
		{"registre inconnu", nil, StatutFaux},
		{"compteur deja branche", RegistreReplis{"repli_neuf": true}, StatutFaux},
		{"absent du registre d'avant", RegistreReplis{"repli_ancien": true}, StatutFaux},
		{"compteur nouvellement branche", RegistreReplis{"repli_neuf": false}, StatutOK},
	}
	for _, k := range cas {
		if c := comparerDocs(avant, apres, k.reg); c.Statut != k.veut {
			t.Errorf("%s : statut %s, veut %s (%+v)", k.nom, c.Statut, k.veut, c.Constats)
		}
	}
	c := comparerDocs(avant, apres, RegistreReplis{"repli_neuf": false})
	exigerConstat(t, c, "R-1 repli repli_ancien", sensInfo)
}

// TestRendre_BloquantsDAbordEtDetailTronque : le rendu nomme le verdict, met les bloquants en tete
// et tronque un detail trop long.
func TestRendre_BloquantsDAbordEtDetailTronque(t *testing.T) {
	apres := documentJuste()
	for i := 0; i < detailMaxLignes+5; i++ {
		apres.Grenades = append(apres.Grenades, Action{T: 1000 + i, Slot: ptr(999)})
	}
	apres.ScoreTimeline.Players[0].Kills = serie(Pas{0, 0}, Pas{50, 2}) // un gain inverse : MANQUE
	var sb strings.Builder
	if err := Rendre(&sb, "temoin", comparerDocs(documentJuste(), apres, nil)); err != nil {
		t.Fatal(err)
	}
	txt := sb.String()
	if !strings.HasPrefix(txt, "BANC DE VERITE — temoin : FAUX\n  [FAUX] "+ViolHorsVie) {
		t.Errorf("tete du rendu :\n%s", txt)
	}
	if !strings.Contains(txt, "... 5 ligne(s) de plus") {
		t.Errorf("detail non tronque :\n%s", txt)
	}
	if strings.Index(txt, "[MANQUE]") < strings.Index(txt, "[FAUX]") {
		t.Errorf("les FAUX doivent preceder les MANQUE :\n%s", txt)
	}
}

// documentReattribue : le kill de B (222, qui en manquait un) passe a A (111, qui etait juste) —
// avant, A 3/3 et B 0/1 ; apres, A 2/3 et B 1/1. Les totaux FP/FN sont EGAUX des deux cotes.
func documentsReattribues() (avant, apres *Document) {
	avant, apres = documentJuste(), documentJuste()
	avant.ScoreTimeline.Players[1].Kills = serie(Pas{0, 0})
	apres.ScoreTimeline.Players[0].Kills = serie(Pas{0, 0}, Pas{50, 2})
	return avant, apres
}

// TestComparer_ReattributionEstVisible — revue finale P1-e (2026-10-02) : un kill qui change de
// joueur sans changer les totaux FP/FN laissait `ok` SANS AUCUN CONSTAT. On ne sait pas lequel est
// juste ; la reattribution doit etre VISIBLE : un constat non bloquant, joueur par joueur.
// Mutation vue rouge : ne plus rendre de constat quand seuls les ecarts par joueur bougent.
func TestComparer_ReattributionEstVisible(t *testing.T) {
	avant, apres := documentsReattribues()
	c := comparerDocs(avant, apres, nil)
	if c.Statut != StatutOK {
		t.Fatalf("statut %s : une reattribution n'est pas bloquante", c.Statut)
	}
	x := exigerConstat(t, c, ScoreKills, sensReattribution)
	if len(x.Detail) != 2 || !strings.Contains(x.Detail[0], "111") || !strings.Contains(x.Detail[1], "222") {
		t.Fatalf("detail %v, veut les deux joueurs nommes (111 puis 222)", x.Detail)
	}
	var b strings.Builder
	if err := Rendre(&b, "t", c); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "["+string(sensReattribution)+"] "+ScoreKills) ||
		!strings.Contains(b.String(), "222 : publie 0 / officiel 1 -> exact") {
		t.Fatalf("la reattribution n'est pas rendue au rapport texte :\n%s", b.String())
	}
}
