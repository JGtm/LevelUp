package replaydiff

import "testing"

// Un compteur d'echec de la couverture qui BAISSE est un gain ; qui MONTE, une perte ; les
// autres mesures gardent la lecture generique (plus = mieux). Mutation : retirer l'inversion
// dans `ajouter` -> les trois premiers cas rougissent.
func TestComparer_CompteursDEchecLusAlEnvers(t *testing.T) {
	num := func(v float64) Mesure { return Mesure{EstNum: true, Num: v} }
	kUnpub := cle("couverture", "coverage.objectives.unpublished")
	kNoSlot := cle("couverture", "coverage.shots.noSlot")
	kUnnamed := cle("couverture", "coverage.bridge.unnamedLives")
	kNamed := cle("couverture", "coverage.bridge.livesNamed")
	kCaptures := cle("objectifs", "captures")
	a := Empreinte{Mesures: map[string]Mesure{
		kUnpub: num(35), kNoSlot: num(35), kUnnamed: num(0), kNamed: num(40), kCaptures: num(3),
	}}
	b := Empreinte{Mesures: map[string]Mesure{
		kUnpub:    num(0),  // echec qui baisse : GAIN
		kNoSlot:   num(50), // echec qui monte : PERTE
		kUnnamed:  num(2),  // echec qui apparait : PERTE
		kNamed:    num(38), // richesse qui baisse : PERTE (generique)
		kCaptures: num(4),  // richesse qui monte : GAIN (generique)
	}}
	rap := Comparer(a, b)
	attendu := map[string]string{
		kUnpub: SensGain, kNoSlot: SensPerte, kUnnamed: SensPerte, kNamed: SensPerte, kCaptures: SensGain,
	}
	vus := map[string]string{}
	for _, d := range rap.Differences {
		vus[cle(d.Axe, d.Metrique)] = d.Sens
	}
	for k, sens := range attendu {
		got, ok := vus[k]
		if !ok {
			t.Fatalf("%s : aucun ecart rapporte (vus : %v)", k, vus)
		}
		if got != sens {
			t.Errorf("%s : sens %q, attendu %q", k, got, sens)
		}
	}
	if bil := rap.Bilans["couverture"]; bil.Gains != 1 || bil.Pertes != 3 {
		t.Errorf("bilan couverture : gains=%d pertes=%d, attendu 1 gain / 3 pertes", bil.Gains, bil.Pertes)
	}
}

// Une voie de nommage (`coverage.bridge.namedBy*`) qui cede a une autre n'est ni un gain ni une
// perte : un CHANGEMENT. Mutation : retirer le cas `estCompteurDeMethode` dans `ajouter` -> rouge.
func TestComparer_VoiesDeNommageSontDesChangements(t *testing.T) {
	num := func(v float64) Mesure { return Mesure{EstNum: true, Num: v} }
	k := cle("couverture", "coverage.bridge.namedByNextLife")
	rap := Comparer(Empreinte{Mesures: map[string]Mesure{k: num(13)}}, Empreinte{Mesures: map[string]Mesure{k: num(8)}})
	if len(rap.Differences) != 1 || rap.Differences[0].Sens != SensChangement {
		t.Fatalf("namedByNextLife 13 -> 8 doit etre un changement, obtenu %+v", rap.Differences)
	}
	if bil := rap.Bilans["couverture"]; bil.Changements != 1 || bil.Pertes != 0 || bil.Gains != 0 {
		t.Fatalf("bilan : %+v", bil)
	}
	if !estCompteurDEchec(cle("couverture", "coverage.flagCarries.noTrack")) {
		t.Fatal("noTrack est un compteur d'echec")
	}
}

func TestEstCompteurDEchec_HorsCouvertureJamais(t *testing.T) {
	if estCompteurDEchec(cle("objectifs", "unpublished")) {
		t.Fatal("hors du chemin coverage., aucune inversion")
	}
	if estCompteurDEchec(cle("couverture", "coverage.bridge.deathOffsetRunnerUp")) {
		t.Fatal("deathOffsetRunnerUp est un nombre de voix, pas un echec")
	}
	if !estCompteurDEchec(cle("couverture", "coverage.objectives.unpublished")) {
		t.Fatal("unpublished est un compteur d'echec")
	}
}

// TestFermeturesSontDesVoiesPasDesRichesses — LE LOT E2 LES A REDUITES A ZERO, ET C'EST UN GAIN.
//
// `closedByShot` / `closedByRespawn` disent par quelle preuve une FERMETURE a comble le pont.
// Une fermeture ne s'applique qu'aux slots que la lecture n'a pas nommes : quand le lien direct
// corps -> joueur en nomme 100 %, elles tombent a zero. Sans cette regle, le gate corpus refusait
// exactement le progres du lot (mesure : 2/5/5/3/2 -> 0 sur les cinq films).
//
// MUTATION : retirer `coverage.bridge.closedBy` de `prefixesMethode` -> ROUGE.
func TestFermeturesSontDesVoiesPasDesRichesses(t *testing.T) {
	for _, c := range []string{
		"coverage.bridge.closedByShot",
		"coverage.bridge.closedByRespawn",
	} {
		if !estCompteurDeMethode(cle("couverture", c)) {
			t.Errorf("%s n'est pas lu comme une voie : sa disparition sortira en PERTE alors "+
				"qu'elle dit que la lecture a pris le pas sur la deduction", c)
		}
	}
	// `closedContested` / `closedRefused` restent des ECHECS : ils comptent ce que la fermeture
	// a REFUSE, pas la voie par laquelle elle a conclu.
	for _, c := range []string{"coverage.bridge.closedContested", "coverage.bridge.closedRefused"} {
		if estCompteurDeMethode(cle("couverture", c)) {
			t.Errorf("%s est lu comme une voie : c'est un refus, il doit rester un echec", c)
		}
		if !estCompteurDEchec(cle("couverture", c)) {
			t.Errorf("%s n'est plus lu comme un echec", c)
		}
	}
}

// TestNoBridgeEstUnCompteurDEchec : `coverage.bombCarries.noBridge` compte les portages de bombe
// qu'aucun pont ne nommait. Sa BAISSE est un gain — le meme defaut que `noTrack` la veille.
//
// MUTATION : retirer `noBridge` de `compteursDEchec` -> ROUGE.
func TestNoBridgeEstUnCompteurDEchec(t *testing.T) {
	if !estCompteurDEchec(cle("couverture", "coverage.bombCarries.noBridge")) {
		t.Error("coverage.bombCarries.noBridge n'est pas lu comme un echec : sa disparition " +
			"sortira en perte alors que plus aucun portage n'est orphelin")
	}
}
