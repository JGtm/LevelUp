package replay

// player_teams_slot_ambigu_test.go — LE PONT SLOT -> INDEX SE TAIT SUR UN SLOT QUE DEUX JOUEURS
// SE SONT PARTAGE (revue de jalon M1, lentille L4).
//
// # LE DEFAUT QUE CE FICHIER EPINGLE
//
// `equipeDuSlot` lisait `IdentityRegistry.IndexParSlot()` sans consulter `SlotsAmbigus()`.
// `ownersFromLives` garde dans `Owner` le PREMIER occupant nomme d'un slot recycle et marque le
// slot ambigu sans trancher : sur un slot que deux joueurs d'EQUIPES DIFFERENTES se partagent,
// la vie du SECOND recevait donc l'equipe du PREMIER, et `coverage.teams.tracksNamed` l'affirmait
// nommee. C'est exactement le defaut qui a fait retirer le pont aplati `PontParSlot` au lot 6.1
// (rapport `.ai/V7.5/RAPPORT_PONT_APLATI_2026-09-10.md`), et `PontDeSlot` s'en abstient depuis.
// Le parc mesure le volume (9 artefacts portent `slotCollisions > 0`) ; ce fichier epingle la
// REGLE sur un slot synthetique.
//
// # MUTATION (a rejouer pour prouver que ces tests mordent)
//
// Retirer la garde `if p.slotsAmbigus[slot]` de `equipeDuSlot` : la vie du second occupant
// repart avec l'equipe du premier, `tracksNamed` remonte, et `tracksSlotAmbiguous` retombe a
// zero — les trois assertions rougissent ensemble.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

// Les deux slots du cas : 900 partage par deux joueurs d'equipes OPPOSEES, 901 tenu par un seul
// — le temoin, qui doit continuer d'etre nomme.
const (
	slotPartageL4 uint32 = 900
	slotSimpleL4  uint32 = 901
)

// registreDeuxEquipesSurUnSlot monte le registre du slot recycle : les vies nomment 111 puis 222,
// `Owner` garde le PREMIER (c'est ce que `ownersFromLives` produit) et le slot est marque AMBIGU.
func registreDeuxEquipesSurUnSlot() IdentityRegistry {
	return IdentityRegistry{own: OwnerReport{
		lives: []lifeSpan{
			{slot: slotPartageL4, from: 0, to: 1_000_000, xuid: 111},
			{slot: slotPartageL4, from: 2_000_000, to: 3_000_000, xuid: 222},
			{slot: slotSimpleL4, from: 0, to: 3_000_000, xuid: 333},
		},
		Owner:         map[uint32]int{slotPartageL4: 0, slotSimpleL4: 2},
		SlotXUID:      map[uint32]uint64{slotPartageL4: 111, slotSimpleL4: 333},
		SlotAmbiguous: map[uint32]bool{slotPartageL4: true},
	}}
}

// equipesDeReference : l'index 0 (le joueur 111) est dans l'equipe 0, l'index 1 (222) dans
// l'equipe 1, l'index 2 (333) dans l'equipe 0. C'est le desaccord d'equipe qui rend le defaut
// VISIBLE : sans lui, emprunter l'identite du premier occupant se lirait comme une lecture juste.
func equipesDeReference() map[int]int { return map[int]int{0: 0, 1: 1, 2: 0} }

