package replay

// zone_states_capturer_test.go — LE CAMP QUI POUSSE, LU ET NON DEDUIT (lot 5.6).
//
// CE QUE CES CAS TIENNENT, ET CE QU'ILS RETOURNENT. Le schema 64 deduisait le camp de l'ISSUE
// d'une rampe, donc une rampe AVORTEE n'en avait aucun. La mesure du lot 5.6 (deux films,
// 69 rampes abouties, 69 accords, 0 desaccord) a trouve le canal POUSSEUR dans le film : le
// second canal `tag 4` de chaque zone. Le premier cas ci-dessous RETOURNE
// `TestRampeAvorteeNeNommePersonne` — avec un canal elu, une rampe avortee nomme son pousseur,
// et ce n'est plus deviner puisque c'est lu.
//
// LES CAS DE REPLI RESTENT DANS `zone_states_gauge_test.go` : eux decrivent une zone SANS canal
// elu, ou le comportement du schema 64 survit a l'identique.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
)

// canalPousseur fabrique une serie de camp : une emission par instant donne.
func canalPousseur(pts ...zoneSample) []zoneSample { return pts }

// TestRampeAvorteeNommeSonPousseurQuandLeCanalEstElu — LE GAIN DU LOT, et le cas retourne.
func TestRampeAvorteeNommeSonPousseurQuandLeCanalEstElu(t *testing.T) {
	gauge := rampeDeJauge(10, 100, 0.80) // avortee : sommet sous zoneGaugeRampComplete
	ramps := findZoneRamps(7, gauge)
	if len(ramps) != 1 {
		t.Fatalf("%d rampe(s), attendu 1", len(ramps))
	}
	// Le canal de PROPRIETE nomme le camp 0 : c'est le DEFENSEUR, celui qui subit.
	owner := []zoneSample{{t: 50, v: 0}, {t: 110, v: 0}}
	// Le canal POUSSEUR nomme le camp 1 pendant la rampe : c'est l'attaquant.
	capt := canalPousseur(zoneSample{t: 105, v: 1})
	out := zoneGaugeRampsOf(ramps, owner, capt, rampsCtxTemoin())
	if len(out) != 1 || out[0].CapturingTeam == nil {
		t.Fatalf("aucun camp publie sur une rampe avortee a canal elu : %v — c'est le gain du lot", out)
	}
	if *out[0].CapturingTeam != 1 {
		t.Fatalf("camp publie = %d, attendu 1 : le pousseur est l'attaquant, pas le defenseur",
			*out[0].CapturingTeam)
	}
}

// TestLeCanalLuPrimeSurLaDeduction : quand les deux repondent, c'est la LECTURE qui sort — et le
// repli ne se declenche pas.
func TestLeCanalLuPrimeSurLaDeduction(t *testing.T) {
	ramps := findZoneRamps(7, rampeDeJauge(10, 100, 0.99))
	owner := []zoneSample{{t: 110, v: 0}} // la deduction dirait 0
	capt := canalPousseur(zoneSample{t: 105, v: 1})
	fb := fallback.NouveauCompteur()
	c := rampsCtxTemoin()
	c.fb = fb
	out := zoneGaugeRampsOf(ramps, owner, capt, c)
	if len(out) != 1 || out[0].CapturingTeam == nil || *out[0].CapturingTeam != 1 {
		t.Fatalf("camp publie = %v, attendu 1 : la lecture prime sur la deduction", out)
	}
	if n := fb.Compte(fallback.NomZoneCampDeCaptureDeduitDeLIssue); n != 0 {
		t.Fatalf("repli declenche %d fois alors que la lecture a repondu", n)
	}
}

// TestLeNeutreLuEstUneReponse : le canal elu qui nomme le NEUTRE dit « personne ne pousse ». Le
// deduire par-dessus publierait un camp que le film contredit.
func TestLeNeutreLuEstUneReponse(t *testing.T) {
	ramps := findZoneRamps(7, rampeDeJauge(10, 100, 0.99))
	owner := []zoneSample{{t: 110, v: 1}} // la deduction dirait 1
	capt := canalPousseur(zoneSample{t: 105, v: zoneNeutralOwner})
	fb := fallback.NouveauCompteur()
	c := rampsCtxTemoin()
	c.fb = fb
	out := zoneGaugeRampsOf(ramps, owner, capt, c)
	if len(out) != 1 || out[0].CapturingTeam != nil {
		t.Fatalf("camp publie = %v alors que le canal lu nomme le neutre", out)
	}
	if n := fb.Compte(fallback.NomZoneCampDeCaptureDeduitDeLIssue); n != 0 {
		t.Fatalf("repli declenche %d fois : le neutre LU n'est pas une absence de lecture", n)
	}
}

