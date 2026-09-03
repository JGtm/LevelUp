package replay

// document_ability_impulses_test.go — LES BRANCHES LIMITES de la jointure d'identité des
// impulsions de capacité. Le golden (renderAbilityImpulses) verrouille le comportement sur
// les données réelles du film de référence, et le test d'acceptation
// (ability_impulses_film_test.go) le confronte à un relevé Theater ; ces tests-ci verrouillent
// ce que ni l'un ni l'autre n'exerce à coup sûr :
//
//   - LE REPLIEMENT (i57 et i59 co-transmis, retransmissions, gestes successifs) ;
//   - LES DEUX CONTRAINTES D'IDENTITÉ — même vie ET antériorité —, celles dont le rapport R8
//     par. 8.4 dit qu'elles déplacent la moitié des attributions quand on les oublie ;
//   - LE REFUS D'UNE FAMILLE NON MESURÉE, qui est ce qui garde le répulseur dehors.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	// aiStep : 100 ms par frame, origine 0 — la grille du rejeu.
	aiStep = uint64(100_000)
	// aiRankThruster / aiRankRepulsor : deux rangs d'une palette de test. Leurs VALEURS ne
	// veulent rien dire (elles changent d'une palette à l'autre) — c'est la table qui nomme.
	aiRankThruster = 21
	aiRankRepulsor = 6
)

// aiPalette : une palette de test qui nomme deux rangs et leur donne une famille.
func aiPalette() *AbilityPalette {
	return &AbilityPalette{
		ID:      "test",
		Markers: []int{aiRankThruster, aiRankRepulsor},
		Ranks: map[int]Label{
			aiRankThruster: {En: "Thruster", Fr: "propulseur"},
			aiRankRepulsor: {En: "Repulsor", Fr: "repulseur"},
		},
		Families: map[int]string{aiRankThruster: "thruster", aiRankRepulsor: "repulsor"},
	}
}

// aiTrack : une vie publiée couvrant tout l'axe du scénario.
func aiTrack(slot uint32) Track {
	return Track{Slot: slot, StartFrame: 0, EndFrame: 10_000,
		Points: []Point{{T: 0}, {T: 10_000}}}
}

func aiRead(slot uint32, tsUS uint64, predicted bool) filmdec.AbilityImpulse {
	return filmdec.AbilityImpulse{Slot: slot, TimestampUS: tsUS, Predicted: predicted}
}

func aiRank(slot uint32, tsUS uint64, rank int) filmdec.AbilityRank {
	return filmdec.AbilityRank{Slot: slot, TimestampUS: tsUS, Rank: rank}
}

// aiInputs assemble une entrée de test : une vie qui couvre [0, 600 s], le rang donné, et
// la seule famille `thruster` déclarée mesurée.
func aiInputs(reads []filmdec.AbilityImpulse, ranks []filmdec.AbilityRank,
	lives []lifeSpan) abilityImpulseInputs {
	if lives == nil {
		lives = []lifeSpan{{slot: 10, from: 0, to: 600_000_000}}
	}
	return abilityImpulseInputs{
		reads: reads, ranks: ranks, lives: lives,
		palette: aiPalette(), measured: []string{"thruster"},
	}
}

func TestBuildAbilityImpulses_ReplieLesLecturesCoTransmisesEnUnSeulGeste(t *testing.T) {
	reads := []filmdec.AbilityImpulse{
		aiRead(10, 10_000_000, true),  // i57
		aiRead(10, 10_000_000, false), // i59, MEME instant : le meme geste
		aiRead(10, 10_600_000, true),  // retransmission a 0,6 s : encore le meme geste
		aiRead(10, 30_000_000, false), // 19,4 s plus tard : un SECOND geste
	}
	ranks := []filmdec.AbilityRank{aiRank(10, 5_000_000, aiRankThruster)}
	out, cov := buildAbilityImpulses(aiInputs(reads, ranks, nil), []Track{aiTrack(10)}, 0, aiStep)
	if cov.Reads != 4 || cov.Episodes != 2 {
		t.Fatalf("lectures=%d gestes=%d, attendu 4 et 2 — le repliement a bouge (cov=%+v)",
			cov.Reads, cov.Episodes, cov)
	}
	if len(out) != 2 || out[0].T != 100 || out[1].T != 300 {
		t.Fatalf("impulsions %+v, attendu deux gestes aux frames 100 et 300", out)
	}
	for _, im := range out {
		if im.Family != "thruster" || im.Slot != 10 {
			t.Fatalf("impulsion %+v : famille ou slot inattendu", im)
		}
	}
}

