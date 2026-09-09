package replay

import "testing"

// unnamed_lives_test.go — LA DÉCISION PRODUIT DU 2026-09-07, VERROUILLÉE.
//
// « Les vies anonymes n'existent pas : une vie est un humain ou un bot, point. » Ce qui reste
// sans nom après les quatre passes de nommage est un DÉFAUT du pont, pas une catégorie de
// donnée : il se répare par l'occupation du slot DANS LE TEMPS, et le résidu se compte.

// TestUneVieNommableParLeTempsNeResteJamaisSansNom — LA MUTATION EXIGÉE.
//
// Une vie que le fil des morts ne nomme pas mais dont le slot porte une vie nommée AVANT elle
// prend l'identité de cet occupant. C'est la population majoritaire du défaut : une vie
// qu'aucune mort ne termine est, dans la vie d'un joueur, sa DERNIÈRE — celle qui court de sa
// dernière mort au coup de sifflet.
//
// MUTATION : retirer l'appel à `nameRemainingLives` (ou sa voie `occupantPrevious`) rougit —
// la piste reste sans identité et le résidu passe à 1.
func TestUneVieNommableParLeTempsNeResteJamaisSansNom(t *testing.T) {
	tracks := []Track{
		{Slot: 7, StartFrame: 0, EndFrame: 100, XUID: "111"},
		{Slot: 7, StartFrame: 200, EndFrame: 300}, // nommage non résolu : dernière vie
	}
	lives := []lifeSpan{{slot: 7, from: 0, to: 10_000_000, xuid: 111}}

	rep := nameRemainingLives(tracks, regDeTest(lives, nil, nil), 0, 100_000)
	if tracks[1].XUID != "111" {
		t.Fatalf("la vie qui suit celle de 111 sur le MÊME slot devait lui revenir, obtenu %q",
			tracks[1].XUID)
	}
	if rep.byPrevious != 1 || rep.remaining != 0 {
		t.Errorf("rapport %+v, attendu 1 par vie précédente et 0 de résidu", rep)
	}
}

// TestLeTempsTRANCHEEntreDeuxOccupantsDunMemeSlot — LA RÈGLE DE COLLISION, ET ELLE PASSE PAR LE
// TEMPS. Quand deux joueurs nommés se partagent un slot, la vie non résolue revient à celui qui
// l'occupait JUSTE AVANT — jamais au « premier », qui est ce que `SlotXUID` retient (et le
// défaut que le constat P1-7 a fait corriger ailleurs).
func TestLeTempsTRANCHEEntreDeuxOccupantsDunMemeSlot(t *testing.T) {
	tracks := []Track{{Slot: 7, StartFrame: 400, EndFrame: 500}}
	lives := []lifeSpan{
		{slot: 7, from: 0, to: 10_000_000, xuid: 111},          // premier occupant
		{slot: 7, from: 20_000_000, to: 30_000_000, xuid: 222}, // occupant suivant
	}
	// `SlotXUID` désignerait 111 (première vie nommée) : c'est exactement ce qu'on refuse.
	rep := nameRemainingLives(tracks, regDeTest(lives, map[uint32]uint64{7: 111}, nil), 0, 100_000)
	if tracks[0].XUID != "222" {
		t.Fatalf("la vie à 40 s revient à l'occupant du moment (222), obtenu %q — le pont par "+
			"slot aurait dit 111", tracks[0].XUID)
	}
	if rep.byPrevious != 1 {
		t.Errorf("rapport %+v, attendu 1 par vie précédente", rep)
	}
}

// TestUneVieAnterieureAuPremierDecesPrendLOccupantSUIVANT — la voie (b) : une vie publiée avant
// la première mort du joueur n'a aucune vie nommée qui la précède sur son slot.
func TestUneVieAnterieureAuPremierDecesPrendLOccupantSUIVANT(t *testing.T) {
	tracks := []Track{{Slot: 7, StartFrame: 0, EndFrame: 50}}
	lives := []lifeSpan{{slot: 7, from: 20_000_000, to: 30_000_000, xuid: 222}}

	rep := nameRemainingLives(tracks, regDeTest(lives, nil, nil), 0, 100_000)
	if tracks[0].XUID != "222" || rep.byNext != 1 {
		t.Errorf("piste = %q, rapport %+v ; attendu 222 par vie suivante", tracks[0].XUID, rep)
	}
}

