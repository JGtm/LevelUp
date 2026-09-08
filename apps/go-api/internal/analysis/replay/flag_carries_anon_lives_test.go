package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// TestFlagCarriesVieAnonymeNEstPasUneAbsence — UNE VIE ANONYME EST UNE PRESENCE SANS IDENTITE
// PUBLIEE, PAS UNE ABSENCE (correctif du 2026-09-06 ; meme principe que le gate des portages de
// crane, schema 43).
//
// Depuis le schema 36 (« une track = une vie ») un slot recycle publie PLUSIEURS pistes, et le
// fil des morts n'en nomme pas toujours toutes. `attachFlagCarryPositions` n'indexait que les
// pistes NOMMEES : une prise que seule la vie ANONYME du porteur recouvre sortait `NoTrack`, et
// le portage disparaissait du calque. Mesure du corpus temoin (`bcb6d393`) : 9 prises sur 16
// perdues, toutes celles du slot 536 apres sa mort a la frame 2736 — la vie suivante du meme
// slot est publiee sans nom, alors que le PONT canonique la nomme.
func TestFlagCarriesVieAnonymeNEstPasUneAbsence(t *testing.T) {
	const porteur = "2535429985869093"
	tracks := []Track{
		flagTestTrack(536, porteur, 0, 40, 30, 40), // vie NOMMEE, terminee avant la prise
		flagTestTrack(536, "", 60, 99, 30, 40),     // vie ANONYME : elle seule couvre la prise
	}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 7000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 9000, Slot: 12, Stat: objectiveevents.StatFlagCaptures},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: porteur}),
		Spawns:   []FlagSpawn{{Team: 0, X: 0, Y: 0}, {Team: 1, X: 100, Y: 100}},
	}
	ctx := flagTestCtx(tracks, nil, 100)
	ctx.slotXUID = map[uint32]uint64{536: 2535429985869093}

	got, cov := buildFlagCarries(scan, ctx)
	if cov.NoTrack != 0 || cov.Carries != 1 {
		t.Fatalf("couverture %+v : attendu 1 portage et 0 sansPiste — la vie ANONYME du slot 536 "+
			"couvre la prise, et le pont la nomme", *cov)
	}
	f := flagOfTeam(t, got, 0)
	assertFlagStates(t, f, []string{FlagStateHome, FlagStateCarried, FlagStateHome})
	if f.Spans[1].T0 != 70 || f.Spans[1].T1 != 90 {
		t.Errorf("portage sur [%d,%d], attendu [70,90]", f.Spans[1].T0, f.Spans[1].T1)
	}
	if f.Spans[1].X != 30 || f.Spans[1].Y != 40 {
		t.Errorf("prise en (%v,%v), attendu la position de la vie anonyme (30,40)",
			f.Spans[1].X, f.Spans[1].Y)
	}
}

// TestFlagCarriesVieAnonymeSansPontResteEcartee — LA CONTRE-EPREUVE. Le correctif ci-dessus ne
// dit pas « toute piste anonyme fera l'affaire » : l'identite vient du PONT CANONIQUE
// (`ResolveSlotXUID`), jamais d'une deduction locale. Un slot que le pont ne nomme pas, ou qu'il
// nomme un AUTRE joueur, ne prete pas sa position — la prise reste comptee `NoTrack`.
func TestFlagCarriesVieAnonymeSansPontResteEcartee(t *testing.T) {
	const porteur = "2535429985869093"
	tracks := []Track{flagTestTrack(536, "", 60, 99, 30, 40)}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 7000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 9000, Slot: 12, Stat: objectiveevents.StatFlagCaptures},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: porteur}),
		Spawns:   []FlagSpawn{{Team: 0, X: 0, Y: 0}, {Team: 1, X: 100, Y: 100}},
	}
	for nom, pont := range map[string]map[uint32]uint64{
		"pont muet":              nil,
		"pont sur un autre slot": {512: 2535429985869093},
		"pont sur un autre nom":  {536: 2533274823110022},
	} {
		t.Run(nom, func(t *testing.T) {
			ctx := flagTestCtx(tracks, nil, 100)
			ctx.slotXUID = pont
			_, cov := buildFlagCarries(scan, ctx)
			if cov.NoTrack != 1 || cov.Carries != 0 {
				t.Errorf("couverture %+v : attendu 0 portage et 1 sansPiste — rien ne rattache "+
					"cette vie anonyme au porteur", *cov)
			}
		})
	}
}

