package filmdec

// equipment_assemble_test.go — LA FUSION strict + récupérées, testée à sec (revue
// adversariale ronde 1, F3/F4) : ordre publié, chaînage Previous, gap CALCULÉ, stats sur la
// chaîne FINALE, départage déterministe au paquet frontière, et verrou final. Ces tests ne
// dépendent d'aucun film : la fusion est pure, c'est ce qui la rend mutation-résistante.

import (
	"reflect"
	"testing"
)

// emissAt fabrique une émission stricte complète (clé de tri comprise).
func emissAt(slot uint32, ts uint64, chunk, pkt int, counter uint32, rank int) abilityEmission {
	return abilityEmission{
		Slot: slot, TimestampUS: ts, Chunk: chunk, PacketIndex: pkt, Counter: counter, Rank: rank,
	}
}

// recAt fabrique une émission récupérée avec son offset de bit.
func recAt(e abilityEmission, off int) equipRecovered {
	return equipRecovered{abilityEmission: e, off: off}
}

// recHeadAt fabrique une émission récupérée issue de la fenêtre de TÊTE d'une vie.
func recHeadAt(e abilityEmission, off int) equipRecovered {
	return equipRecovered{abilityEmission: e, off: off, head: true}
}

func TestAssembleOrdreEtChainagePrevious(t *testing.T) {
	// Le cas fondateur (Dynasty slot 535), en synthétique : taken r4, RÉCUPÉRÉE r11, spent.
	// Entrées volontairement DANS LE DÉSORDRE : l'ordre publié vient du tri, pas de l'entrée.
	strict := []abilityEmission{
		emissAt(535, 3_000, 10, 8, 7, AbilitySetNoRank), // spent
		emissAt(535, 1_000, 8, 3, 5, 4),                 // taken grappin
	}
	rec := []equipRecovered{recAt(emissAt(535, 2_000, 9, 5, 6, 11), 3055)}
	out, st := assembleEquipmentChanges(strict, rec, nil)
	if len(out) != 3 {
		t.Fatalf("%d émission(s) publiée(s), attendu 3 : %+v", len(out), out)
	}
	// ORDRE PUBLIÉ : l'ordre du film, quel que soit l'ordre d'entrée.
	if out[0].Counter != 5 || out[1].Counter != 6 || out[2].Counter != 7 {
		t.Fatalf("ordre publié c%d c%d c%d, attendu c5 c6 c7 — le tri de fusion a bougé",
			out[0].Counter, out[1].Counter, out[2].Counter)
	}
	// CHAÎNAGE Previous : la récupérée vient du grappin, le spent vient du translocateur.
	if !out[1].Recovered || out[1].Previous != 4 || out[1].Rank != 11 {
		t.Errorf("récupérée : %+v, attendu Recovered from=4 rang=11", out[1])
	}
	if out[2].Kind != EquipmentSpent || out[2].Previous != 11 {
		t.Errorf("spent : %+v, attendu Previous=11 — c'est le correctif que D1 existe pour rendre", out[2])
	}
	if st.Recovered != 1 || st.CounterJumps != 0 || st.MissedEstimate != 0 || st.Repeats != 0 {
		t.Errorf("stats : %+v, attendu 1 récupérée et une chaîne close", st)
	}
}

func TestAssembleGapCalculeEtStatsResiduelles(t *testing.T) {
	// Un saut RÉEL non comblé (c5 -> c1, pas 4) : le gap se PUBLIE et les stats le comptent.
	strict := []abilityEmission{
		emissAt(600, 1_000, 2, 1, 5, 4),
		emissAt(600, 9_000, 5, 7, 1, AbilitySetNoRank),
	}
	out, st := assembleEquipmentChanges(strict, nil, nil)
	if len(out) != 2 {
		t.Fatalf("%d émission(s), attendu 2", len(out))
	}
	if out[0].Gap != 0 || out[1].Gap != 3 {
		t.Fatalf("gaps publiés %d/%d, attendu 0/3 : le saut résiduel DOIT voyager (mutation "+
			"« gap=0 » interdite)", out[0].Gap, out[1].Gap)
	}
	if st.CounterJumps != 1 || st.MissedEstimate != 3 {
		t.Errorf("stats %+v, attendu 1 saut / 3 manquantes sur la chaîne finale", st)
	}
	// Tête hors norme + vies : les stats finales se mesurent sur la fusion, pas sur le strict.
	strict2 := []abilityEmission{emissAt(601, 1_000, 2, 1, 6, 9)}
	_, st2 := assembleEquipmentChanges(strict2, nil, nil)
	if st2.Lives != 1 || st2.LivesFirstOffSpec != 1 {
		t.Errorf("stats tête hors norme : %+v, attendu Lives=1 LivesFirstOffSpec=1", st2)
	}
}

