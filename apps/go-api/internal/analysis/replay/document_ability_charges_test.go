package replay

// document_ability_charges_test.go — LES BRANCHES LIMITES de la jointure d'identité des
// charges d'équipement. Le golden (renderAbilityCharges) verrouille le comportement sur les
// données réelles du film de référence, et le test d'acceptation
// (ability_charges_film_test.go) le confronte au relevé Theater ; ces tests-ci verrouillent
// ce que ni l'un ni l'autre n'exerce à coup sûr :
//
//   - LES DEUX CONTRAINTES D'IDENTITÉ — même vie ET antériorité — celles dont le rapport R8
//     §8.4 dit qu'elles déplacent la moitié des attributions quand on les oublie (la MÊME
//     jointure `rankInLife` que les impulsions, réutilisée et non recopiée) ;
//   - LE REFUS D'UNE FAMILLE NON MESURÉE, qui est ce qui garde le répulseur dehors ;
//   - LA SOMME DE LA COUVERTURE (six cases = lectures, exactement) ;
//   - LA PUBLICATION TELLE QUELLE des lectures — jamais un repliement ni un compte
//     d'usages dérivé (piège (b) de R11).

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	// acRankGrapple / acRankThruster / acRankRepulsor : trois rangs d'une palette de test.
	// Leurs VALEURS ne veulent rien dire (elles changent d'une palette à l'autre) — c'est la
	// table qui nomme.
	acRankGrapple  = 20
	acRankThruster = 21
	acRankRepulsor = 6
)

// acPalette : une palette de test qui nomme trois rangs et leur donne une famille.
func acPalette() *AbilityPalette {
	return &AbilityPalette{
		ID:      "test",
		Markers: []int{acRankGrapple, acRankThruster, acRankRepulsor},
		Ranks: map[int]Label{
			acRankGrapple:  {En: "Grappleshot", Fr: "grappin"},
			acRankThruster: {En: "Thruster", Fr: "propulseur"},
			acRankRepulsor: {En: "Repulsor", Fr: "repulseur"},
		},
		Families: map[int]string{
			acRankGrapple: "grapple", acRankThruster: "thruster", acRankRepulsor: "repulsor",
		},
	}
}

func acRead(slot uint32, tsUS uint64, charges int) filmdec.AbilityCharge {
	return filmdec.AbilityCharge{Slot: slot, TimestampUS: tsUS, Charges: charges}
}

// acInputs assemble une entrée de test : une vie qui couvre [0, 600 s], les rangs donnés,
// et les familles `grapple` + `thruster` déclarées mesurées — celles du manifeste réel.
func acInputs(reads []filmdec.AbilityCharge, ranks []filmdec.AbilityRank,
	lives []lifeSpan) abilityChargeInputs {
	if lives == nil {
		lives = []lifeSpan{{slot: 10, from: 0, to: 600_000_000}}
	}
	return abilityChargeInputs{
		reads: reads, ranks: ranks, lives: lives,
		palette: acPalette(), measured: []string{"grapple", "thruster"},
	}
}

func TestBuildAbilityCharges_PublieLesLecturesTellesQuelles(t *testing.T) {
	// LA SÉRIE DU TÉMOIN R11 (4, 3, 2, 1, 0) sort EN CINQ LECTURES — jamais repliée, jamais
	// convertie en « 5 usages » : le compte d'usages est un dérivé que le contrat interdit
	// (une baisse peut valoir plusieurs usages, le film ne transmet pas les intermédiaires).
	reads := []filmdec.AbilityCharge{
		acRead(10, 10_000_000, 4), acRead(10, 13_000_000, 3), acRead(10, 21_000_000, 2),
		acRead(10, 23_000_000, 1), acRead(10, 33_000_000, 0),
	}
	ranks := []filmdec.AbilityRank{{Slot: 10, TimestampUS: 5_000_000, Rank: acRankThruster}}
	out, cov := buildAbilityCharges(acInputs(reads, ranks, nil), []Track{aiTrack(10)}, 0, aiStep)
	if cov.Reads != 5 || cov.Published != 5 || len(out) != 5 {
		t.Fatalf("lectures=%d publiees=%d out=%d, attendu 5/5/5 (cov=%+v)",
			cov.Reads, cov.Published, len(out), cov)
	}
	wantCharges := []int{4, 3, 2, 1, 0}
	wantT := []int{100, 130, 210, 230, 330}
	for i, c := range out {
		if c.Charges != wantCharges[i] || c.T != wantT[i] || c.Family != "thruster" || c.Slot != 10 {
			t.Fatalf("lecture %d = %+v, attendu charges=%d t=%d family=thruster slot=10",
				i, c, wantCharges[i], wantT[i])
		}
	}
	// LE ZÉRO EST UNE MESURE : la dernière lecture publie charges=0, pas une absence.
	if out[4].Charges != 0 {
		t.Fatalf("la lecture a zero charge s'est perdue : %+v", out[4])
	}
}