// TestFlagCarriesSlotPartageRefuseLeRepli — LA GARDE DU CONSTAT C1 (revue DUREES-R1). Le pont
// canonique NE REFUSE PAS un slot que deux joueurs se partagent : `ownersFromLives` compte la
// collision puis `continue`, et le PREMIER nomme reste publie dans `SlotXUID`. Preter les vies
// ANONYMES d'un tel slot au joueur que le pont designe reviendrait a dessiner un portage a la
// position d'un autre — un resultat FAUX la ou l'ancien code rendait un resultat ABSENT.
//
// Declenchement mesure au parc : `084a804d` slot 734 — `[5872..6981]` nommee A, `[7123..7158]`
// ANONYME, `[7457..7591]` nommee B ; 9 artefacts sur 106 portent au moins un slot en collision.
func TestFlagCarriesSlotPartageRefuseLeRepli(t *testing.T) {
	const porteur, autre = "2535429985869093", "2533274823110022"
	// Le slot 536 est partage : A, puis une vie ANONYME, puis B. Le pont nomme A.
	tracks := []Track{
		flagTestTrack(536, porteur, 0, 40, 30, 40),
		flagTestTrack(536, "", 60, 99, 30, 40), // la vie ambigue : elle couvre la prise
		flagTestTrack(536, autre, 100, 120, 30, 40),
	}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 7000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 9000, Slot: 12, Stat: objectiveevents.StatFlagCaptures},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: porteur}),
		Spawns:   []FlagSpawn{{Team: 0, X: 0, Y: 0}, {Team: 1, X: 100, Y: 100}},
	}
	ctx := flagTestCtx(tracks, nil, 130)
	ctx.slotXUID = map[uint32]uint64{536: 2535429985869093}

	_, cov := buildFlagCarries(scan, ctx)
	if cov.Carries != 0 || cov.NoTrack != 1 {
		t.Errorf("couverture %+v : attendu 0 portage et 1 sansPiste — deux joueurs se partagent "+
			"le slot 536, la vie anonyme n'appartient a personne", *cov)
	}
	if cov.AmbiguousSlot != 1 {
		t.Errorf("ambiguousSlot = %d, attendu 1 — le refus doit se COMPTER, sinon un portage "+
			"manquant faute d'identite est indistinguable d'un portage qui n'a jamais eu lieu",
			cov.AmbiguousSlot)
	}
}

// TestFlagCarriesSlotNonPartageAccepteEtNeCompteRien — la CONTRE-EPREUVE de la garde ci-dessus :
// le meme slot avec UN SEUL occupant nomme prete bien sa vie anonyme, et ne compte aucun refus.
func TestFlagCarriesSlotNonPartageAccepteEtNeCompteRien(t *testing.T) {
	const porteur = "2535429985869093"
	tracks := []Track{
		flagTestTrack(536, porteur, 0, 40, 30, 40),
		flagTestTrack(536, "", 60, 99, 30, 40),
	}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 7000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 9000, Slot: 12, Stat: objectiveevents.StatFlagCaptures},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: porteur}),
		Spawns:   []FlagSpawn{{Team: 0, X: 0, Y: 0}, {Team: 1, X: 100, Y: 100}},
	}
	ctx := flagTestCtx(tracks, nil, 130)
	ctx.slotXUID = map[uint32]uint64{536: 2535429985869093}

	_, cov := buildFlagCarries(scan, ctx)
	if cov.Carries != 1 || cov.NoTrack != 0 || cov.AmbiguousSlot != 0 {
		t.Errorf("couverture %+v : attendu 1 portage, 0 sansPiste, 0 slot ambigu — un seul "+
			"occupant nomme ne contredit pas le pont", *cov)
	}
}