// TestUnSlotQueSeuleUneFERMETURENommePasseParLePont — la voie (c) : un slot dont AUCUNE vie
// n'est nommée peut malgré tout être au pont, via `extendSlotXUID` (une fermeture l'a attribué).
func TestUnSlotQueSeuleUneFERMETURENommePasseParLePont(t *testing.T) {
	tracks := []Track{{Slot: 9, StartFrame: 0, EndFrame: 50}}

	rep := nameRemainingLives(tracks, regDeTest(nil, map[uint32]uint64{9: 333}, nil), 0, 100_000)
	if tracks[0].XUID != "333" || rep.byBridge != 1 {
		t.Errorf("piste = %q, rapport %+v ; attendu 333 par le pont", tracks[0].XUID, rep)
	}
}

// TestCeQuiResisteEstCOMPTE_JamaisDevine — LA CONTRE-ÉPREUVE, et c'est la garde la plus
// importante : on ne publie PAS une identité inventée. Un slot que rien ne nomme reste sans nom,
// et il entre au compteur publié (`Coverage.Bridge.UnnamedLives`) doublé d'un `slog.Error`.
func TestCeQuiResisteEstCOMPTE_JamaisDevine(t *testing.T) {
	tracks := []Track{
		{Slot: 9, StartFrame: 0, EndFrame: 50},
		{Slot: 10, StartFrame: 0, EndFrame: 50},
	}
	// Des vies nommées existent, mais sur un AUTRE slot : on ne traverse jamais la frontière.
	lives := []lifeSpan{{slot: 7, from: 0, to: 10_000_000, xuid: 111}}

	rep := nameRemainingLives(tracks, regDeTest(lives, nil, nil), 0, 100_000)
	if tracks[0].XUID != "" || tracks[1].XUID != "" {
		t.Errorf("des identités ont été INVENTÉES : %q et %q", tracks[0].XUID, tracks[1].XUID)
	}
	if rep.remaining != 2 || rep.total() != 2 {
		t.Errorf("rapport %+v, attendu 2 de résidu et rien de nommé", rep)
	}
}

// TestUneVieDeBotNEstPasUnDefautDeNommage — un bot EST une identité (il n'a pas de xuid, c'est
// tout). La passe ne le touche pas et ne le compte pas au résidu.
func TestUneVieDeBotNEstPasUnDefautDeNommage(t *testing.T) {
	tracks := []Track{{Slot: 7, StartFrame: 0, EndFrame: 50, Bot: "343 Razzle [bot]"}}
	lives := []lifeSpan{{slot: 7, from: 20_000_000, to: 30_000_000, xuid: 222}}

	rep := nameRemainingLives(tracks, regDeTest(lives, map[uint32]uint64{7: 222}, nil), 0, 100_000)
	if tracks[0].XUID != "" || tracks[0].Bot != "343 Razzle [bot]" {
		t.Errorf("la vie de bot a été réécrite : %+v", tracks[0])
	}
	if rep.total() != 0 {
		t.Errorf("rapport %+v, attendu aucune vie traitée", rep)
	}
}

// TestUneVieDejaNommeeNEstJamaisReecrite — la lecture prime sur la déduction, partout et
// toujours (même règle que `nameClosedLives`).
func TestUneVieDejaNommeeNEstJamaisReecrite(t *testing.T) {
	tracks := []Track{{Slot: 7, StartFrame: 400, EndFrame: 500, XUID: "111"}}
	lives := []lifeSpan{{slot: 7, from: 0, to: 10_000_000, xuid: 999}}

	rep := nameRemainingLives(tracks, regDeTest(lives, map[uint32]uint64{7: 999}, nil), 0, 100_000)
	if tracks[0].XUID != "111" || rep.total() != 0 {
		t.Errorf("piste = %q, rapport %+v ; la vie lue devait garder son xuid",
			tracks[0].XUID, rep)
	}
}

