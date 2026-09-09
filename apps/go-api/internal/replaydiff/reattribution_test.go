package replaydiff

import (
	"strings"
	"testing"
)

// Une mesure par joueur dont la SOMME sur tous les joueurs est conservee a change de main, pas
// de valeur : reattribution = changement, ni gain ni perte. Une somme qui baisse reste une perte.
// Mutation : retirer le cas `estReattribution` dans `ajouter` -> le premier cas rougit.
func TestComparer_ReattributionASommeConservee(t *testing.T) {
	num := func(v float64) Mesure { return Mesure{EstNum: true, Num: v} }
	kA := cle("armes", "pickups/par-xuid/111")
	kB := cle("armes", "pickups/par-xuid/222")
	rap := Comparer(
		Empreinte{Mesures: map[string]Mesure{kA: num(10), kB: num(5)}},
		Empreinte{Mesures: map[string]Mesure{kA: num(8), kB: num(7)}},
	)
	if len(rap.Differences) != 2 {
		t.Fatalf("attendu 2 ecarts, obtenu %+v", rap.Differences)
	}
	for _, d := range rap.Differences {
		attendu := SensGain // le joueur 222 gagne : un gain reste un gain
		if strings.HasSuffix(d.Metrique, "/111") {
			attendu = SensChangement // le joueur 111 perd ce que 222 gagne : reattribution
		}
		if d.Sens != attendu {
			t.Errorf("%s/%s : sens %q, attendu %q (somme 15 = 15)", d.Axe, d.Metrique, d.Sens, attendu)
		}
	}
	if bil := rap.Bilans["armes"]; bil.Pertes != 0 || bil.Gains != 1 || bil.Changements != 1 {
		t.Errorf("bilan armes : %+v", bil)
	}

	// Somme qui baisse (15 -> 13) : la baisse du joueur 111 reste une perte a instruire.
	rap = Comparer(
		Empreinte{Mesures: map[string]Mesure{kA: num(10), kB: num(5)}},
		Empreinte{Mesures: map[string]Mesure{kA: num(8), kB: num(5)}},
	)
	if len(rap.Differences) != 1 || rap.Differences[0].Sens != SensPerte {
		t.Fatalf("somme non conservee : attendu une perte, obtenu %+v", rap.Differences)
	}

	// Somme qui MONTE (15 -> 20) : un joueur perd 2, un autre gagne 7 (des ramassages sans auteur
	// ont trouve le leur) — encore une reattribution, la perte individuelle est un changement.
	rap = Comparer(
		Empreinte{Mesures: map[string]Mesure{kA: num(10), kB: num(5)}},
		Empreinte{Mesures: map[string]Mesure{kA: num(8), kB: num(12)}},
	)
	if bil := rap.Bilans["armes"]; bil.Pertes != 0 {
		t.Fatalf("somme en hausse : aucune perte attendue, obtenu %+v", rap.Differences)
	}

	// `bombStats.coverage.periodsNoBridge` : un compteur d'echec porte par un calque.
	if !estCompteurDEchec(cle("assaut", "bombStats.coverage.periodsNoBridge")) {
		t.Fatal("periodsNoBridge sous bombStats.coverage est un compteur d'echec")
	}

	// Les durees par joueur suivent la meme regle (`.../duree-totale/par-xuid/`), et les vies
	// par joueur (`tracks/vies-par-xuid/`).
	kD1, kD2 := cle("vehicules", "vehicles.rides/duree-totale/par-xuid/111"), cle("vehicules", "vehicles.rides/duree-totale/par-xuid/222")
	rap = Comparer(
		Empreinte{Mesures: map[string]Mesure{kD1: num(2711), kD2: num(0)}},
		Empreinte{Mesures: map[string]Mesure{kD1: num(687), kD2: num(2024)}},
	)
	if bil := rap.Bilans["vehicules"]; bil.Pertes != 0 || bil.Changements == 0 {
		t.Errorf("duree reattribuee : %+v", bil)
	}
	if g, ok := groupeParXUID(cle("pistes", "tracks/vies-par-xuid/111")); !ok || g != cle("pistes", "tracks/vies-par-*/") {
		t.Errorf("groupe vies-par-xuid : %q %v", g, ok)
	}

	// Un trajet ventile PAR SLOT (porteur sans nom) qui passe PAR JOUEUR (porteur nomme) : la
	// ligne par-slot « disparait », la somme des deux ventilations est conservee -> changement.
	kS := cle("vehicules", "vehicles.rides/duree-totale/par-slot/513")
	kX := cle("vehicules", "vehicles.rides/duree-totale/par-xuid/111")
	rap = Comparer(
		Empreinte{Mesures: map[string]Mesure{kS: num(2672)}},
		Empreinte{Mesures: map[string]Mesure{kX: num(2672)}},
	)
	for _, d := range rap.Differences {
		if d.Sens == SensPerte || d.Sens == SensDisparu {
			t.Errorf("slot -> joueur a somme conservee : %s/%s lu %q", d.Axe, d.Metrique, d.Sens)
		}
	}
}