func TestBuildAbilityImpulses_LeRangPosterieurNIdentifiePas(t *testing.T) {
	// LA MORSURE : le joueur ramasse le propulseur APRES l'impulsion. Sans la contrainte
	// d'anteriorite, l'impulsion lui serait creditee — c'est le defaut que R8 par. 8.4 mesure
	// (4 des 8 lectures de `00ba2e1c` changent de rang quand on l'oublie).
	reads := []filmdec.AbilityImpulse{aiRead(10, 10_000_000, true)}
	ranks := []filmdec.AbilityRank{aiRank(10, 12_000_000, aiRankThruster)}
	out, cov := buildAbilityImpulses(aiInputs(reads, ranks, nil), []Track{aiTrack(10)}, 0, aiStep)
	if len(out) != 0 || cov.NoIdentity != 1 {
		t.Fatalf("out=%+v cov=%+v : un rang POSTERIEUR a identifie l'impulsion", out, cov)
	}
}

func TestBuildAbilityImpulses_LeRangDeLaViePrecedenteNIdentifiePas(t *testing.T) {
	// LA SECONDE MORSURE : le slot MIGRE aux reapparitions. Le rang lu dans la vie precedente
	// du meme slot ne dit rien de ce que porte l'occupant suivant (signature vue sur
	// `11de8353`, R8 par. 8.4). Les deux vies sont ecartees de plus de deux fois la tolerance
	// de bord, pour que la premiere ne puisse pas couvrir l'instant par elle.
	lives := []lifeSpan{
		{slot: 10, from: 0, to: 20_000_000},
		{slot: 10, from: 100_000_000, to: 200_000_000},
	}
	reads := []filmdec.AbilityImpulse{aiRead(10, 150_000_000, true)}
	ranks := []filmdec.AbilityRank{aiRank(10, 10_000_000, aiRankThruster)}
	out, cov := buildAbilityImpulses(aiInputs(reads, ranks, lives), []Track{aiTrack(10)}, 0, aiStep)
	if len(out) != 0 || cov.NoIdentity != 1 {
		t.Fatalf("out=%+v cov=%+v : le rang de la vie PRECEDENTE a identifie l'impulsion", out, cov)
	}
	// Contrôle POSITIF sur la même entrée : une lecture dans LA BONNE vie identifie bien.
	ranks = append(ranks, aiRank(10, 110_000_000, aiRankThruster))
	out, cov = buildAbilityImpulses(aiInputs(reads, ranks, lives), []Track{aiTrack(10)}, 0, aiStep)
	if len(out) != 1 || cov.NoIdentity != 0 {
		t.Fatalf("out=%+v cov=%+v : le rang de la MEME vie aurait du identifier", out, cov)
	}
}

func TestBuildAbilityImpulses_UneFamilleNonMesureeEstEcarteeEtComptee(t *testing.T) {
	// LE REPULSEUR RESTE DEHORS, et il est COMPTE : le canal n'est prouve que pour les
	// familles que le titre declare (R8 par. 8.8, R9). Le publier ferait dessiner un geste que
	// le film n'enregistre pas.
	reads := []filmdec.AbilityImpulse{aiRead(10, 10_000_000, true)}
	ranks := []filmdec.AbilityRank{aiRank(10, 5_000_000, aiRankRepulsor)}
	out, cov := buildAbilityImpulses(aiInputs(reads, ranks, nil), []Track{aiTrack(10)}, 0, aiStep)
	if len(out) != 0 || cov.OtherFamily != 1 || cov.NoIdentity != 0 {
		t.Fatalf("out=%+v cov=%+v : le repulseur devait etre ecarte SOUS otherFamily", out, cov)
	}
}

func TestBuildAbilityImpulses_UnRangSansFamilleEstEcarte(t *testing.T) {
	// Un rang nomme mais SANS famille (les power-ups du manifeste) ne se joint a rien : il ne
	// se publie pas, et il ne passe pas non plus pour une absence d'identite.
	reads := []filmdec.AbilityImpulse{aiRead(10, 10_000_000, true)}
	ranks := []filmdec.AbilityRank{aiRank(10, 5_000_000, 8)}
	out, cov := buildAbilityImpulses(aiInputs(reads, ranks, nil), []Track{aiTrack(10)}, 0, aiStep)
	if len(out) != 0 || cov.OtherFamily != 1 {
		t.Fatalf("out=%+v cov=%+v : un rang sans famille devait etre ecarte", out, cov)
	}
}