// TestUneFrontiereEntreDeuxOccupantsNeSeTranchePasAuHASARD — LE COMPLÉMENT DE P0-0, issu de la
// revue du lot des durées (2026-09-07).
//
// Quand la vie non résolue tombe ENTRE deux vies nommées de joueurs DIFFÉRENTS, « la précédente »
// n'est PAS une preuve : c'est un choix par l'ordre, exactement ce que `ownersFromLives` fait
// déjà (il garde le premier occupant nommé) et que ce fichier existe pour corriger.
//
// LE CAS EST MESURÉ : `084a804d` slot 734 — A `[5872..6981]`, la vie non résolue `[7123..7158]`,
// B `[7457..7591]`. Rien dans le film ne dit de quel côté de la relève elle tombe. Ce qui
// pourrait la dater serait une SUCCESSION, mais `attributeSuccessions` a déjà couru et ne date
// que les relèves de BOT.
//
// MUTATION : retirer la branche `prev != nil && next != nil && prev.xuid != next.xuid` rougit —
// la vie prend l'identité de A sans preuve.
func TestUneFrontiereEntreDeuxOccupantsNeSeTranchePasAuHASARD(t *testing.T) {
	tracks := []Track{{Slot: 734, StartFrame: 7123, EndFrame: 7158}}
	lives := []lifeSpan{
		{slot: 734, from: 587_200_000, to: 698_100_000, xuid: 2535430265968559},
		{slot: 734, from: 745_700_000, to: 759_100_000, xuid: 2535456423427614},
	}
	// Le pont garde le PREMIER occupant : il ne doit servir de repli sur AUCUN slot ambigu.
	rep := nameRemainingLives(tracks, regDeTest(lives, map[uint32]uint64{734: 2535430265968559}, map[uint32]bool{734: true}), 0, 100_000)

	if tracks[0].XUID != "" {
		t.Fatalf("la vie de la frontière a été nommée %q sans preuve — « l'occupant précédent » "+
			"est un choix par l'ordre, pas par le temps", tracks[0].XUID)
	}
	if rep.contested != 1 || rep.remaining != 1 {
		t.Errorf("rapport %+v, attendu 1 frontière indécidable comptée au résidu", rep)
	}
}

// TestUneFrontiereEntreDEUXVIESDuMemeJoueurSeTrancheBien — LA CONTRE-ÉPREUVE : le refus porte sur
// des occupants DIFFÉRENTS, pas sur le fait d'être encadré. Deux vies du même joueur encadrant la
// vie non résolue ne créent aucune ambiguïté.
func TestUneFrontiereEntreDEUXVIESDuMemeJoueurSeTrancheBien(t *testing.T) {
	tracks := []Track{{Slot: 734, StartFrame: 7123, EndFrame: 7158}}
	lives := []lifeSpan{
		{slot: 734, from: 587_200_000, to: 698_100_000, xuid: 111},
		{slot: 734, from: 745_700_000, to: 759_100_000, xuid: 111},
	}
	rep := nameRemainingLives(tracks, regDeTest(lives, nil, nil), 0, 100_000)
	if tracks[0].XUID != "111" || rep.byPrevious != 1 || rep.contested != 0 {
		t.Errorf("piste = %q, rapport %+v ; attendu 111 par la vie précédente",
			tracks[0].XUID, rep)
	}
}

// TestLePontNeSertPasDeRepliSurUnSlotAMBIGU — le second bout du même principe : `SlotXUID` garde
// le PREMIER occupant nommé d'un slot partagé. S'en servir comme repli publierait ce nom
// arbitraire sur une vie que la lecture n'a pas nommée.
func TestLePontNeSertPasDeRepliSurUnSlotAMBIGU(t *testing.T) {
	tracks := []Track{{Slot: 734, StartFrame: 10, EndFrame: 50}}

	rep := nameRemainingLives(tracks, regDeTest(nil, map[uint32]uint64{734: 111}, map[uint32]bool{734: true}), 0, 100_000)
	if tracks[0].XUID != "" || rep.remaining != 1 {
		t.Errorf("piste = %q, rapport %+v ; le pont ne doit pas servir sur un slot ambigu",
			tracks[0].XUID, rep)
	}
	// Sur un slot NON ambigu, le repli joue toujours.
	tracks = []Track{{Slot: 900, StartFrame: 10, EndFrame: 50}}
	if rep := nameRemainingLives(tracks, regDeTest(nil, map[uint32]uint64{900: 111}, nil), 0, 100_000); tracks[0].XUID != "111" || rep.byBridge != 1 {
		t.Errorf("piste = %q, rapport %+v ; attendu 111 par le pont", tracks[0].XUID, rep)
	}
}

// regDeTest fabrique un registre d'identite a partir de ses seules tables — l'entree que la
// passe de nommage final consomme. Sert aux scenarios synthetiques de ce fichier, qui n'ont ni
// positions ni fil des morts a faire traverser a `BuildIdentityRegistry`.
func regDeTest(lives []lifeSpan, slotXUID map[uint32]uint64,
	ambigus map[uint32]bool) IdentityRegistry {
	return IdentityRegistry{own: OwnerReport{
		lives: lives, SlotXUID: slotXUID, SlotAmbiguous: ambigus,
	}}
}