func TestBuildAbilityCharges_LeRangPosterieurNIdentifiePas(t *testing.T) {
	// LA MORSURE : le joueur ramasse l'équipement APRÈS la transmission de charge. Sans la
	// contrainte d'antériorité, la lecture lui serait créditée — le défaut que R8 §8.4
	// mesure, et que R11 §6 a payé sous une autre forme (rangs vieux de 65 à 162 s qui
	// nommaient « répulseur » des accroches de grappin).
	reads := []filmdec.AbilityCharge{acRead(10, 10_000_000, 3)}
	ranks := []filmdec.AbilityRank{{Slot: 10, TimestampUS: 12_000_000, Rank: acRankThruster}}
	out, cov := buildAbilityCharges(acInputs(reads, ranks, nil), []Track{aiTrack(10)}, 0, aiStep)
	if len(out) != 0 || cov.NoIdentity != 1 {
		t.Fatalf("out=%+v cov=%+v : un rang POSTERIEUR a identifie la lecture", out, cov)
	}
}

func TestBuildAbilityCharges_LeRangDeLaViePrecedenteNIdentifiePas(t *testing.T) {
	// LA SECONDE MORSURE : le slot MIGRE aux réapparitions. Le rang lu dans la vie
	// précédente du même slot ne dit rien de ce que porte l'occupant suivant.
	lives := []lifeSpan{
		{slot: 10, from: 0, to: 20_000_000},
		{slot: 10, from: 100_000_000, to: 200_000_000},
	}
	reads := []filmdec.AbilityCharge{acRead(10, 150_000_000, 2)}
	ranks := []filmdec.AbilityRank{{Slot: 10, TimestampUS: 10_000_000, Rank: acRankGrapple}}
	out, cov := buildAbilityCharges(acInputs(reads, ranks, lives), []Track{aiTrack(10)}, 0, aiStep)
	if len(out) != 0 || cov.NoIdentity != 1 {
		t.Fatalf("out=%+v cov=%+v : le rang de la vie PRECEDENTE a identifie la lecture", out, cov)
	}
	// Contrôle POSITIF sur la même entrée : une lecture dans LA BONNE vie identifie bien.
	ranks = append(ranks, filmdec.AbilityRank{Slot: 10, TimestampUS: 110_000_000, Rank: acRankGrapple})
	out, cov = buildAbilityCharges(acInputs(reads, ranks, lives), []Track{aiTrack(10)}, 0, aiStep)
	if len(out) != 1 || cov.NoIdentity != 0 || out[0].Family != "grapple" {
		t.Fatalf("out=%+v cov=%+v : le rang de la MEME vie aurait du identifier", out, cov)
	}
}

func TestBuildAbilityCharges_UneFamilleNonMesureeEstEcarteeEtComptee(t *testing.T) {
	// LE RÉPULSEUR RESTE DEHORS, et il est COMPTÉ : le canal n'est prouvé que pour les
	// familles que le titre déclare (R11 §4-5 — 218 vies de répulseur, 0 baisse). Publier
	// une lecture sous cette famille affirmerait une mesure que le film ne porte pas.
	reads := []filmdec.AbilityCharge{acRead(10, 10_000_000, 3)}
	ranks := []filmdec.AbilityRank{{Slot: 10, TimestampUS: 5_000_000, Rank: acRankRepulsor}}
	out, cov := buildAbilityCharges(acInputs(reads, ranks, nil), []Track{aiTrack(10)}, 0, aiStep)
	if len(out) != 0 || cov.OtherFamily != 1 || cov.NoIdentity != 0 {
		t.Fatalf("out=%+v cov=%+v : le repulseur devait etre ecarte SOUS otherFamily", out, cov)
	}
	// Un rang nommé mais SANS famille (les power-ups du manifeste) tombe au même compteur.
	ranks = []filmdec.AbilityRank{{Slot: 10, TimestampUS: 5_000_000, Rank: 8}}
	out, cov = buildAbilityCharges(acInputs(reads, ranks, nil), []Track{aiTrack(10)}, 0, aiStep)
	if len(out) != 0 || cov.OtherFamily != 1 {
		t.Fatalf("out=%+v cov=%+v : un rang sans famille devait etre ecarte", out, cov)
	}
}