// TestBuildAbilityImpulses_AttributionIndisponibleNEstPasUnAutreEquipement — LE CONSTAT H2 DE
// LA REVUE, ferme.
//
// Quand une des trois pieces de l'attribution manque, le calque ne peut RIEN attribuer. Verser
// ces gestes dans `otherFamily` annoncerait « ils viennent d'un AUTRE equipement » alors
// qu'aucune famille n'a jamais ete resolue ; les verser dans `noIdentity` affirmerait « le rang
// n'a pas ete lu dans la vie » la ou il n'y a pas de vie. Les deux se liraient comme des
// mesures. `noResolver` dit ce qui s'est reellement passe : la lecture n'a pas pu tourner.
func TestBuildAbilityImpulses_AttributionIndisponibleNEstPasUnAutreEquipement(t *testing.T) {
	reads := []filmdec.AbilityImpulse{aiRead(10, 10_000_000, true)}
	ranks := []filmdec.AbilityRank{aiRank(10, 5_000_000, aiRankThruster)}
	cas := []struct {
		nom   string
		casse func(*abilityImpulseInputs)
	}{
		{"palette non classee", func(in *abilityImpulseInputs) { in.palette = nil }},
		{"aucune famille mesuree declaree", func(in *abilityImpulseInputs) { in.measured = nil }},
		{"aucune vie decoupee", func(in *abilityImpulseInputs) { in.lives = nil }},
	}
	for _, c := range cas {
		in := aiInputs(reads, ranks, nil)
		c.casse(&in)
		out, cov := buildAbilityImpulses(in, []Track{aiTrack(10)}, 0, aiStep)
		if len(out) != 0 {
			t.Fatalf("%s : out=%+v, rien ne doit sortir", c.nom, out)
		}
		if cov.NoResolver != 1 || cov.OtherFamily != 0 || cov.NoIdentity != 0 {
			t.Fatalf("%s : cov=%+v, attendu noResolver=1 et les deux autres refus a zero",
				c.nom, cov)
		}
	}
	// CONTROLE POSITIF sur la MEME entree : les trois pieces presentes, l'attribution tourne
	// et `noResolver` reste a zero — sans quoi le compteur serait un attrape-tout.
	out, cov := buildAbilityImpulses(aiInputs(reads, ranks, nil), []Track{aiTrack(10)}, 0, aiStep)
	if len(out) != 1 || cov.NoResolver != 0 {
		t.Fatalf("controle positif : out=%+v cov=%+v", out, cov)
	}
}

// TestFoldAbilityImpulses_LaFenetreGLISSE_ElleNeSAncrePasSurLaPremiere — LE CONSTAT H3, ferme.
//
// La regle est « deux lectures du meme slot a moins d'une seconde portent le MEME geste », et
// la fenetre glisse sur la DERNIERE lecture. Une salve de retransmissions espacees de 0,9 s
// couvre 2,7 s : la regle glissante rend UN geste, une regle ancree sur la premiere lecture en
// rendrait DEUX. Les deux verdicts different, donc ce test les separe — ce que le cas
// 10,0 / 10,0 / 10,6 / 30,0 s ne faisait pas.
func TestFoldAbilityImpulses_LaFenetreGLISSE_ElleNeSAncrePasSurLaPremiere(t *testing.T) {
	eps := foldAbilityImpulses([]filmdec.AbilityImpulse{
		aiRead(10, 0, true),
		aiRead(10, 900_000, false),
		aiRead(10, 1_800_000, true),
		aiRead(10, 2_700_000, false),
	})
	if len(eps) != 1 {
		t.Fatalf("%d geste(s), attendu 1 : la fenetre s'ancre sur la premiere lecture au lieu "+
			"de glisser sur la derniere (gestes %+v)", len(eps), eps)
	}
	if eps[0].tsUS != 0 {
		t.Fatalf("le geste est date a %d us, attendu 0 : c'est la PREMIERE lecture qui le date",
			eps[0].tsUS)
	}
	// La borne, elle, separe bien : 1,1 s apres la derniere lecture ouvre un SECOND geste.
	eps = foldAbilityImpulses([]filmdec.AbilityImpulse{
		aiRead(10, 0, true), aiRead(10, 1_100_000, true),
	})
	if len(eps) != 2 {
		t.Fatalf("%d geste(s), attendu 2 : au-dela d'une seconde, c'est un autre geste", len(eps))
	}
}