// TestCanalMuetSurLaRampeRetombeSurLeRepli : un canal elu qui n'emet rien dans `[t0, tPeak]` n'a
// pas repondu — le repli s'exerce, et il se COMPTE.
func TestCanalMuetSurLaRampeRetombeSurLeRepli(t *testing.T) {
	ramps := findZoneRamps(7, rampeDeJauge(10, 100, 0.99))
	owner := []zoneSample{{t: 110, v: 1}}
	capt := canalPousseur(zoneSample{t: 20, v: 0}) // hors de la fenetre de la rampe
	fb := fallback.NouveauCompteur()
	c := rampsCtxTemoin()
	c.fb = fb
	out := zoneGaugeRampsOf(ramps, owner, capt, c)
	if len(out) != 1 || out[0].CapturingTeam == nil || *out[0].CapturingTeam != 1 {
		t.Fatalf("camp publie = %v, attendu 1 par le repli", out)
	}
	if n := fb.Compte(fallback.NomZoneCampDeCaptureDeduitDeLIssue); n != 1 {
		t.Fatalf("repli compte %d fois, attendu 1 : un repli non compte est un repli anonyme", n)
	}
}

// TestLaDerniereValeurDeLaFenetreGagne : la rampe peut commencer avant que le pousseur soit
// pose ; c'est la valeur AU SOMMET qui designe celui qui a mene la poussee.
func TestLaDerniereValeurDeLaFenetreGagne(t *testing.T) {
	ramps := findZoneRamps(7, rampeDeJauge(10, 100, 0.99))
	capt := canalPousseur(zoneSample{t: 100, v: 0}, zoneSample{t: 108, v: 1})
	out := zoneGaugeRampsOf(ramps, nil, capt, rampsCtxTemoin())
	if len(out) != 1 || out[0].CapturingTeam == nil || *out[0].CapturingTeam != 1 {
		t.Fatalf("camp publie = %v, attendu 1 : la derniere emission de la fenetre gagne", out)
	}
}

// ----------------------------------------------------------------------------------------------
// L'ELECTION
// ----------------------------------------------------------------------------------------------

// serieTemoin fabrique la serie zoneSeries minimale qu'une election lit.
func serieTemoin(chained map[uint32][]zoneSample) zoneSeries {
	return zoneSeries{gauge: map[uint32][]zoneSample{}, owner: map[uint32][]zoneSample{},
		keys: map[uint32]uint32{}, desig: map[uint32][]zoneSample{}, ownerChained: chained}
}

// TestElectionRetientLeCanalQuiExpliqueLesRampes : le candidat dont la valeur PENDANT la rampe
// vaut le proprietaire APRES le sommet, sur deux rampes, est elu.
func TestElectionRetientLeCanalQuiExpliqueLesRampes(t *testing.T) {
	gauge := append(rampeDeJauge(10, 100, 0.99), rampeDeJauge(10, 300, 0.99)...)
	ramps := findZoneRamps(7, gauge)
	owner := []zoneSample{{t: 111, v: 1}, {t: 311, v: 0}}
	bon := []zoneSample{{t: 105, v: 1}, {t: 305, v: 0}}
	mauvais := []zoneSample{{t: 105, v: 0}, {t: 305, v: 1}}
	ser := serieTemoin(map[uint32][]zoneSample{5: owner, 6: bon, 8: mauvais})
	capt := electZoneCapturer(ser, ramps, zoneCapturerCtx{owner: owner, ownerSlot: 5, win: 20})
	if len(capt) != len(bon) || capt[0].v != bon[0].v || capt[1].v != bon[1].v {
		t.Fatalf("canal elu = %v, attendu %v : le critere est l'accord SANS desaccord", capt, bon)
	}
}

// TestElectionRefuseUnSeulAccord : un accord unique est ce que le hasard produit sur un canal a
// trois valeurs — meme seuil et meme raison que l'election du proprietaire.
func TestElectionRefuseUnSeulAccord(t *testing.T) {
	ramps := findZoneRamps(7, rampeDeJauge(10, 100, 0.99))
	owner := []zoneSample{{t: 111, v: 1}}
	ser := serieTemoin(map[uint32][]zoneSample{
		5: owner,
		6: {{t: 105, v: 1}},
	})
	if capt := electZoneCapturer(ser, ramps, zoneCapturerCtx{
		owner: owner, ownerSlot: 5, win: 20,
	}); capt != nil {
		t.Fatalf("canal elu sur UN seul accord : %v", capt)
	}
}

// TestElectionRefuseUnCanalQuiSeContredit : un seul desaccord disqualifie, meme avec des accords
// par ailleurs. Le film ne se contredit pas sur les 69 rampes mesurees ; un canal qui se
// contredit n'est pas le bon canal.
func TestElectionRefuseUnCanalQuiSeContredit(t *testing.T) {
	gauge := append(rampeDeJauge(10, 100, 0.99), rampeDeJauge(10, 300, 0.99)...)
	gauge = append(gauge, rampeDeJauge(10, 500, 0.99)...)
	ramps := findZoneRamps(7, gauge)
	owner := []zoneSample{{t: 111, v: 1}, {t: 311, v: 0}, {t: 511, v: 1}}
	tordu := []zoneSample{{t: 105, v: 1}, {t: 305, v: 0}, {t: 505, v: 0}} // le 3e se contredit
	ser := serieTemoin(map[uint32][]zoneSample{5: owner, 6: tordu})
	if capt := electZoneCapturer(ser, ramps, zoneCapturerCtx{
		owner: owner, ownerSlot: 5, win: 20,
	}); capt != nil {
		t.Fatalf("canal elu malgre un desaccord : %v", capt)
	}
}