func TestAssembleDepartageDeterministeAuPaquetFrontiere(t *testing.T) {
	// CLÉ ÉGALE (même instant, même chunk, même paquet) : la STRICTE passe avant la
	// récupérée (off -1 < off), et le résultat est LE MÊME quel que soit l'ordre d'entrée.
	before := emissAt(535, 1_000, 8, 3, 5, 4)
	after := emissAt(535, 2_000, 9, 7, 7, AbilitySetNoRank)
	borne := recAt(emissAt(535, 2_000, 9, 7, 6, 11), 3055) // même clé que `after`
	outA, stA := assembleEquipmentChanges([]abilityEmission{before, after}, []equipRecovered{borne}, nil)
	outB, stB := assembleEquipmentChanges([]abilityEmission{after, before}, []equipRecovered{borne}, nil)
	if !reflect.DeepEqual(outA, outB) || !reflect.DeepEqual(stA, stB) {
		t.Fatalf("la fusion dépend de l'ordre d'entrée :\nA=%+v (%+v)\nB=%+v (%+v)", outA, stA, outB, stB)
	}
	// Au départage « stricte d'abord », la récupérée du paquet frontière arrive à
	// contre-chaîne (c5, c7, c6) : LE VERROU FINAL la retire — jamais une répétition ni un
	// saut créé dans la sortie.
	if len(outA) != 2 || stA.Recovered != 0 {
		t.Fatalf("sortie %+v (recovered=%d) : la récupérée à contre-chaîne devait être retirée",
			outA, stA.Recovered)
	}
	if stA.Repeats != 0 || stA.CounterJumps != 1 {
		t.Errorf("stats %+v : la chaîne finale doit rester celle du strict (1 saut, 0 répétition)", stA)
	}
}

func TestVerrouFinalRetireRepetitionEtNouveauSaut(t *testing.T) {
	// RÉPÉTITION : une récupérée au compteur d'une stricte voisine est retirée.
	strict := []abilityEmission{
		emissAt(700, 1_000, 1, 1, 5, 4),
		emissAt(700, 3_000, 1, 9, 6, 9),
	}
	rec := []equipRecovered{recAt(emissAt(700, 2_000, 1, 5, 6, 11), 40)}
	out, st := assembleEquipmentChanges(strict, rec, nil)
	if len(out) != 2 || st.Recovered != 0 || st.Repeats != 0 {
		t.Fatalf("sortie %+v (stats %+v) : la récupérée en répétition devait être retirée", out, st)
	}
	// NOUVEAU SAUT : une récupérée intercalée qui éclate un saut en deux est retirée
	// (le milieu seul de {6,7,0} — la règle du trou unique, vérifiée SUR LA SORTIE).
	strict2 := []abilityEmission{
		emissAt(701, 1_000, 1, 1, 5, 4),
		emissAt(701, 9_000, 3, 9, 1, AbilitySetNoRank),
	}
	rec2 := []equipRecovered{recAt(emissAt(701, 5_000, 2, 5, 7, 9), 60)}
	out2, st2 := assembleEquipmentChanges(strict2, rec2, nil)
	if len(out2) != 2 || st2.Recovered != 0 || st2.CounterJumps != 1 {
		t.Fatalf("sortie %+v (stats %+v) : la récupérée qui éclate le saut devait être retirée",
			out2, st2)
	}
	// Une récupérée SAINE, elle, survit au verrou : son retrait aggraverait la chaîne.
	rec3 := []equipRecovered{recAt(emissAt(701, 5_000, 2, 5, 6, 9), 60)}
	out3, st3 := assembleEquipmentChanges(strict2, rec3, nil)
	if len(out3) != 3 || st3.Recovered != 1 || st3.CounterJumps != 1 {
		t.Fatalf("sortie %+v (stats %+v) : une récupérée saine ne doit pas être retirée "+
			"(le trou résiduel c6 -> c1 reste UN saut)", out3, st3)
	}
	if out3[2].Gap != 2 {
		t.Errorf("gap du spent après récupération partielle = %d, attendu 2", out3[2].Gap)
	}
}

// TestVerrouTeteConserveUneRecuperationPartielle rejoue LA SONDE de la revue ronde 2 (défaut
// P2 « verrou tête partielle », corrigé en P1bis) : une vie dont la première stricte porte c7
// ouvre une fenêtre de tête [compteur virtuel 4 -> 7] qui prédit c5 et c6 ; seul c5 est
// retrouvé. Le trou résiduel est le MÊME avec ou sans elle (4 -> 7 vaut un saut, 4 -> 5 -> 7
// aussi) : le verrou doit la GARDER. Avant le correctif, il la comparait à une chaîne sans
// amorce et la retirait — sortie de 2 émissions, Recovered=0.
func TestVerrouTeteConserveUneRecuperationPartielle(t *testing.T) {
	strict := []abilityEmission{
		emissAt(800, 1_000, 1, 3, 7, 4),                // première stricte : c7, hors norme
		emissAt(800, 3_000, 1, 9, 0, AbilitySetNoRank), // spent, chaîne close derrière
	}
	rec := []equipRecovered{recHeadAt(emissAt(800, 500, 1, 1, 5, 11), 120)}
	out, st := assembleEquipmentChanges(strict, rec, nil)
	if len(out) != 3 || st.Recovered != 1 {
		t.Fatalf("sortie %+v (recovered=%d) : la récupérée de tête PARTIELLE devait être gardée "+
			"— le trou résiduel est le même avec ou sans elle", out, st.Recovered)
	}
	if !out[0].Recovered || out[0].Counter != 5 {
		t.Fatalf("première émission publiée %+v, attendu la récupérée c5", out[0])
	}
	// La chaîne finale se mesure sur ce qui est publié : c5 -> c7 laisse un saut d'une
	// émission, et c'est ce que le document doit dire.
	if st.CounterJumps != 1 || st.MissedEstimate != 1 || out[1].Gap != 1 {
		t.Errorf("stats %+v / gap %d : attendu 1 saut résiduel d'une émission", st, out[1].Gap)
	}
}