// TestBuildAbilityImpulses_LaCouvertureBoucle — L'ENTONNOIR EST COMPLET : tout episode compte
// dans EXACTEMENT une case. Un refus ajoute plus tard sans compteur ferait chuter la somme, et
// « N publiees » cesserait d'etre jugeable.
func TestBuildAbilityImpulses_LaCouvertureBoucle(t *testing.T) {
	lives := []lifeSpan{
		{slot: 10, from: 0, to: 600_000_000}, {slot: 99, from: 0, to: 600_000_000},
	}
	reads := []filmdec.AbilityImpulse{
		aiRead(10, 1_000_000, true),  // avant l'origine
		aiRead(99, 20_000_000, true), // sans piste publiee
		aiRead(10, 20_000_000, true), // publiee (propulseur)
		aiRead(10, 40_000_000, true), // famille non mesuree (repulseur)
		aiRead(10, 60_000_000, true), // sans identite (aucun rang avant, dans la vie)
	}
	ranks := []filmdec.AbilityRank{
		aiRank(10, 6_000_000, aiRankThruster), aiRank(10, 30_000_000, aiRankRepulsor),
		aiRank(99, 6_000_000, aiRankThruster),
	}
	// La derniere impulsion tombe dans une SECONDE vie, ou aucun rang n'a ete lu.
	lives = append(lives, lifeSpan{slot: 10, from: 0, to: 0})
	reads[4].Slot = 11
	lives = append(lives, lifeSpan{slot: 11, from: 55_000_000, to: 70_000_000})
	out, cov := buildAbilityImpulses(aiInputs(reads, ranks, lives),
		[]Track{aiTrack(10), aiTrack(11)}, 5_000_000, aiStep)
	somme := cov.Published + cov.BeforeOrigin + cov.Unpublished +
		cov.NoIdentity + cov.OtherFamily + cov.NoResolver
	if somme != cov.Episodes {
		t.Fatalf("somme des cases = %d, episodes = %d — un refus n'est pas compte (cov=%+v)",
			somme, cov.Episodes, cov)
	}
	if cov.Published != len(out) {
		t.Fatalf("publiees=%d mais %d impulsion(s) rendues", cov.Published, len(out))
	}
	if cov.Published != 1 || cov.BeforeOrigin != 1 || cov.Unpublished != 1 ||
		cov.OtherFamily != 1 || cov.NoIdentity != 1 || cov.NoResolver != 0 {
		t.Fatalf("cov=%+v : attendu une case a 1 pour chacun des cinq sorts, noResolver a 0", cov)
	}
}

func TestBuildAbilityImpulses_EcarteAvantOrigineEtSansPiste(t *testing.T) {
	reads := []filmdec.AbilityImpulse{
		aiRead(10, 1_000_000, true),  // avant l'origine (fixee a 5 s ci-dessous)
		aiRead(99, 10_000_000, true), // slot sans trajectoire publiee
		aiRead(10, 10_000_000, true), // celle qui doit sortir
	}
	ranks := []filmdec.AbilityRank{
		aiRank(10, 6_000_000, aiRankThruster), aiRank(99, 6_000_000, aiRankThruster),
	}
	lives := []lifeSpan{
		{slot: 10, from: 0, to: 600_000_000}, {slot: 99, from: 0, to: 600_000_000},
	}
	out, cov := buildAbilityImpulses(aiInputs(reads, ranks, lives),
		[]Track{aiTrack(10)}, 5_000_000, aiStep)
	if cov.BeforeOrigin != 1 || cov.Unpublished != 1 || len(out) != 1 {
		t.Fatalf("out=%+v cov=%+v : attendu 1 avant l'origine, 1 sans piste, 1 publiee", out, cov)
	}
	if out[0].T != 50 {
		t.Fatalf("frame %d, attendu 50 (10 s − 5 s d'origine, pas de 100 ms)", out[0].T)
	}
}

func TestBuildAbilityImpulses_TemoinComposantAbsentVoyageJusquALaCouverture(t *testing.T) {
	// UN ZERO N'EST PAS L'AUTRE : un film qui ne declare NI i57 NI i59 ne se lit pas comme un
	// film ou personne ne s'est servi de son propulseur.
	in := aiInputs(nil, nil, nil)
	in.stats = filmdec.AbilityImpulseStats{Absent: true}
	out, cov := buildAbilityImpulses(in, []Track{aiTrack(10)}, 0, aiStep)
	if len(out) != 0 || !cov.ComponentAbsent {
		t.Fatalf("out=%+v cov=%+v : le temoin d'absence de composant s'est perdu", out, cov)
	}
}

func TestFoldAbilityImpulses_OrdreTotalParInstantPuisSlot(t *testing.T) {
	// L'ORDRE DU DOCUMENT EST DETERMINISTE : deux slots au meme instant sortent par slot
	// croissant, sans quoi l'artefact dependrait de l'ordre de parcours du film.
	eps := foldAbilityImpulses([]filmdec.AbilityImpulse{
		aiRead(30, 5_000_000, true), aiRead(10, 5_000_000, false), aiRead(20, 1_000_000, true),
	})
	if len(eps) != 3 {
		t.Fatalf("%d geste(s), attendu 3", len(eps))
	}
	want := []struct {
		slot uint32
		ts   uint64
	}{{20, 1_000_000}, {10, 5_000_000}, {30, 5_000_000}}
	for i, w := range want {
		if eps[i].slot != w.slot || eps[i].tsUS != w.ts {
			t.Fatalf("geste %d = %+v, attendu slot=%d ts=%d", i, eps[i], w.slot, w.ts)
		}
	}
}