// TestEquipeDuSlotSAbstientSurUnSlotAmbigu — la propriete, au niveau du pont.
func TestEquipeDuSlotSAbstientSurUnSlotAmbigu(t *testing.T) {
	p := newTeamPublication(registreDeuxEquipesSurUnSlot(), equipesDeReference(),
		filmdec.TeamScanReport{Component: "team_designator", Records: 3}, nil)

	if eq, lue, ambigu := p.equipeDuSlot(slotPartageL4); lue || !ambigu || eq != 0 {
		t.Errorf("slot partage : equipe=%d lue=%v ambigu=%v — attendu une ABSTENTION"+
			" (lue=false, ambigu=true)", eq, lue, ambigu)
	}
	if eq, lue, ambigu := p.equipeDuSlot(slotSimpleL4); !lue || ambigu || eq != 0 {
		t.Errorf("slot a occupant unique : equipe=%d lue=%v ambigu=%v — attendu l equipe 0 LUE",
			eq, lue, ambigu)
	}
	if _, lue, ambigu := p.equipeDuSlot(999); lue || ambigu {
		t.Errorf("slot inconnu du pont : lue=%v ambigu=%v — attendu les deux faux, un slot"+
			" qu on ne connait pas n est pas un slot conteste", lue, ambigu)
	}
}

// TestVieDuSecondOccupantNeRecoitPasLEquipeDuPremier — le defaut LA OU IL SE VOYAIT : sur la vie
// publiee et sur le compte de couverture.
func TestVieDuSecondOccupantNeRecoitPasLEquipeDuPremier(t *testing.T) {
	p := newTeamPublication(registreDeuxEquipesSurUnSlot(), equipesDeReference(),
		filmdec.TeamScanReport{Component: "team_designator", Records: 3}, nil)
	// Deux vies SANS xuid (le seul regime ou le pont par slot decide) : celle du slot partage,
	// celle du temoin.
	tracks := []Track{
		{Slot: slotPartageL4, Team: -1},
		{Slot: slotSimpleL4, Team: -1},
	}

	total, nommees, slotAmbigu := p.poserSurLesTraces(tracks)

	if total != 2 || nommees != 1 || slotAmbigu != 1 {
		t.Fatalf("comptes = total %d / nommees %d / slotAmbigu %d, attendu 2 / 1 / 1",
			total, nommees, slotAmbigu)
	}
	if tracks[0].Team != -1 {
		t.Errorf("la vie du slot partage porte l equipe %d : le pont a publie l equipe du"+
			" PREMIER occupant sur la vie du second", tracks[0].Team)
	}
	if tracks[1].Team != 0 {
		t.Errorf("la vie du slot a occupant unique porte l equipe %d, attendu 0 — l abstention"+
			" ne doit pas deborder sur les slots sains", tracks[1].Team)
	}
	cov := p.couverture(total, nommees, slotAmbigu, nil)
	if cov.Tracks != 2 || cov.TracksNamed != 1 || cov.TracksSlotAmbiguous != 1 {
		t.Errorf("couverture = %d vies / %d nommees / %d slot ambigu, attendu 2 / 1 / 1",
			cov.Tracks, cov.TracksNamed, cov.TracksSlotAmbiguous)
	}
}

// TestLeXUIDPRIMESURLeSlotAmbigu — L'ORDRE EST PRESERVE. Une vie NOMMEE par son xuid passe par le
// lien direct du lot 1.6 et ne descend jamais au pont : l'abstention ne doit pas lui coûter son
// equipe sous pretexte que son slot a ete recycle.
func TestLeXUIDPRIMESURLeSlotAmbigu(t *testing.T) {
	reg := registreDeuxEquipesSurUnSlot()
	reg.filmTable.table = PlayerIndexTable{ByXUID: map[uint64]int{111: 0, 222: 1, 333: 2}}
	p := newTeamPublication(reg, equipesDeReference(),
		filmdec.TeamScanReport{Component: "team_designator", Records: 3}, nil)

	tracks := []Track{{Slot: slotPartageL4, XUID: "222", Team: -1}}
	total, nommees, slotAmbigu := p.poserSurLesTraces(tracks)

	if total != 1 || nommees != 1 || slotAmbigu != 0 {
		t.Fatalf("comptes = %d / %d / %d, attendu 1 / 1 / 0", total, nommees, slotAmbigu)
	}
	if tracks[0].Team != 1 {
		t.Errorf("la vie nommee 222 porte l equipe %d, attendu 1 : le xuid decide AVANT le pont",
			tracks[0].Team)
	}
}
