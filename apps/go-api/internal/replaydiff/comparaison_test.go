package replaydiff

// comparaison_test.go — LES DEUX INVARIANTS SANS LESQUELS L'OUTIL MENT.
//
// 1. Un champ ABSENT DE L'ANCIEN et present dans le nouveau est un GAIN. Sans cet invariant, un
//    balayage a travers quarante bumps de schema rendrait des milliers de fausses regressions
//    et le vrai signal serait noye.
// 2. Un champ PRESENT DANS L'ANCIEN et absent du nouveau est une PERTE — et il doit l'etre
//    MEME SI le code d'aujourd'hui ne declare plus ce champ. C'est la raison d'etre de la
//    lecture generique : deserialiser `replay.ReplayDocument` rendrait ce cas invisible.

import (
	"encoding/json"
	"strings"
	"testing"
)

// doc lit un artefact de test depuis son texte JSON.
func doc(t *testing.T, texte string) map[string]any {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(texte))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		t.Fatalf("fixture illisible : %v", err)
	}
	return m
}

// sensDe rend le sens de l'ecart sur une metrique, ou "" quand il n'y en a pas.
func sensDe(r Rapport, axe, metrique string) string {
	for _, d := range r.Differences {
		if d.Axe == axe && d.Metrique == metrique {
			return d.Sens
		}
	}
	return ""
}

// TestChampNeufEstUnGain — l'ancien ne connait pas `pickups` ; le nouveau en porte 3.
func TestChampNeufEstUnGain(t *testing.T) {
	ancien := doc(t, `{"schemaVersion":20,"matchId":"aaaaaaaa","tracks":[]}`)
	nouveau := doc(t, `{"schemaVersion":39,"matchId":"aaaaaaaa","tracks":[],
		"pickups":[{"t":1,"xuid":"1"},{"t":2,"xuid":"1"},{"t":3,"xuid":"2"}]}`)
	r := Comparer(Empreindre(ancien), Empreindre(nouveau))
	if got := sensDe(r, "armes", "pickups/n"); got != SensApparu {
		t.Fatalf("un calque neuf doit etre un gain, pas %q", got)
	}
	if got := sensDe(r, "armes", "pickups/par-xuid/1"); got != SensApparu {
		t.Fatalf("le compte par joueur d'un calque neuf doit etre un gain, pas %q", got)
	}
}

// TestChampDisparuEstUnePerte — l'ancien portait `objectives` ; le nouveau ne les porte plus.
//
// LE CHAMP `stat` "flag_captures" N'EXISTE NULLE PART DANS LE CODE DE CE PAQUET : c'est
// precisement le point. L'outil doit le voir disparaitre sans en avoir jamais entendu parler.
func TestChampDisparuEstUnePerte(t *testing.T) {
	ancien := doc(t, `{"schemaVersion":20,"matchId":"aaaaaaaa","tracks":[],
		"objectives":[{"t":1,"xuid":"7","stat":"flag_captures"},
		              {"t":2,"xuid":"7","stat":"flag_steals"}]}`)
	nouveau := doc(t, `{"schemaVersion":39,"matchId":"aaaaaaaa","tracks":[]}`)
	r := Comparer(Empreindre(ancien), Empreindre(nouveau))
	if got := sensDe(r, "objectifs", "objectives/n"); got != SensDisparu {
		t.Fatalf("un calque disparu doit etre une perte, pas %q", got)
	}
	if got := sensDe(r, "objectifs", "objectives/par-joueur/7/flag_captures"); got != SensDisparu {
		t.Fatalf("l'action perdue doit etre nommee joueur x famille, sens %q", got)
	}
}