// TestFlagCarriesSlotEpureDuPontComptEncoreLeRefus — LA FUSION DES DEUX GARDES (integration de
// `feat/v2-vies-anonymes` et `feat/v2-durees`, 2026-09-07).
//
// Les deux revues ont pose une garde, sur deux matieres differentes, et la fusion ne doit en
// perdre AUCUNE :
//
//	VIES-R1 C2   `OwnerReport.NamingBridge()` RETIRE du pont les slots que deux vies nommees se
//	             partagent — la matiere est `own.Vies()`, les vies DECOUPEES, y compris celles que
//	             `minPoints` n'a pas publiees ;
//	DUREES-R1 C1 `slotAmbigu` juge sur les vies PUBLIEES et attrape la contradiction pont <->
//	             document, que la premiere ne voit pas.
//
// LE PIEGE DE L'INTEGRATION, ET C'EST CE QUE CE TEST FERME : un slot retire du pont epure est
// INDISTINGUABLE d'un slot que le pont n'a jamais nomme. Le repli echoue silencieusement, la
// piste est simplement ignoree, et `coverage.flagCarries.ambiguousSlot` retombe a ZERO — le
// compteur que DUREES-R1 a fait servir jusqu'au contrat cesserait de compter la moitie de sa
// population sans que rien ne le dise. `slotAmbiguous` voyage donc A COTE du pont epure.
//
// ICI les vies PUBLIEES ne se contredisent pas (un seul nom) : seule la garde amont peut
// refuser, et elle doit COMPTER.
//
// MUTATION : retirer la branche `slotAmbiguous[slot]` de `replierRefuse` rougit — 0 portage
// (le pont epure ne nomme rien) mais `ambiguousSlot = 0`, le refus devenu muet.
func TestFlagCarriesSlotEpureDuPontComptEncoreLeRefus(t *testing.T) {
	const porteur = "2535429985869093"
	// Une seule vie NOMMEE publiee : `slotAmbigu` ne voit aucune contradiction. La collision a
	// ete constatee en amont, sur une vie que `minPoints` n'a pas publiee.
	tracks := []Track{
		flagTestTrack(536, porteur, 0, 40, 30, 40),
		flagTestTrack(536, "", 60, 99, 30, 40), // la vie que le repli aurait prise
	}
	scan := FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectiveevents.NamedEvent{
			{TimeMS: 7000, Slot: 12, Stat: objectiveevents.StatFlagSteals},
			{TimeMS: 9000, Slot: 12, Stat: objectiveevents.StatFlagCaptures},
		},
		Identity: objectiveevents.FlatRoundIdentity(map[int]string{12: porteur}),
		Spawns:   []FlagSpawn{{Team: 0, X: 0, Y: 0}, {Team: 1, X: 100, Y: 100}},
	}
	ctx := flagTestCtx(tracks, nil, 130)
	// Le pont EPURE : le slot 536 en est absent, exactement ce que rend `NamingBridge()`.
	ctx.slotXUID = map[uint32]uint64{}
	ctx.slotAmbiguous = map[uint32]bool{536: true}

	_, cov := buildFlagCarries(scan, ctx)
	if cov.Carries != 0 || cov.NoTrack != 1 {
		t.Errorf("couverture %+v : attendu 0 portage et 1 sansPiste", *cov)
	}
	if cov.AmbiguousSlot != 1 {
		t.Errorf("ambiguousSlot = %d, attendu 1 — un slot RETIRE du pont epure doit rester "+
			"COMPTE : sinon il devient indistinguable d'un slot que le pont n'a jamais nomme, "+
			"et la moitie de la population du compteur disparait en silence", cov.AmbiguousSlot)
	}
}