// TestElectionEcarteLeProprietaireLuiMeme : la reference ne peut pas etre son propre candidat,
// sinon toute zone elirait son canal de propriete et le camp publie resterait celui de l'issue.
func TestElectionEcarteLeProprietaireLuiMeme(t *testing.T) {
	gauge := append(rampeDeJauge(10, 100, 0.99), rampeDeJauge(10, 300, 0.99)...)
	ramps := findZoneRamps(7, gauge)
	// Un canal de propriete qui vaut DEJA le bon camp pendant la rampe (re-securisation) :
	// il s'elirait lui-meme si la garde manquait.
	owner := []zoneSample{{t: 105, v: 1}, {t: 111, v: 1}, {t: 305, v: 1}, {t: 311, v: 1}}
	ser := serieTemoin(map[uint32][]zoneSample{5: owner})
	if capt := electZoneCapturer(ser, ramps, zoneCapturerCtx{
		owner: owner, ownerSlot: 5, win: 20,
	}); capt != nil {
		t.Fatalf("le canal de propriete s'est elu pousseur : %v", capt)
	}
}

// TestElectionEcarteUnCanalQuiNEstPasUnCamp : la garde de valeur. Sur `396cfc92`, 4 des 11
// canaux `tag 4` portent des `u32` de l'ordre de 5 x 10^8 — des identifiants, pas des equipes.
func TestElectionEcarteUnCanalQuiNEstPasUnCamp(t *testing.T) {
	gauge := append(rampeDeJauge(10, 100, 0.99), rampeDeJauge(10, 300, 0.99)...)
	ramps := findZoneRamps(7, gauge)
	owner := []zoneSample{{t: 111, v: 1}, {t: 311, v: 1}}
	// Ce canal est en accord PARFAIT sur les deux rampes... et porte aussi un identifiant.
	identifiant := []zoneSample{{t: 105, v: 1}, {t: 305, v: 1}, {t: 400, v: 540951580}}
	ser := serieTemoin(map[uint32][]zoneSample{5: owner, 6: identifiant})
	if capt := electZoneCapturer(ser, ramps, zoneCapturerCtx{
		owner: owner, ownerSlot: 5, win: 20,
	}); capt != nil {
		t.Fatalf("canal elu malgre une valeur hors plage d'equipe : %v", capt)
	}
}

// TestElectionNeLitQueLaSerieChainee : le temoin de fiabilite est OBLIGATOIRE, et c'est mesure —
// sans lui, le canal pousseur de la troisieme zone de `7344d24f` porte sept valeurs distinctes
// et n'est meme pas candidat.
func TestElectionNeLitQueLaSerieChainee(t *testing.T) {
	gauge := append(rampeDeJauge(10, 100, 0.99), rampeDeJauge(10, 300, 0.99)...)
	ramps := findZoneRamps(7, gauge)
	owner := []zoneSample{{t: 111, v: 1}, {t: 311, v: 0}}
	bon := []zoneSample{{t: 105, v: 1}, {t: 305, v: 0}}
	ser := serieTemoin(map[uint32][]zoneSample{5: owner})
	// `bon` vit dans la serie COMPLETE mais pas dans la chainee : il ne doit pas etre elu.
	ser.owner[6] = bon
	if capt := electZoneCapturer(ser, ramps, zoneCapturerCtx{
		owner: owner, ownerSlot: 5, win: 20,
	}); capt != nil {
		t.Fatalf("canal elu depuis la serie NON chainee : %v", capt)
	}
}

// TestElectionEstDeterministe : a egalite d'accords, le slot le plus petit gagne — deux cuissons
// du meme film doivent elire le meme canal.
func TestElectionEstDeterministe(t *testing.T) {
	gauge := append(rampeDeJauge(10, 100, 0.99), rampeDeJauge(10, 300, 0.99)...)
	ramps := findZoneRamps(7, gauge)
	owner := []zoneSample{{t: 111, v: 1}, {t: 311, v: 0}}
	a := []zoneSample{{t: 105, v: 1}, {t: 305, v: 0}}
	b := []zoneSample{{t: 106, v: 1}, {t: 306, v: 0}}
	ser := serieTemoin(map[uint32][]zoneSample{5: owner, 6: a, 9: b})
	for i := 0; i < 8; i++ {
		capt := electZoneCapturer(ser, ramps, zoneCapturerCtx{owner: owner, ownerSlot: 5, win: 20})
		if len(capt) != 2 || capt[0].t != 105 {
			t.Fatalf("passe %d : canal elu = %v, attendu celui du slot 6", i, capt)
		}
	}
}