// TestBaisseDeCompteEstUnePerte — le calque reste, il porte moins.
func TestBaisseDeCompteEstUnePerte(t *testing.T) {
	ancien := doc(t, `{"schemaVersion":20,"matchId":"a","grappleLines":[{"t0":1},{"t0":2},{"t0":3}]}`)
	nouveau := doc(t, `{"schemaVersion":39,"matchId":"a","grappleLines":[{"t0":1}]}`)
	r := Comparer(Empreindre(ancien), Empreindre(nouveau))
	if got := sensDe(r, "equipement", "grappleLines/n"); got != SensPerte {
		t.Fatalf("3 -> 1 doit etre une perte, pas %q", got)
	}
	if b := r.Bilans["equipement"]; b.Pertes == 0 {
		t.Fatalf("le bilan de l'axe doit compter la perte : %+v", b)
	}
	// LE SENS INVERSE EST TESTE ICI, ET IL DOIT L'ETRE : sans cette assertion, un outil qui
	// nommerait « perte » TOUTE variation passerait les autres cas de ce fichier.
	inverse := Comparer(Empreindre(nouveau), Empreindre(ancien))
	if got := sensDe(inverse, "equipement", "grappleLines/n"); got != SensGain {
		t.Fatalf("1 -> 3 doit etre un gain, pas %q", got)
	}
	if b := inverse.Bilans["equipement"]; b.Pertes != 0 || b.Gains == 0 {
		t.Fatalf("le bilan inverse doit compter un gain et zero perte : %+v", b)
	}
}

// TestZeroNEstPasUneDifference — une mesure nulle d'un cote et absente de l'autre dit la meme
// chose. Les distinguer remplirait le rapport de bruit a chaque champ optionnel.
func TestZeroNEstPasUneDifference(t *testing.T) {
	ancien := doc(t, `{"schemaVersion":38,"matchId":"a","shots":[]}`)
	nouveau := doc(t, `{"schemaVersion":39,"matchId":"a"}`)
	r := Comparer(Empreindre(ancien), Empreindre(nouveau))
	if got := sensDe(r, "armes", "shots/n"); got != "" {
		t.Fatalf("0 contre absent ne doit pas etre un ecart, sens %q", got)
	}
}

// TestDerniereValeurDeSerie — les compteurs de fin de match sont la DERNIERE valeur de la
// serie, l'axe `t` faisant foi pour l'ordre (un tableau relu d'un JSON n'est pas trie).
func TestDerniereValeurDeSerie(t *testing.T) {
	d := doc(t, `{"schemaVersion":39,"matchId":"a","scoreTimeline":{"players":[
		{"xuid":"42","kills":{"total":[{"t":9,"v":12},{"t":1,"v":3}]}}]}}`)
	e := Empreindre(d)
	m, ok := e.Mesures["joueurs/joueur/42/kills"]
	if !ok || !m.EstNum || m.Num != 12 {
		t.Fatalf("la derniere valeur de la serie doit etre 12, mesure=%+v presente=%v", m, ok)
	}
}

// TestToleranceSurLesFlottants — les bornes de carte sont des dequantifications : un ecart au
// dernier bit n'est pas une regression.
func TestToleranceSurLesFlottants(t *testing.T) {
	ancien := doc(t, `{"schemaVersion":38,"matchId":"a","bounds":{"maxX":43.79}}`)
	nouveau := doc(t, `{"schemaVersion":39,"matchId":"a","bounds":{"maxX":43.790000001}}`)
	r := Comparer(Empreindre(ancien), Empreindre(nouveau))
	if got := sensDe(r, "carte", "bounds.maxX"); got != "" {
		t.Fatalf("un ecart au dernier bit ne doit pas etre un ecart, sens %q", got)
	}
	loin := doc(t, `{"schemaVersion":39,"matchId":"a","bounds":{"maxX":1.5}}`)
	r2 := Comparer(Empreindre(ancien), Empreindre(loin))
	if got := sensDe(r2, "carte", "bounds.maxX"); got != SensPerte {
		t.Fatalf("217 -> 1,5 doit etre un ecart, sens %q", got)
	}
}