// TestBuildAbilityCharges_AttributionIndisponibleNEstPasUnAutreEquipement — la leçon H2 de
// la revue P3, appliquée d'emblée : une indisponibilité ne se déguise jamais en mesure.
func TestBuildAbilityCharges_AttributionIndisponibleNEstPasUnAutreEquipement(t *testing.T) {
	reads := []filmdec.AbilityCharge{acRead(10, 10_000_000, 3)}
	ranks := []filmdec.AbilityRank{{Slot: 10, TimestampUS: 5_000_000, Rank: acRankThruster}}
	cas := []struct {
		nom   string
		casse func(*abilityChargeInputs)
	}{
		{"palette non classee", func(in *abilityChargeInputs) { in.palette = nil }},
		{"aucune famille mesuree declaree", func(in *abilityChargeInputs) { in.measured = nil }},
		{"aucune vie decoupee", func(in *abilityChargeInputs) { in.lives = nil }},
	}
	for _, c := range cas {
		in := acInputs(reads, ranks, nil)
		c.casse(&in)
		out, cov := buildAbilityCharges(in, []Track{aiTrack(10)}, 0, aiStep)
		if len(out) != 0 {
			t.Fatalf("%s : out=%+v, rien ne doit sortir", c.nom, out)
		}
		if cov.NoResolver != 1 || cov.OtherFamily != 0 || cov.NoIdentity != 0 {
			t.Fatalf("%s : cov=%+v, attendu noResolver=1 et les deux autres refus a zero",
				c.nom, cov)
		}
	}
	// CONTRÔLE POSITIF sur la MÊME entrée : les trois pièces présentes, l'attribution tourne
	// et `noResolver` reste à zéro — sans quoi le compteur serait un attrape-tout.
	out, cov := buildAbilityCharges(acInputs(reads, ranks, nil), []Track{aiTrack(10)}, 0, aiStep)
	if len(out) != 1 || cov.NoResolver != 0 {
		t.Fatalf("controle positif : out=%+v cov=%+v", out, cov)
	}
}

// TestBuildAbilityCharges_LaCouvertureBoucle — L'ENTONNOIR EST COMPLET : toute lecture
// compte dans EXACTEMENT une case. Un refus ajouté plus tard sans compteur ferait chuter la
// somme, et « N publiées » cesserait d'être jugeable — le patron exact de
// TestBuildAbilityImpulses_LaCouvertureBoucle.
func TestBuildAbilityCharges_LaCouvertureBoucle(t *testing.T) {
	lives := []lifeSpan{
		{slot: 10, from: 0, to: 600_000_000},
		{slot: 99, from: 0, to: 600_000_000},
		{slot: 11, from: 55_000_000, to: 70_000_000},
	}
	reads := []filmdec.AbilityCharge{
		acRead(10, 1_000_000, 3),  // avant l'origine (fixee a 5 s)
		acRead(99, 20_000_000, 3), // sans piste publiee
		acRead(10, 20_000_000, 3), // publiee (grappin)
		acRead(10, 40_000_000, 2), // famille non mesuree (repulseur ramasse entre-temps)
		acRead(11, 60_000_000, 1), // sans identite (aucun rang dans SA vie)
	}
	ranks := []filmdec.AbilityRank{
		{Slot: 10, TimestampUS: 6_000_000, Rank: acRankGrapple},
		{Slot: 10, TimestampUS: 30_000_000, Rank: acRankRepulsor},
		{Slot: 99, TimestampUS: 6_000_000, Rank: acRankGrapple},
	}
	out, cov := buildAbilityCharges(acInputs(reads, ranks, lives),
		[]Track{aiTrack(10), aiTrack(11)}, 5_000_000, aiStep)
	somme := cov.Published + cov.BeforeOrigin + cov.Unpublished +
		cov.NoIdentity + cov.OtherFamily + cov.NoResolver
	if somme != cov.Reads {
		t.Fatalf("somme des cases = %d, lectures = %d — un refus n'est pas compte (cov=%+v)",
			somme, cov.Reads, cov)
	}
	if cov.Published != len(out) {
		t.Fatalf("publiees=%d mais %d lecture(s) rendues", cov.Published, len(out))
	}
	if cov.Published != 1 || cov.BeforeOrigin != 1 || cov.Unpublished != 1 ||
		cov.OtherFamily != 1 || cov.NoIdentity != 1 || cov.NoResolver != 0 {
		t.Fatalf("cov=%+v : attendu une case a 1 pour chacun des cinq sorts, noResolver a 0", cov)
	}
}