// TestVerrouTeteBoutEnBoutParLaFenetre est le test DE BOUT EN BOUT du correctif (revue P1bis
// ronde 1, G1) : la fenêtre de tête passe par `acceptEquipRecovery` — le SEUL endroit qui pose
// `head` sur un candidat — puis par la fusion. Les deux tests voisins fabriquent `head:true` à
// la main ; celui-ci ne le fabrique pas, si bien que neutraliser la propagation
// (`c.head = w.head`, equipment_recovery.go) le fait tomber. Sans lui, `head` pouvait
// disparaître du chemin réel sans qu'une seule assertion bouge.
func TestVerrouTeteBoutEnBoutParLaFenetre(t *testing.T) {
	// Une vie dont la première stricte porte c7 : buildEquipRecoveryWindows ouvre une fenêtre
	// de TÊTE [compteur virtuel 4 -> 7], qui prédit c5 et c6. Seul c5 existe dans le film.
	w := equipRecoveryWindow{
		slot: 800, fromC: equipRecoveryHeadCounter, toC: 7,
		miss: counterStep(equipmentFirstCounter, 7), tsMax: 10_000, head: true,
	}
	w.cands = []equipRecovered{recCand(800, 500, 5, 11, 120)}
	kept := acceptEquipRecovery(&w)
	if len(kept) != 1 || !kept[0].head {
		t.Fatalf("le témoin de compteur devait accepter c5 ET le marquer « tête » : %+v", kept)
	}
	strict := []abilityEmission{
		emissAt(800, 1_000, 1, 3, 7, 4),
		emissAt(800, 3_000, 1, 9, 0, AbilitySetNoRank),
	}
	out, st := assembleEquipmentChanges(strict, kept, nil)
	if len(out) != 3 || st.Recovered != 1 {
		t.Fatalf("sortie %+v (recovered=%d) : la chaîne fenêtre -> témoin -> fusion doit GARDER "+
			"la récupération de tête partielle", out, st.Recovered)
	}
	if !out[0].Recovered || out[0].Counter != 5 || out[0].Rank != 11 {
		t.Errorf("première émission publiée %+v, attendu la récupérée c5 rang 11", out[0])
	}
}

// TestVerrouTeteRetireUneRecupereeQuiViole est le CONTRE-CAS, et il existe pour que le
// correctif ne se lise pas « les récupérées de tête sont intouchables » : le verrou continue
// de mordre sur une récupérée de tête qui, elle, aggrave RÉELLEMENT la chaîne (ici une
// répétition de la première stricte). Une amorce ne relâche aucune garde.
//
// CE QU'IL PROUVE, ET CE QU'IL NE PROUVE PAS (revue P1bis ronde 1, G1). L'entrée est
// DÉLIBÉRÉMENT hors de ce qu'`acceptEquipRecovery` sait produire : pour la fenêtre de tête
// [4 -> 7], les compteurs prédits sont {5, 6}, jamais c7. Le verrou final est une DÉFENSE EN
// PROFONDEUR — il juge la chaîne assemblée, d'où qu'elle vienne (un départage de paquet
// frontière peut intercaler ce que la fenêtre n'aurait pas accepté) —, et c'est cette
// morsure-là que le test fige. La morsure sur une entrée atteignable de bout en bout est
// couverte, elle, par TestAssembleDepartageDeterministeAuPaquetFrontiere.
func TestVerrouTeteRetireUneRecupereeQuiViole(t *testing.T) {
	strict := []abilityEmission{
		emissAt(801, 1_000, 1, 3, 7, 4),
		emissAt(801, 3_000, 1, 9, 0, AbilitySetNoRank),
	}
	rec := []equipRecovered{recHeadAt(emissAt(801, 500, 1, 1, 7, 11), 120)} // c7 : une répétition
	out, st := assembleEquipmentChanges(strict, rec, nil)
	if len(out) != 2 || st.Recovered != 0 || st.Repeats != 0 {
		t.Fatalf("sortie %+v (recovered=%d, répétitions=%d) : la récupérée de tête en RÉPÉTITION "+
			"devait être retirée", out, st.Recovered, st.Repeats)
	}
}