func TestBuildAbilityCharges_TemoinComposantAbsentVoyageJusquALaCouverture(t *testing.T) {
	// UN ZÉRO N'EST PAS L'AUTRE : un film qui ne déclare pas i56 ne se lit pas comme un film
	// où personne n'use ses charges.
	in := acInputs(nil, nil, nil)
	in.stats = filmdec.AbilityChargeStats{Absent: true}
	out, cov := buildAbilityCharges(in, []Track{aiTrack(10)}, 0, aiStep)
	if len(out) != 0 || !cov.ComponentAbsent {
		t.Fatalf("out=%+v cov=%+v : le temoin d'absence de composant s'est perdu", out, cov)
	}
}

// TestBuildFromPositions_PasDeCouvertureDeChargesQuandLeBalayageNAPasTourne — la leçon H1
// de la seconde passe de revue P3, appliquée d'emblée sur le CHEMIN DE PRODUCTION : une
// couverture publiée sur un balayage qui n'a pas tourné est un mensonge. Les trois zéros
// sont séparés, comme pour les impulsions.
func TestBuildFromPositions_PasDeCouvertureDeChargesQuandLeBalayageNAPasTourne(t *testing.T) {
	base := Options{FilmClockOriginUS: 1_000_000}

	// (a) BALAYAGE EN ÉCHEC (le repli de BuildFromFilm) : AUCUNE couverture.
	doc := BuildFromPositions("m", "halo_infinite", positionsPourOrigine(), nil, base)
	if doc.Coverage == nil {
		t.Fatal("document sans couverture du tout : le scenario ne mesure plus rien")
	}
	if doc.Coverage.AbilityCharges != nil {
		t.Fatalf("couverture publiee alors que le balayage n a pas tourne : %+v",
			*doc.Coverage.AbilityCharges)
	}

	// (b) BALAYAGE ABOUTI SUR UN FILM QUI NE DÉCLARE PAS LE COMPOSANT : la couverture EST
	// publiée, et elle porte `componentAbsent`. Un zéro de balayage n'est pas l'autre.
	base.AbilityChargeStats = filmdec.AbilityChargeStats{Scanned: true, Absent: true}
	doc = BuildFromPositions("m", "halo_infinite", positionsPourOrigine(), nil, base)
	cov := doc.Coverage.AbilityCharges
	if cov == nil {
		t.Fatal("balayage abouti mais aucune couverture : un resultat de lecture s est perdu")
	}
	if !cov.ComponentAbsent || cov.Reads != 0 {
		t.Fatalf("couverture %+v : attendue vide et marquee componentAbsent", *cov)
	}

	// (c) BALAYAGE ABOUTI SUR UN FILM QUI DÉCLARE LE COMPOSANT SANS AUCUNE LECTURE ARMÉE :
	// la couverture est publiée, à zéro et SANS `componentAbsent` — le troisième zéro,
	// distinct des deux autres (c'est celui que R11 §4 mesure 485 fois sur six films).
	base.AbilityChargeStats = filmdec.AbilityChargeStats{Scanned: true, Records: 1234}
	doc = BuildFromPositions("m", "halo_infinite", positionsPourOrigine(), nil, base)
	cov = doc.Coverage.AbilityCharges
	if cov == nil || cov.ComponentAbsent || cov.Reads != 0 {
		t.Fatalf("couverture %+v : attendue publiee, a zero, sans componentAbsent", cov)
	}
}
